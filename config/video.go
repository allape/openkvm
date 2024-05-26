package config

import (
	"image"
)

type Frame image.Image

// FlipCode
// gocv.Flip, Flip flips a 2D array around horizontal(0), vertical(1), or both axes(-1). -2 for nothing
type FlipCode int

// SliceCount how many number use to cut the frame in both x-axis and y-axis.
// For example,
//
//	if SliceCount is 4, then the frame of 1920x1080 will be divided into 16 slices,
//	and the size of each slice should be 480x270.
type SliceCount int

const (
	Horizontal FlipCode = 0
	Vertical   FlipCode = 1
	Both       FlipCode = -1
	NoFlip     FlipCode = -2
)

type Rect struct {
	X     uint64
	Y     uint64
	Frame Frame
}
