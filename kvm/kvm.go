package kvm

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/allape/gogger"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/crypto/des"
	"github.com/allape/openkvm/kvm/codec"
	"github.com/allape/openkvm/kvm/keymouse"
	"github.com/allape/openkvm/kvm/video"
	"io"
	"sync"
	"time"
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

var (
	HandshakeFailed     = errors.New("handshake failed")
	AuthFailed          = errors.New("auth failed")
	UnsupportedAuthType = errors.New("unsupported auth type")
)

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

func (s *Server) handshake(client *Client) (ok bool, err error) {
	_, err = client.Write([]byte(Version))
	if err != nil {
		return false, err
	}

	resp := make([]byte, len(Version))
	err = client.Read(resp)
	if err != nil {
		return false, err
	}

	if !bytes.Equal(resp, []byte(Version)) {
		return false, client.Close("Unsupported protocol version")
	}

	if s.Options.Config.VNC.Password == "" {
		_, err = client.Write([]byte{NumberOfSecurityTypes, byte(None)})
	} else {
		_, err = client.Write([]byte{NumberOfSecurityTypes, byte(VNCAuthentication)})
	}

	return true, nil
}

func (s *Server) challenge(client *Client) (err error) {
	authType := make([]byte, 1)

	err = client.Read(authType)
	if err != nil {
		return err
	}

	switch SecurityType(authType[0]) {
	case None:
		//err = client.Close("Unsupported auth type")
		//if err != nil {
		//	return err
		//}
		//return UnsupportedAuthType
		return nil
	case VNCAuthentication:
		if s.Options.Config.VNC.Password == "" {
			err = client.Close("Internal Server Error")
			if err != nil {
				return err
			}
			return UnsupportedAuthType
		}

		client.challenge = make([]byte, ChallengeSize)
		n, err := rand.Read(client.challenge)
		if err != nil {
			return err
		} else if n != ChallengeSize {
			return io.ErrShortWrite
		}

		_, err = client.Write(client.challenge)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) auth(client *Client) (ok bool, err error) {
	if client.challenge == nil {
		_, err = client.Write(SecurityResultOK[:])
		if err != nil {
			return false, err
		}
		return true, nil
	}

	d := des.New([]byte(s.Options.Config.VNC.Password))
	expectedChallenged := make([]byte, ChallengeSize)
	err = d.Encrypt(expectedChallenged, client.challenge)
	if err != nil {
		return false, err
	}

	clientChallenged := make([]byte, ChallengeSize)
	err = client.Read(clientChallenged)
	if err != nil {
		return false, err
	}

	if bytes.Compare(expectedChallenged, clientChallenged) != 0 {
		_, err = client.Write(SecurityResultFail[:])
		if err != nil {
			return false, err
		}
		return false, client.Close("Password is incorrect")
	}

	_, err = client.Write(SecurityResultOK[:])
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *Server) init(client *Client) (err error) {
	shareFlag := make([]byte, 1)
	err = client.Read(shareFlag)
	if err != nil {
		return err
	}

	// TODO share flag

	msg, err := s.GetServerInitBytes()
	if err != nil {
		return err
	}

	_, err = client.Write(msg)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) HandleClient(client *Client) error {
	ok, err := s.handshake(client)
	if err != nil {
		return err
	} else if !ok {
		return HandshakeFailed
	}

	err = s.challenge(client)
	if err != nil {
		return err
	}

	authed, err := s.auth(client)
	if err != nil {
		return err
	} else if !authed {
		return AuthFailed
	}

	err = s.init(client)
	if err != nil {
		return err
	}

	shouldSendFullFrame := false

	msgType := make([]byte, 1)
	for {
		err = client.Read(msgType)
		if err != nil {
			return err
		}
		switch msgType[0] {
		case 0: // SetPixelFormat
		// 0000 0000 2018 0001 00ff 00ff 00ff 1008 0000 0000
		case 2: // SetEncodings
		case 3: // FramebufferUpdateRequest
			s.locker.Lock()
			err := func() error {
				defer s.locker.Unlock()
				rects, err := s.Video.GetNextImageRects(s.Options.Config.Video.SliceCount, shouldSendFullFrame)
				if err != nil {
					l.Error().Println("GetNextImageRects error:", err)
					//continue
				}
				frame, err := s.VideoCodec.FramebufferUpdate(rects)
				if err != nil {
					l.Error().Println("FramebufferUpdate error:", err)
					//continue
				}

				if len(rects) > 0 {
					shouldSendFullFrame = false
				}

				_, err = client.Write(frame)
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
			keyEvent := make([]byte, 8)
			err = client.Read(keyEvent)
			if err != nil {
				l.Error().Println("Read KeyEvent error:", err)
				continue
			}
			err := s.Keyboard.SendKeyEvent(keyEvent)
			if err != nil {
				l.Error().Println("SendKeyEvent error:", err)
			}
		case 5: // PointerEvent
			if s.Mouse == nil {
				l.Warn().Println("Mouse driver is not available")
				continue
			}

			pointerEvent := make([]byte, 6)
			err = client.Read(pointerEvent)
			if err != nil {
				l.Error().Println("Read PointerEvent error:", err)
				continue
			}

			//               +--------------+--------------+--------------+
			//              | No. of bytes | Type [Value] | Description  |
			//              +--------------+--------------+--------------+
			//              | 1            | U8 [5]       | message-type |
			//              | 1            | U8           | button-mask  |
			//              | 2            | U16          | x-position   |
			//              | 2            | U16          | y-position   |
			//              +--------------+--------------+--------------+
			oldX := binary.BigEndian.Uint16(pointerEvent[2:4])
			oldY := binary.BigEndian.Uint16(pointerEvent[4:6])
			x := uint16(float64(oldX) * s.Options.Config.Mouse.CursorXScale)
			y := uint16(float64(oldY) * s.Options.Config.Mouse.CursorYScale)
			//l.Verbose().Printf("%s Rescale PointerEvent from (%d, %d) to (%d, %d)\n", oldX, oldY, x, y)
			copy(pointerEvent[2:6], []byte{byte(x >> 8), byte(x), byte(y >> 8), byte(y)})

			err := s.Mouse.SendPointerEvent(pointerEvent)
			if err != nil {
				l.Error().Println("SendPointerEvent error:", err)
			}
		case 6: // ClientCutText
		default:
			l.Warn().Println("Unsupported message type:", hex.EncodeToString(msgType))
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

	return s.serverInitBytes, nil
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

type Client struct {
	locker    sync.Locker
	buffer    []byte
	leftover  []byte
	challenge []byte
	Messager  io.ReadWriteCloser
}

func (c *Client) Write(msg []byte) (int, error) {
	return c.Messager.Write(msg)
}

func (c *Client) Close(reason string) error {
	if reason != "" {
		length := len(reason)
		bs := make([]byte, 4)
		binary.BigEndian.PutUint32(bs, uint32(length))

		// ignore error, close client anyway
		_, _ = c.Messager.Write(append(bs, []byte(reason)...))
	}
	return c.Messager.Close()
}

func (c *Client) Read(dst []byte) error {
	c.locker.Lock()
	defer c.locker.Unlock()

	length := len(dst)

	if len(c.leftover) >= length {
		copy(dst, c.leftover[:length])
		c.leftover = c.leftover[length:]
		return nil
	}

	errCh := make(chan error, 1)

	buf := c.leftover
	go func() {
		for {
			n, err := c.Messager.Read(c.buffer)
			if err != nil {
				go func() {
					errCh <- err
				}()
				return
			}
			buf = append(buf, c.buffer[:n]...)
			if len(buf) >= length {
				c.leftover = buf[length:]
				errCh <- nil
				return
			}
		}
	}()

	select {
	case <-time.After(30 * time.Second):
		_ = c.Close("Read timeout")
		return io.ErrNoProgress
	case err := <-errCh:
		close(errCh)

		if err == nil {
			copy(dst, buf)
			return nil
		}

		_ = c.Close("")
		return err
	}
}

func NewClient(message io.ReadWriteCloser) *Client {
	return &Client{
		locker:   &sync.Mutex{},
		buffer:   make([]byte, 1024),
		Messager: message,
	}
}
