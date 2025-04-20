package dummy

import (
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/video"
	"github.com/allape/openkvm/kvm/video/placeholder"
	"image"
	"image/color"
	"time"
)

type Driver struct {
	video.Driver

	src      string
	lastImg  config.Frame
	lastTime int64

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

func (d *Driver) GetSize() (*image.Point, error) {
	return &image.Point{X: d.Width, Y: d.Height}, nil
}

func (d *Driver) GetFrame() (config.Frame, video.Changed, error) {
	img, err := placeholder.CreatePlaceholder(
		d.Width, d.Height,
		color.RGBA{A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
		d.src,
		true,
	)

	return img, true, err
}

func (d *Driver) GetNextImageRects(sliceCount config.SliceCount, full bool) ([]config.Rect, error) {
	now := time.Now().UnixMilli()
	if now-d.lastTime <= int64(1000/d.FrameRate) {
		return nil, nil
	}

	d.lastTime = now

	im, _, err := d.GetFrame()
	if err != nil {
		return nil, err
	}

	var lastImage image.Image

	if d.lastImg != nil {
		lastImage = d.lastImg
	}

	d.lastImg = im.(image.Image)

	rects, err := video.GetNextImageRects(lastImage, d.lastImg, sliceCount, full)

	return rects, nil
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
		Height:    options.Width,
		FrameRate: options.FrameRate,
	}
}
