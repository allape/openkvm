package clt

import (
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/video"
)

type Driver struct {
	video.Driver

	cmd        config.ShellCommand
	preludeCmd config.ShellCommand

	Width  int
	Height int
}

// TODO

type Options struct {
	video.Options
	StartMaker []byte
	EndMarker  []byte
}

func NewClt(src config.ShellCommand, options *Options) video.Driver {
	if options == nil {
		options = &Options{}
	}

	if options.Width == 0 {
		options.Width = 1920
	}
	if options.Height == 0 {
		options.Height = 1080
	}
	if options.FrameRate == 0 {
		options.FrameRate = 30
	}

	return &Driver{
		cmd:        src,
		preludeCmd: options.PreludeCommand,

		Width:  options.Width,
		Height: options.Width,
	}
}
