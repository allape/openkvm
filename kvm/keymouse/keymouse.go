package keymouse

import (
	"io"
)

type KeyEvent []byte
type PointerEvent []byte

type KeyboardMouseDriver interface {
	io.Writer
	io.Closer
	Open() error
	SendKeyEvent(e KeyEvent) error
	SendPointerEvent(e PointerEvent) error
}
