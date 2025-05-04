package shell

import (
	"bytes"
	"errors"
	"github.com/allape/gogger"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/video"
	"image/jpeg"
	"io"
	"os"
	"sync"
	"time"
)

var l = gogger.New("kvm.video.shell")

type Driver struct {
	video.Driver

	src           config.VideoShellSrc
	setupCommands []config.SetupCommand

	process *os.Process
	locker  sync.Locker

	frameBuffer          []byte
	bufferLocker         sync.Locker
	frameBufferUpdatedAt int64

	lastTime           int64
	lastFrame          config.Frame
	nextFrameLocker    sync.Locker
	nextFrameInvokedAt int64

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

	cmd, err := d.src.ToCommand()
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
					l.Verbose().Println(err)
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
					l.Verbose().Println("frame updated")
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
					l.Error().Println(err)
				}
				return
			}
			l.Verbose().Print(string(buf[:n]))
		}
	}()

	for _, command := range d.setupCommands {
		setup, err := command.ToCommand()
		if err != nil {
			return err
		}
		l.Verbose().Println(setup.Path, setup.Args)
		output, err := setup.CombinedOutput()
		o := string(output)
		l.Verbose().Print("prelude output:", o)
		if err != nil {
			return errors.New(o)
		}
	}

	l.Verbose().Println(cmd.Path, cmd.Args)

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

func (d *Driver) GetSize() (*config.Size, error) {
	return &config.Size{X: d.Width, Y: d.Height}, nil
}

func (d *Driver) NextFrame() (config.Frame, error) {
	d.nextFrameLocker.Lock()
	defer d.nextFrameLocker.Unlock()

	now := time.Now().UnixMilli()
	if now-d.lastTime <= int64(1000/d.FrameRate) {
		return d.lastFrame, nil
	}

	d.lastTime = now

	buf := d.frameBuffer
	updatedAt := d.frameBufferUpdatedAt

	if buf == nil {
		return nil, nil
	}

	if updatedAt > 0 && updatedAt == d.nextFrameInvokedAt {
		return d.lastFrame, nil
	}

	img, err := jpeg.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}

	d.lastFrame = img
	d.nextFrameInvokedAt = updatedAt

	return img, nil
}

type Options struct {
	video.Options
}

func NewDriver(src config.VideoShellSrc, options *Options) video.Driver {
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
		src:           src,
		setupCommands: options.SetupCommands,

		locker:          &sync.Mutex{},
		bufferLocker:    &sync.Mutex{},
		nextFrameLocker: &sync.Mutex{},

		Width:       options.Width,
		Height:      options.Width,
		FrameRate:   options.FrameRate,
		StartMarker: []byte{0xff, 0xd8},
		EndMarker:   []byte{0xff, 0xd9},
	}
}
