package kvm

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"github.com/allape/gogger"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/crypto/des"
	"github.com/allape/openkvm/kvm/codec"
	"github.com/allape/openkvm/kvm/keymouse"
	"github.com/allape/openkvm/kvm/video"
	"io"
	"slices"
	"sync"
)

const (
	Version               = "RFB 003.008\n"
	ChallengeSize         = des.BlockSize * 2
	NumberOfSecurityTypes = 1
)

type SecurityResult [4]byte

var (
	SecurityResultOK   SecurityResult = [4]byte{0, 0, 0, 0}
	SecurityResultFail SecurityResult = [4]byte{0, 0, 0, 1}
)

type SecurityType byte

const (
	None              SecurityType = 1
	VNCAuthentication SecurityType = 2
)

var l = gogger.New("kvm")

type PixelFormat struct {
	BitsPerPixel uint8
	Depth        uint8
	BigEndian    uint8
	TrueColor    uint8
	RedMax       uint16
	GreenMax     uint16
	BlueMax      uint16
	RedShift     uint8
	GreenShift   uint8
	BlueShift    uint8
}

type ServerInit struct {
	Name        string
	Width       uint16
	Height      uint16
	PixelFormat PixelFormat
}

type Options struct {
	Config config.Config
}

type Server struct {
	Keyboard   keymouse.Driver
	Video      video.Driver
	Mouse      keymouse.Driver
	VideoCodec codec.Codec

	Options Options

	serverInitBytes []byte
	locker          sync.Locker
}

func (s *Server) CloseClient(client Client, message string) error {
	if message != "" {
		length := len(message)
		bs := make([]byte, 4)
		binary.BigEndian.PutUint32(bs, uint32(length))

		// ignore error, close client anyway
		_, _ = client.Write(append(bs, []byte(message)...))
	}
	return client.Close()
}

func (s *Server) HandleClient(client Client) error {
	password := s.Options.Config.VNC.Password
	noPassword := password == ""

	handshake := false
	challenged := noPassword
	authed := false
	init := false

	var challenge []byte

	// send full frame without diff process for first request
	full := true

	// this buffer is bigger enough for VNC
	buf := make([]byte, 1024)

	_, err := client.Write([]byte(Version))
	if err != nil {
		return err
	}

	for {
		n, err := client.Read(buf)
		if err != nil {
			return err
		}

		msg := buf[:n]

		if !handshake {
			if slices.Equal(msg, []byte(Version)) {
				handshake = true
			} else {
				return s.CloseClient(client, "Unsupported protocol version")
			}
			if noPassword {
				_, err = client.Write([]byte{NumberOfSecurityTypes, byte(None)})
			} else {
				_, err = client.Write([]byte{NumberOfSecurityTypes, byte(VNCAuthentication)})
			}
			if err != nil {
				return err
			}
			continue
		}

		if !challenged {
			challenge = make([]byte, ChallengeSize)
			n, err := rand.Read(challenge)
			if err != nil {
				return err
			} else if n != ChallengeSize {
				return io.ErrShortWrite
			}

			_, err = client.Write(challenge)
			if err != nil {
				return err
			}
			challenged = true
			continue
		}

		if !authed {
			if !noPassword {
				d := des.New([]byte(password))
				expectedChallenged := make([]byte, ChallengeSize)
				err = d.Encrypt(expectedChallenged, challenge)
				if err != nil {
					return err
				}
				if bytes.Compare(expectedChallenged, msg[:ChallengeSize]) != 0 {
					_, err = client.Write(SecurityResultFail[:])
					if err != nil {
						return err
					}
					return s.CloseClient(client, "Password is incorrect")
				}
			}

			_, err = client.Write(SecurityResultOK[:])
			if err != nil {
				return err
			}
			authed = true
			continue
		}

		if !init {
			if len(msg) == 1 {
				msg, err := s.GetServerInitBytes()
				if err != nil {
					return err
				}
				_, err = client.Write(msg)
				if err != nil {
					return err
				}
			} else {
				return s.CloseClient(client, "This kind of `ClientInit` is not supported")
			}
			init = true
			continue
		}

		//l.Verbose().Println("msg:", hex.EncodeToString(msg))

		switch msg[0] {
		case 0: // SetPixelFormat
		// 0000 0000 2018 0001 00ff 00ff 00ff 1008 0000 0000
		case 2: // SetEncodings
		case 3: // FramebufferUpdateRequest
			s.locker.Lock()
			err := func() error {
				defer s.locker.Unlock()
				rects, err := s.Video.GetNextImageRects(s.Options.Config.Video.SliceCount, full)
				if err != nil {
					l.Error().Println("GetNextImageRects error:", err)
					//continue
				}
				msg, err = s.VideoCodec.FramebufferUpdate(rects)
				if err != nil {
					l.Error().Println("FramebufferUpdate error:", err)
					//continue
				}

				if len(rects) > 0 {
					full = false
				}

				_, err = client.Write(msg)
				if err != nil {
					return err
				}

				return nil
			}()
			if err != nil {
				return err
			}
		case 4: // KeyEvent
			if s.Keyboard == nil {
				l.Warn().Println("Keyboard driver is not available")
				continue
			}
			err := s.Keyboard.SendKeyEvent(msg)
			if err != nil {
				l.Error().Println("SendKeyEvent error:", err)
			}
		case 5: // PointerEvent
			if s.Mouse == nil {
				l.Warn().Println("Mouse driver is not available")
				continue
			}

			// apply the scale
			if len(msg) == 6 {
				//               +--------------+--------------+--------------+
				//              | No. of bytes | Type [Value] | Description  |
				//              +--------------+--------------+--------------+
				//              | 1            | U8 [5]       | message-type |
				//              | 1            | U8           | button-mask  |
				//              | 2            | U16          | x-position   |
				//              | 2            | U16          | y-position   |
				//              +--------------+--------------+--------------+
				oldX := binary.BigEndian.Uint16(msg[2:4])
				oldY := binary.BigEndian.Uint16(msg[4:6])
				x := uint16(float64(oldX) * s.Options.Config.Mouse.CursorXScale)
				y := uint16(float64(oldY) * s.Options.Config.Mouse.CursorYScale)
				//l.Verbose().Printf("%s Rescale PointerEvent from (%d, %d) to (%d, %d)\n", oldX, oldY, x, y)
				copy(msg[2:6], []byte{byte(x >> 8), byte(x), byte(y >> 8), byte(y)})
			}

			err := s.Mouse.SendPointerEvent(msg)
			if err != nil {
				l.Error().Println("SendPointerEvent error:", err)
			}
		case 6: // ClientCutText
		default:
			l.Warn().Println("Unsupported message type:", hex.EncodeToString(msg))
		}
	}
}

