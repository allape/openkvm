package serialport

import (
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/clipboard"
	"github.com/allape/openkvm/kvm/keymouse"
	"io"
)

type Clipboard struct {
	clipboard.Driver

	Config        config.Clipboard
	KeyboardMouse keymouse.Driver
}

func (c *Clipboard) Close() error {
	return c.KeyboardMouse.Close()
}

func (c *Clipboard) Open() error {
	return c.KeyboardMouse.Open()
}

func (c *Clipboard) Write(buffer []byte) (int, error) {
	length := len(buffer)

	lengthByte1 := byte((length >> 8) & 0xFF)
	lengthByte2 := byte(length & 0xFF)

	n, err := c.KeyboardMouse.Write(append([]byte{0xFE, lengthByte1, lengthByte2}, buffer...))
	if err != nil {
		return n, err
	} else if n < 3 {
		return 0, io.ErrShortWrite
	}

	return n - 3, nil
}

func (c *Clipboard) Read(_ []byte) (int, error) {
	// TODO not supported yet
	return 0, nil
}
