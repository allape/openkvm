package video

import (
	"github.com/allape/openkvm/config"
)

type (
	Changed bool
)

type Driver interface {
	Open() error
	Close() error

	GetFrameRate() float64
	GetSize() (*config.Size, error)

	NextFrame() (config.Frame, error)
}

type Options struct {
	Width         int
	Height        int
	FrameRate     float64
	SetupCommands []config.SetupCommand
}
