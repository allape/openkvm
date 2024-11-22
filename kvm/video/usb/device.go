package usb

//import (
//	"errors"
//	"github.com/allape/openkvm/config"
//	"github.com/allape/openkvm/kvm/video"
//	"github.com/allape/openkvm/kvm/video/placeholder"
//	"github.com/allape/openkvm/logger"
//	"gocv.io/x/gocv"
//	"image"
//	"image/color"
//	"strings"
//	"sync"
//	"time"
//)
//
//var log = logger.NewVerboseLogger("[video-usb]")
//
//type Device struct {
//	video.Driver
//
//	locker          sync.Locker
//	interpolateTime time.Duration
//	mat             *gocv.Mat
//
//	// tmp
//	img   image.Image
//	rects []config.Rect
//
//	LastCaptureTime *time.Time
//	WebCam          *gocv.VideoCapture
//
//	Src            string
//	Width          int
//	Height         int
//	FrameRate      float64
//	FlipCode       config.FlipCode
//	PreludeCommand config.ShellCommand
//}
//
//func (d *Device) GetMat() (*gocv.Mat, video.Changed, error) {
//	if d.WebCam == nil || d.mat == nil {
//		return nil, false, errors.New("webcam is not opened")
//	}
//
//	d.locker.Lock()
//	defer func() {
//		d.locker.Unlock()
//	}()
//
//	now := time.Now()
//
//	if d.LastCaptureTime != nil && !now.After(d.LastCaptureTime.Add(d.interpolateTime)) {
//		return d.mat, false, nil
//	}
//
//	if ok := d.WebCam.Read(d.mat); !ok {
//		return nil, true, errors.New("failed to read frame")
//	}
//
//	// flip mat at horizon
//	if d.FlipCode != config.NoFlip {
//		gocv.Flip(*d.mat, d.mat, int(d.FlipCode))
//	}
//
//	d.LastCaptureTime = &now
//
//	return d.mat, true, nil
//}
//
//func (d *Device) Open() error {
//	d.locker.Lock()
//	defer d.locker.Unlock()
//
//	var err error
//
//	cmd, err := d.PreludeCommand.ToCommand()
//	if err != nil {
//		return err
//	} else if cmd != nil {
//		output, err := cmd.CombinedOutput()
//		log.Println("prelude command:", strings.TrimSpace(string(output)))
//		if err != nil {
//			return err
//		}
//	}
//
//	d.WebCam, err = gocv.OpenVideoCapture(d.Src)
//	if err != nil {
//		return err
//	}
//
//	buffer := gocv.NewMat()
//	d.mat = &buffer
//
//	// We should read these from OpenCV instead of setting them
//	//d.WebCam.Set(gocv.VideoCaptureFrameWidth, float64(d.Width))
//	//d.WebCam.Set(gocv.VideoCaptureFrameHeight, float64(d.Height))
//	//d.WebCam.Set(gocv.VideoCaptureFPS, d.FrameRate)
//
//	return nil
//}
//
//func (d *Device) Close() error {
//	if d.WebCam == nil {
//		return nil
//	}
//	_ = d.mat.Close()
//	return d.WebCam.Close()
//}
//
//func (d *Device) GetSize() (*image.Point, error) {
//	frame, _, err := d.GetFrame()
//	if err != nil {
//		return nil, err
//	}
//	size := frame.Bounds().Size()
//	return &size, err
//}
//
//func (d *Device) GetFrameRate() float64 {
//	return d.FrameRate
//}
//
//func (d *Device) Reset() error {
//	d.rects = nil
//	d.img = nil
//	return nil
//}
//
//func (d *Device) GetPlaceholderImage(text string) (config.Frame, error) {
//	return placeholder.CreatePlaceholder(
//		d.Width, d.Height,
//		color.RGBA{A: 255},
//		color.RGBA{R: 255, G: 0, B: 0, A: 255},
//		text,
//		true,
//	)
//}
//
//func (d *Device) GetFrame() (config.Frame, video.Changed, error) {
//	mat, changed, err := d.GetMat()
//	if err != nil {
//		ph, phErr := d.GetPlaceholderImage(err.Error())
//		if phErr == nil {
//			d.img = ph
//			return ph, true, nil
//		}
//		return nil, changed, err
//	}
//
//	if d.img != nil && !changed {
//		return d.img, changed, nil
//	}
//
//	sizes := mat.Size()
//	if sizes[0] != d.Width || sizes[1] != d.Height {
//		gocv.Resize(*mat, mat, image.Point{X: d.Width, Y: d.Height}, 0, 0, gocv.InterpolationLinear)
//	}
//
//	d.img, err = mat.ToImage()
//	return d.img, changed, err
//}
//
//func (d *Device) GetNextImageRects(sliceCount config.SliceCount, full bool) ([]config.Rect, error) {
//	var lastImage image.Image
//
//	if d.img != nil {
//		lastImage = d.img
//	}
//
//	im, changed, err := d.GetFrame()
//	if err != nil {
//		return nil, err
//	}
//
//	if len(d.rects) > 0 && changed == false && !full {
//		return d.rects, nil
//	}
//
//	img := im.(image.Image)
//
//	rects, err := video.GetNextImageRects(lastImage, img, sliceCount, full)
//
//	if !full {
//		d.rects = rects
//	}
//
//	return rects, nil
//}
//
//type Options struct {
//	video.Options
//}
//
//// NewDevice
//// deprecated
//// go get -u gocv.io/x/gocv v0.39.0
//func NewDevice(src string, options *Options) video.Driver {
//	if options == nil {
//		options = &Options{}
//	}
//
//	if options.Width == 0 {
//		options.Width = 1920
//	}
//	if options.Height == 0 {
//		options.Height = 1080
//	}
//	if options.FrameRate == 0 {
//		options.FrameRate = 30
//	}
//
//	dev := &Device{
//		locker:          &sync.Mutex{},
//		interpolateTime: time.Duration(float64(time.Second) / options.FrameRate),
//
//		Src:            src,
//		Width:          options.Width,
//		Height:         options.Height,
//		FrameRate:      options.FrameRate,
//		FlipCode:       options.FlipCode,
//		PreludeCommand: options.PreludeCommand,
//	}
//
//	return dev
//}