func (s *Server) GetServerInit() (*ServerInit, error) {
	size, err := s.Video.GetSize()
	if err != nil {
		return nil, err
	}

	return &ServerInit{
		Name:   "OpenKVM",
		Width:  uint16(size.X),
		Height: uint16(size.Y),
		PixelFormat: PixelFormat{
			BitsPerPixel: 32,
			Depth:        24,
			BigEndian:    0,
			TrueColor:    1,
			RedMax:       0xff,
			GreenMax:     0xff,
			BlueMax:      0xff,
			RedShift:     16,
			GreenShift:   8,
			BlueShift:    0,
		},
	}, nil
}

func (s *Server) GetServerInitBytes() ([]byte, error) {
	if len(s.serverInitBytes) > 0 {
		return s.serverInitBytes, nil
	}

	si, err := s.GetServerInit()
	if err != nil {
		return nil, err
	}
	pf := si.PixelFormat

	nameBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(nameBytes, uint32(len(si.Name)))
	nameBytes = append(nameBytes, []byte(si.Name)...)

	msg := append([]byte{
		byte(si.Width >> 8), byte(si.Width),
		byte(si.Height >> 8), byte(si.Height),
		pf.Depth,
		pf.BitsPerPixel,
		pf.BigEndian,
		pf.TrueColor,
		byte(pf.RedMax >> 8), byte(pf.RedMax),
		byte(pf.GreenMax >> 8), byte(pf.GreenMax),
		byte(pf.BlueMax >> 8), byte(pf.BlueMax),
		pf.RedShift,
		pf.GreenShift,
		pf.BlueShift,
		// padding
		0x00, 0x00, 0x00,
	}, nameBytes...)

	s.serverInitBytes = msg

	return msg, nil
}

func New(
	k keymouse.Driver,
	v video.Driver,
	m keymouse.Driver,
	videoCodec codec.Codec,
	options Options,
) (*Server, error) {
	s := &Server{
		Options: options,

		Keyboard:   k,
		Video:      v,
		Mouse:      m,
		VideoCodec: videoCodec,

		locker: &sync.Mutex{},
	}

	return s, nil
}

type Client interface {
	io.ReadWriteCloser
}
