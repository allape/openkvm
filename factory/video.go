package factory

import (
	"errors"
	"fmt"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/video"
	"github.com/allape/openkvm/kvm/video/shell"
)

func VideoFromConfig(conf config.Config) (vd video.Driver, err error) {
	vos := video.Options{
		Width:          conf.Video.Width,
		Height:         conf.Video.Height,
		FrameRate:      conf.Video.FrameRate,
		Quality:        conf.Video.Quality,
		SliceCount:     conf.Video.SliceCount,
		PreludeCommand: conf.Video.PreludeCommand,
	}

	switch conf.Video.Type {
	case config.VideoUSBDevice:
		return nil, errors.New("video usb device is deprecated")
		//vd = usb.NewDevice(conf.Video.Src, &usb.Options{
		//	Options: vos,
		//})
	case config.VideoShellDevice:
		if conf.Video.Src == "" {
			return nil, fmt.Errorf("video source is empty")
		}
		vd = shell.NewDriver(config.NewShellCommand(conf.Video.Src), &shell.Options{
			Options: vos,
		})
	default:
		return nil, fmt.Errorf("unknown video driver: %s", conf.Video.Type)
	}

	//err = vd.Open()
	//if err != nil {
	//	return nil, err
	//}

	return vd, err
}
