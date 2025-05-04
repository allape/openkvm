package config

import (
	"image"
)

type (
	Frame image.Image
	Size  image.Point

	// SliceCount how many number use to cut the frame in both x-axis and y-axis.
	// For example,
	//
	//	if SliceCount is 4, then the frame of 1920x1080 will be divided into 16 slices,
	//	and the size of each slice should be 480x270.
	SliceCount int

	Rect struct {
		X     uint64
		Y     uint64
		Frame Frame
	}
)
