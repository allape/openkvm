package shell

import (
	"bytes"
	"errors"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/video"
	"github.com/allape/openkvm/kvm/video/placeholder"
	"github.com/allape/openkvm/logger"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"
	"sync"
	"time"
)

var log = logger.New("[kvm.video.shell]")
var verbose = logger.NewVerboseLogger("[kvm.video.shell]")

type Driver struct {
	video.Driver

	cmd        config.ShellCommand
	preludeCmd config.ShellCommand

	process *os.Process
	locker  sync.Locker

	frameBuffer          []byte
	bufferLocker         sync.Locker
	frameBufferUpdatedAt int64

	lastGotFrame   config.Frame
	getFrameLocker sync.Locker
	getFrameAt     int64

	rects       []config.Rect
	rectsLocker sync.Locker

	Width       int
	Height      int
	FrameRate   float64
	StartMarker []byte
	EndMarker   []byte
}

func (d *Driver) Open() error {
	d.locker.Lock()
	defer d.locker.Unlock()

	if d.process != nil {
		return nil
	}

	prelude, err := d.preludeCmd.ToCommand()
	if err != nil {
		return err
	}

	cmd, err := d.cmd.ToCommand()
	if err != nil {
		return err
	} else if cmd == nil {
		return errors.New("command is nil")
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	readyChan := make(chan struct{}, 1)

	go func() {
		ready := false
		started := false

		var frameBuffer []byte
		buf := make([]byte, 1024)

		for {
			n, err := stdout.Read(buf)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					verbose.Println(err)
				}
				return
			}
			seg := buf[:n]

			for {
				if len(seg) == 0 {
					break
				}

				if !started {
					index := bytes.Index(seg, d.StartMarker)
					if index != -1 {
						started = true
						seg = seg[index:]

						if !ready {
							ready = true
							go func() {
								readyChan <- struct{}{}
							}()
						}
					} else {
						seg = nil
						continue
					}
				}

				index := bytes.Index(seg, d.EndMarker)
				if index != -1 {
					index = index + len(d.EndMarker)
					d.bufferLocker.Lock()
					d.frameBuffer = append(frameBuffer, seg[:index]...)
					d.frameBufferUpdatedAt = time.Now().UnixMicro()
					verbose.Println("frame updated")
					d.bufferLocker.Unlock()
					started = false
					frameBuffer = nil
					seg = seg[index:]
					continue
				}

				frameBuffer = append(frameBuffer, seg...)
				seg = nil
			}
		}
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					log.Println(err)
				}
				return
			}
			verbose.Print(string(buf[:n]))
		}
	}()

	if prelude != nil {
		verbose.Println(prelude.Path, prelude.Args)
		output, err := prelude.CombinedOutput()
		if err != nil {
			return errors.New(string(output))
		}
		verbose.Print("prelude output:", string(output))
	}

	verbose.Println(cmd.Path, cmd.Args)

	err = cmd.Start()
	if err != nil {
		return err
	}

	d.process = cmd.Process

	<-readyChan

	return nil
}

func (d *Driver) Close() error {
	d.locker.Lock()
	defer d.locker.Unlock()

	if d.process == nil {
		return nil
	}

	err := d.process.Kill()
	if err != nil {
		return err
	}

	d.process = nil

	return nil
}

func (d *Driver) GetFrameRate() float64 {
	return d.FrameRate
}

func (d *Driver) GetSize() (*image.Point, error) {
	return &image.Point{X: d.Width, Y: d.Height}, nil
}

func (d *Driver) GetPlaceholderImage(text string) (config.Frame, error) {
	return placeholder.CreatePlaceholder(
		d.Width, d.Height,
		color.RGBA{A: 255},
		color.RGBA{R: 255, G: 0, B: 0, A: 255},
		text,
		true,
	)
}

func (d *Driver) Reset() error {
	d.bufferLocker.Lock()
	defer d.bufferLocker.Unlock()

	d.frameBuffer = nil

	return nil
}

func (d *Driver) GetFrame() (config.Frame, video.Changed, error) {
	d.getFrameLocker.Lock()
	defer d.getFrameLocker.Unlock()

	updatedAt := d.frameBufferUpdatedAt
	buf := d.frameBuffer

	if buf == nil {
		return nil, false, nil
	}

	if updatedAt > 0 && updatedAt == d.getFrameAt {
		return d.lastGotFrame, false, nil
	}

	img, err := jpeg.Decode(bytes.NewReader(d.frameBuffer))
	if err != nil {
		return nil, false, err
	}

	d.lastGotFrame = img
	d.getFrameAt = updatedAt

	return img, true, nil
}

func (d *Driver) GetNextImageRects(sliceCount config.SliceCount, full bool) ([]config.Rect, error) {
	d.rectsLocker.Lock()
	defer d.rectsLocker.Unlock()

	var lastImage image.Image

	if d.lastGotFrame != nil {
		lastImage = d.lastGotFrame
	}

	im, changed, err := d.GetFrame()
	if err != nil {
		return nil, err
	}

	if len(d.rects) > 0 && changed == false && !full {
		return d.rects, nil
	}

	img := im.(image.Image)

	rects, err := video.GetNextImageRects(lastImage, img, sliceCount, full)

	if !full {
		d.rects = rects
	}

	return rects, nil
}

type Options struct {
	video.Options
}

func NewDriver(src config.ShellCommand, options *Options) video.Driver {
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

		locker:         &sync.Mutex{},
		bufferLocker:   &sync.Mutex{},
		getFrameLocker: &sync.Mutex{},
		rectsLocker:    &sync.Mutex{},

		Width:       options.Width,
		Height:      options.Width,
		FrameRate:   options.FrameRate,
		StartMarker: []byte{0xff, 0xd8},
		EndMarker:   []byte{0xff, 0xd9},
	}
}
