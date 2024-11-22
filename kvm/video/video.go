package video

import (
	"github.com/allape/openkvm/config"
	"image"
)

type (
	Changed bool
)

type Driver interface {
	Open() error
	Close() error

	GetFrameRate() float64
	GetSize() (*image.Point, error)
	GetPlaceholderImage(text string) (config.Frame, error)

	Reset() error
	GetFrame() (config.Frame, Changed, error)
	GetNextImageRects(count config.SliceCount, full bool) ([]config.Rect, error)
}

type Options struct {
	Width          int
	Height         int
	FrameRate      float64
	Quality        int
	SliceCount     config.SliceCount
	PreludeCommand config.ShellCommand
}
