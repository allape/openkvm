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

var l = gogger.New("kvm")

const (
	Version       = "RFB 003.008\n"
	ChallengeSize = des.BlockSize * 2
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
	Plain             SecurityType = 0 // 256
)

type ClientMessageType byte

const (
	SetPixelFormat           ClientMessageType = 0
	Placeholder              ClientMessageType = 1
	SetEncodings             ClientMessageType = 2
	FramebufferUpdateRequest ClientMessageType = 3
	KeyEvent                 ClientMessageType = 4
	PointerEvent             ClientMessageType = 5
	ClientCutText            ClientMessageType = 6
)

var (
	InternalServerError = errors.New("internal server error")

	HandshakeFailed     = errors.New("handshake failed")
	AuthFailed          = errors.New("auth failed")
	UnsupportedAuthType = errors.New("unsupported auth type")

	KeyboardNotAvailable = errors.New("keyboard driver is not available")
	MouseNotAvailable    = errors.New("mouse driver is not available")
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
		for i := 0; i < 9; i++ {
			l.Warn().Println("No password set, use None auth type")
		}
		_, err = client.Write([]byte{1, byte(None)})
		if err != nil {
			return false, err
		}
		return true, nil
	}

	if s.Options.Config.VNC.Username != "" {
		l.Info().Printf("Use Tight security type with username: %s; And use std VNC as fallback auth", s.Options.Config.VNC.Username)
		_, err = client.Write([]byte{2, byte(Plain), byte(VNCAuthentication)})
		if err != nil {
			return false, err
		}
		return true, nil
	}

	l.Info().Println("Use std VNC auth type")
	_, err = client.Write([]byte{1, byte(VNCAuthentication)})
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *Server) challenge(client *Client) (err error) {
	st := make([]byte, 1)

	err = client.Read(st)
	if err != nil {
		return err
	}

	client.respSecurityType = SecurityType(st[0])

	switch SecurityType(st[0]) {
	case None:
		//err = client.Close("Unsupported auth type")
		//if err != nil {
		//	return err
		//}
		//return UnsupportedAuthType
		return nil
	case VNCAuthentication:
		if s.Options.Config.VNC.Password == "" {
			_ = client.Close(InternalServerError.Error())
			return UnsupportedAuthType
		}

		client.challenge = make([]byte, ChallengeSize)
		n, err := rand.Read(client.challenge)
		if err != nil {
			_ = client.Close(InternalServerError.Error())
			return err
		} else if n != ChallengeSize {
			_ = client.Close(InternalServerError.Error())
			return io.ErrShortWrite
		}

		_, err = client.Write(client.challenge)
		if err != nil {
			return client.Close(InternalServerError.Error())
		}

		return nil
	case Plain:
		return nil
	default:
		return client.Close("Unsupported auth type")
	}
}

