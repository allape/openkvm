package device

import (
	"errors"
	"github.com/allape/openkvm/config/sub"
	"github.com/allape/openkvm/kvm/video"
	"github.com/allape/openkvm/kvm/video/placeholder"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const Tag = "[video-device]"

type PlaceholderOptions struct {
	Width  int
	Height int
}

type Commander struct {
	Command string
	Args    []string
}

type Device struct {
	video.Driver

	locker          sync.Locker
	interpolateTime time.Duration
	mat             *gocv.Mat

	// tmp
	img   image.Image
	rects []video.Rect

	LastCaptureTime *time.Time
	FrameRate       float64
	Src             string
	FlipCode        video.FlipCode
	WebCam          *gocv.VideoCapture
	Placeholder     PlaceholderOptions

	PreludeCommand Commander
}

func (d *Device) GetMat() (*gocv.Mat, video.Changed, error) {
	if d.WebCam == nil || d.mat == nil {
		return nil, false, errors.New("webcam is not opened")
	}

	d.locker.Lock()
	defer func() {
		d.locker.Unlock()
	}()

	now := time.Now()

	if d.LastCaptureTime != nil && !now.After(d.LastCaptureTime.Add(d.interpolateTime)) {
		return d.mat, false, nil
	}

	if d.PreludeCommand.Command != "" {
		cmd := exec.Command(d.PreludeCommand.Command, d.PreludeCommand.Args...)
		output, err := cmd.CombinedOutput()
		log.Println(Tag, "prelude command:", strings.TrimSpace(string(output)))
		if err != nil {
			return nil, true, errors.New("failed to run prelude command: " + err.Error())
		}
	}
	if ok := d.WebCam.Read(d.mat); !ok {
		return nil, true, errors.New("failed to read frame")
	}

	// flip mat at horizon
	if d.FlipCode != video.Nothing {
		gocv.Flip(*d.mat, d.mat, int(d.FlipCode))
	}

	d.LastCaptureTime = &now

	return d.mat, true, nil
}

func (d *Device) Open() error {
	webcam, err := gocv.OpenVideoCapture(d.Src)
	if err != nil {
		return err
	}
	d.WebCam = webcam
	buffer := gocv.NewMat()
	d.mat = &buffer
	return nil
}

func (d *Device) Close() error {
	if d.WebCam == nil {
		return nil
	}
	_ = d.mat.Close()
	return d.WebCam.Close()
}

func (d *Device) GetSize() (*image.Point, error) {
	frame, _, err := d.GetFrame()
	if err != nil {
		return nil, err
	}
	size := frame.Bounds().Size()
	return &size, err
}

func (d *Device) GetFrameRate() float64 {
	return d.FrameRate
}

func (d *Device) Reset() error {
	d.rects = nil
	d.img = nil
	return nil
}

func (d *Device) GetPlaceholderImage(text string) (video.Frame, error) {
	return placeholder.CreatePlaceholder(
		d.Placeholder.Width, d.Placeholder.Height,
		color.RGBA{A: 255},
		color.RGBA{R: 255, G: 0, B: 0, A: 255},
		text,
		true,
	)
}

func (d *Device) GetFrame() (video.Frame, video.Changed, error) {
	mat, changed, err := d.GetMat()
	if err != nil {
		ph, phErr := d.GetPlaceholderImage(err.Error())
		if phErr == nil {
			d.img = ph
			return ph, true, nil
		}
		return nil, changed, err
	}
	if d.img != nil && !changed {
		return d.img, changed, nil
	}
	d.img, err = mat.ToImage()
	return d.img, changed, err
}

type SubImager interface {
	image.Image
	SubImage(r image.Rectangle) image.Image
}

func (d *Device) GetNextImageRects(sliceCount video.SliceCount, full bool) ([]video.Rect, error) {
	sc := int(sliceCount)

	var err error
	var lastImage image.Image

	if d.img != nil {
		lastImage = d.img
	}

	im, changed, err := d.GetFrame()
	if err != nil {
		return nil, err
	}
	if len(d.rects) > 0 && changed == false && !full {
		return d.rects, nil
	}
	img, ok := im.(SubImager)
	if !ok {
		return nil, errors.New("image does not support sub-imaging")
	}

	imageSize := img.Bounds().Size()

	if imageSize.X%sc != 0 {
		return nil, errors.New("image width should be divisible by slice count")
	}

	rectSize := image.Point{X: imageSize.X / sc, Y: imageSize.Y / sc}

	if imageSize.X%rectSize.X != 0 {
		return nil, errors.New("image width should be divisible by rect width")
	} else if imageSize.Y%rectSize.Y != 0 {
		return nil, errors.New("image height should be divisible by rect height")
	}

	colCount := imageSize.X / rectSize.X
	rowCount := imageSize.Y / rectSize.Y

	rectChangedMarks := make([][]bool, colCount)
	for i := range rectChangedMarks {
		rectChangedMarks[i] = make([]bool, rowCount)
	}

	// scanning for section changes
	if lastImage == nil || full {
		for i := 0; i < colCount; i++ {
			for j := 0; j < rowCount; j++ {
				rectChangedMarks[i][j] = true
			}
		}
	} else {
		var wait sync.WaitGroup
		for i := 0; i < colCount; i++ {
			for j := 0; j < rowCount; j++ {
				wait.Add(1)
				go func(x, y int) {
					defer wait.Done()
					rectChangedMarks[x][y] = ImageChanged(lastImage, img, imageSize, x*rectSize.X, y*rectSize.Y, rectSize.X, rectSize.Y)
				}(i, j)
			}
		}
		wait.Wait()
	}

	rects := make([]video.Rect, 0)
	for i := 0; i < colCount; i++ {
		for j := 0; j < rowCount; j++ {
			if !rectChangedMarks[i][j] {
				continue
			}
			rects = append(rects, video.Rect{
				X:     uint64(i * rectSize.X),
				Y:     uint64(j * rectSize.Y),
				Frame: img.SubImage(image.Rect(i*rectSize.X, j*rectSize.Y, (i+1)*rectSize.X, (j+1)*rectSize.Y)),
			})
		}
	}

	if !full {
		d.rects = rects
	}

	return rects, nil
}

func ImageChanged(img1, img2 image.Image, size image.Point, offsetX, offsetY, width, height int) bool {
	rectMaxWidth := offsetX + width
	if rectMaxWidth > size.X {
		rectMaxWidth = size.X
	}
	rectMaxHeight := offsetY + height
	if rectMaxHeight > size.Y {
		rectMaxHeight = size.Y
	}

	for x := offsetX; x < rectMaxWidth; x++ {
		for y := offsetY; y < rectMaxHeight; y++ {
			r1, b1, g1, _ := img1.At(x, y).RGBA()
			r2, b2, g2, _ := img2.At(x, y).RGBA()
			if r1 != r2 || b1 != b2 || g1 != g2 {
				return true
			}
		}
	}

	return false
}

type Options struct {
	PreludeCommand sub.PreludeCommand
	Placeholder    PlaceholderOptions
	FlipCode       video.FlipCode
	FrameRate      float64
}

func NewDevice(src string, options *Options) video.Driver {
	if options == nil {
		options = &Options{}
	}

	if options.FrameRate == 0 {
		options.FrameRate = 30
	}
	if options.Placeholder.Width == 0 {
		options.Placeholder = PlaceholderOptions{
			Width:  1920,
			Height: 1080,
		}
	}

	cmd, args := options.PreludeCommand.Get()

	dev := &Device{
		LastCaptureTime: nil,

		Src:         src,
		FrameRate:   options.FrameRate,
		FlipCode:    options.FlipCode,
		Placeholder: options.Placeholder,
		PreludeCommand: Commander{
			Command: cmd,
			Args:    args,
		},

		locker:          &sync.Mutex{},
		interpolateTime: time.Duration(float64(time.Second) / options.FrameRate),
	}

	return dev
}
