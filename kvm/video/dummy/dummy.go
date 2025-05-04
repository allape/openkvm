package dummy

import (
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/helper/placeholder"
	"github.com/allape/openkvm/kvm/video"
	"image/color"
	"time"
)

type Driver struct {
	video.Driver

	src       string
	lastFrame config.Frame
	lastTime  int64

	Width     int
	Height    int
	FrameRate float64
}

func (d *Driver) Open() error {
	return nil
}

func (d *Driver) Close() error {
	return nil
}

func (d *Driver) GetFrameRate() float64 {
	return d.FrameRate
}

func (d *Driver) GetSize() (*config.Size, error) {
	return &config.Size{X: d.Width, Y: d.Height}, nil
}

func (d *Driver) NextFrame() (config.Frame, error) {
	now := time.Now().UnixMilli()
	if now-d.lastTime <= int64(1000/d.FrameRate) {
		return d.lastFrame, nil
	}

	d.lastTime = now

	img, err := placeholder.CreatePlaceholder(
		d.Width, d.Height,
		color.RGBA{A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
		d.src,
		true,
	)
	if err != nil {
		return nil, err
	}

	d.lastFrame = img

	return img, nil
}

type Options struct {
	video.Options
}

func NewDriver(src string, options *Options) video.Driver {
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
		src:       src,
		Width:     options.Width,
		Height:    options.Height,
		FrameRate: options.FrameRate,
	}
}
