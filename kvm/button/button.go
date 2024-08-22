package button

import "io"

type Type string

const (
	PowerButton Type = "power"
	ResetButton Type = "reset"
	ExtraButton Type = "extra"
)

type Driver interface {
	io.Closer
	Open() error
	Press(t Type) error
	Release(t Type) error
}
