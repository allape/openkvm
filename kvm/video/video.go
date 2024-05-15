package video

import (
	"image"
)

type (
	Frame   image.Image
	Changed bool

	// SliceCount how many number use to cut the frame in both x-axis and y-axis.
	// For example,
	//     if SliceCount is 4, then the frame of 1920x1080 will be divided into 16 slices,
	//     and the size of each slice should be 480x270.
	SliceCount int
)

type Rect struct {
	X     uint64
	Y     uint64
	Frame Frame
}

type Driver interface {
	Open() error
	Close() error

	GetFrameRate() float64
	GetSize() (*image.Point, error)
	GetPlaceholderImage(text string) (Frame, error)

	Reset() error
	GetFrame() (Frame, Changed, error)
	GetNextImageRects(count SliceCount, full bool) ([]Rect, error)
}

// FlipCode
// gocv.Flip, Flip flips a 2D array around horizontal(0), vertical(1), or both axes(-1). -2 for nothing
type FlipCode int

const (
	Horizontal FlipCode = 0
	Vertical   FlipCode = 1
	Both       FlipCode = -1
	NoFlip     FlipCode = -2
)