func (s *Server) auth(client *Client) (ok bool, err error) {
	switch client.respSecurityType {
	case None:
		_, err = client.Write(SecurityResultOK[:])
		if err != nil {
			return false, err
		}
		return true, nil
	case VNCAuthentication:
		if client.challenge == nil {
			return false, client.Close(InternalServerError.Error())
		}

		d := des.New([]byte(s.Options.Config.VNC.Password))
		expectedChallenged := make([]byte, ChallengeSize)
		err = d.Encrypt(expectedChallenged, client.challenge)
		if err != nil {
			_ = client.Close(InternalServerError.Error())
			return false, err
		}

		clientChallenged := make([]byte, ChallengeSize)
		err = client.Read(clientChallenged)
		if err != nil {
			_ = client.Close(InternalServerError.Error())
			return false, err
		}

		if bytes.Compare(expectedChallenged, clientChallenged) != 0 {
			_, _ = client.Write(SecurityResultFail[:])
			return false, client.Close("Password is incorrect")
		}

		_, err = client.Write(SecurityResultOK[:])
		if err != nil {
			_ = client.Close(InternalServerError.Error())
			return false, err
		}

		return true, nil
	case Plain:
		lengthOfUsernameAndPassword := make([]byte, 8)
		err = client.Read(lengthOfUsernameAndPassword)
		if err != nil {
			_ = client.Close(InternalServerError.Error())
			return false, err
		}

		lengthOfUsername := binary.BigEndian.Uint32(lengthOfUsernameAndPassword[:4])
		lengthOfPassword := binary.BigEndian.Uint32(lengthOfUsernameAndPassword[4:])

		usernameAndPassword := make([]byte, lengthOfUsername+lengthOfPassword)
		err = client.Read(usernameAndPassword)
		if err != nil {
			_ = client.Close(InternalServerError.Error())
			return false, err
		}

		username := usernameAndPassword[:lengthOfUsername]
		password := usernameAndPassword[lengthOfUsername:]

		if string(username) != s.Options.Config.VNC.Username || string(password) != s.Options.Config.VNC.Password {
			_, _ = client.Write(SecurityResultFail[:])
			return false, client.Close("Username or password is incorrect")
		}

		_, err = client.Write(SecurityResultOK[:])
		if err != nil {
			_ = client.Close(InternalServerError.Error())
			return false, err
		}

		return true, nil
	default:
		_, _ = client.Write(SecurityResultFail[:])
		return false, client.Close("Unsupported auth type")
	}
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

func (s *Server) handleFramebufferUpdateRequest(client *Client) error {
	s.locker.Lock()
	defer s.locker.Unlock()

	_ = client.Read(client.framebufferUpdateRequest)

	rects, err := s.Video.GetNextImageRects(s.Options.Config.Video.SliceCount, !client.fulfilled)
	if err != nil {
		return err
	}
	frame, err := s.VideoCodec.FramebufferUpdate(rects)
	if err != nil {
		return err
	}

	if len(rects) > 0 {
		client.fulfilled = true
	}

	_, err = client.Write(frame)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) handleEncoding(client *Client) error {
	err := client.Read(client.encodings)
	if err != nil {
		return err
	}

	number := binary.BigEndian.Uint16(client.encodings[1:3])

	encodings := make([]byte, number*4)
	err = client.Read(encodings)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) handleKeyEvent(client *Client) error {
	err := client.Read(client.keyEvent)
	if err != nil {
		return err
	}

	if s.Keyboard == nil {
		return KeyboardNotAvailable
	}

	copy(client.fullKeyEvent[1:], client.keyEvent)

	err = s.Keyboard.SendKeyEvent(client.fullKeyEvent)
	if err != nil {
		return err
	}

	return err
}

func (s *Server) handlePointerEvent(client *Client) error {
	err := client.Read(client.pointerEvent)
	if err != nil {
		return err
	}

	if s.Mouse == nil {
		return MouseNotAvailable
	}

	copy(client.fullPointerEvent[1:], client.pointerEvent)

	//  +--------------+--------------+--------------+
	// | No. of bytes | Type [Value] | Description  |
	// +--------------+--------------+--------------+
	// | 1            | U8 [5]       | message-type |
	// | 1            | U8           | button-mask  |
	// | 2            | U16          | x-position   |
	// | 2            | U16          | y-position   |
	// +--------------+--------------+--------------+

	pointerEvent := client.fullPointerEvent
	oldX := binary.BigEndian.Uint16(pointerEvent[2:4])
	oldY := binary.BigEndian.Uint16(pointerEvent[4:6])
	x := uint16(float64(oldX) * s.Options.Config.Mouse.CursorXScale)
	y := uint16(float64(oldY) * s.Options.Config.Mouse.CursorYScale)

	copy(pointerEvent[2:6], []byte{byte(x >> 8), byte(x), byte(y >> 8), byte(y)})

	l.Verbose().Printf("Rescale PointerEvent from (%d, %d) to (%d, %d)\n", oldX, oldY, x, y)

	err = s.Mouse.SendPointerEvent(pointerEvent)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) handleClientCut(client *Client) error {
	err := client.Read(client.clientCut)
	if err != nil {
		return err
	}

	length := binary.BigEndian.Uint32(client.clientCut[3:])

	text := make([]byte, length)
	err = client.Read(text)
	if err != nil {
		return err
	}

	l.Info().Println("ClientCutText:", string(text))

	// TODO handle clipboard

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

	msgType := make([]byte, 1)
	for {
		err = client.Read(msgType)
		if err != nil {
			return err
		}

		switch ClientMessageType(msgType[0]) {
		case SetPixelFormat:
			// 0000 0000 2018 0001 00ff 00ff 00ff 1008 0000 0000
			_ = client.Read(client.pixelFormat)
			continue
		case Placeholder:
			continue
		case SetEncodings:
			err = s.handleEncoding(client)
			if err != nil {
				l.Warn().Println("SetEncodings error:", err)
				continue
			}
			continue
		case FramebufferUpdateRequest:
			err = s.handleFramebufferUpdateRequest(client)
			if err != nil {
				l.Warn().Println("FramebufferUpdateRequest error:", err)
				continue
			}
		case KeyEvent:
			err = s.handleKeyEvent(client)
			if err != nil {
				l.Warn().Println("KeyEvent error:", err)
				continue
			}
		case PointerEvent:
			err = s.handlePointerEvent(client)
			if err != nil {
				l.Warn().Println("PointerEvent error:", err)
				continue
			}
		case ClientCutText:
			err = s.handleClientCut(client)
			if err != nil {
				l.Warn().Println("ClientCutText error:", err)
				continue
			}
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
	locker   sync.Locker
	buffer   []byte
	leftover []byte

	respSecurityType SecurityType
	challenge        []byte
	fulfilled        bool

	framebufferUpdateRequest []byte
	pixelFormat              []byte
	encodings                []byte
	keyEvent                 []byte
	pointerEvent             []byte
	clientCut                []byte

	fullKeyEvent     []byte
	fullPointerEvent []byte

	Messager io.ReadWriteCloser
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
		locker: &sync.Mutex{},
		buffer: make([]byte, 1024),

		//               +--------------+--------------+--------------+
		//              | No. of bytes | Type [Value] | Description  |
		//              +--------------+--------------+--------------+
		//              | 1            | U8 [0]       | message-type |
		//              | 3            |              | padding      |
		//              | 16           | PIXEL_FORMAT | pixel-format |
		//              +--------------+--------------+--------------+
		pixelFormat: make([]byte, 19),
		//            +--------------+--------------+---------------------+
		//           | No. of bytes | Type [Value] | Description         |
		//           +--------------+--------------+---------------------+
		//           | 1            | U8 [2]       | message-type        |
		//           | 1            |              | padding             |
		//           | 2            | U16          | number-of-encodings |
		//           | 4            | S32          | encoding-type       |
		//           +--------------+--------------+---------------------+
		encodings: make([]byte, 3),
		//               +--------------+--------------+--------------+
		//              | No. of bytes | Type [Value] | Description  |
		//              +--------------+--------------+--------------+
		//              | 1            | U8 [3]       | message-type |
		//              | 1            | U8           | incremental  |
		//              | 2            | U16          | x-position   |
		//              | 2            | U16          | y-position   |
		//              | 2            | U16          | width        |
		//              | 2            | U16          | height       |
		//              +--------------+--------------+--------------+
		framebufferUpdateRequest: make([]byte, 9),
		//               +--------------+--------------+--------------+
		//              | No. of bytes | Type [Value] | Description  |
		//              +--------------+--------------+--------------+
		//              | 1            | U8 [4]       | message-type |
		//              | 1            | U8           | down-flag    |
		//              | 2            |              | padding      |
		//              | 4            | U32          | key          |
		//              +--------------+--------------+--------------+
		keyEvent: make([]byte, 7),
		//               +--------------+--------------+--------------+
		//              | No. of bytes | Type [Value] | Description  |
		//              +--------------+--------------+--------------+
		//              | 1            | U8 [5]       | message-type |
		//              | 1            | U8           | button-mask  |
		//              | 2            | U16          | x-position   |
		//              | 2            | U16          | y-position   |
		//              +--------------+--------------+--------------+
		pointerEvent: make([]byte, 5),
		//               +--------------+--------------+--------------+
		//              | No. of bytes | Type [Value] | Description  |
		//              +--------------+--------------+--------------+
		//              | 1            | U8 [6]       | message-type |
		//              | 3            |              | padding      |
		//              | 4            | U32          | length       |
		//              | length       | U8 array     | text         |
		//              +--------------+--------------+--------------+
		clientCut: make([]byte, 7),

		fullKeyEvent:     append([]byte{byte(KeyEvent)}, bytes.Repeat([]byte{0}, 7)...),
		fullPointerEvent: append([]byte{byte(PointerEvent)}, bytes.Repeat([]byte{0}, 5)...),

		Messager: message,
	}
}
