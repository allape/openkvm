package factory

import (
	"errors"
	"fmt"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/video"
	"github.com/allape/openkvm/kvm/video/dummy"
	"github.com/allape/openkvm/kvm/video/shell"
)

func VideoFromConfig(conf config.Config) (vd video.Driver, err error) {
	vos := video.Options{
		Width:         conf.Video.Width,
		Height:        conf.Video.Height,
		FrameRate:     conf.Video.FrameRate,
		SetupCommands: conf.Video.SetupCommands,
	}

	switch conf.Video.Type {
	case config.VideoUSBDevice:
		return nil, errors.New("video usb device is deprecated")
		//vd = usb.NewDevice(conf.Video.Src, &usb.Options{
		//	Options: vos,
		//})
	case config.VideoShellDevice:
		src := config.VideoShellSrc(conf.Video.Src)
		if src.Empty() {
			return nil, fmt.Errorf("video source is empty")
		}
		vd = shell.NewDriver(src, &shell.Options{
			Options: vos,
		})
	case config.VideoDummyDevice:
		if conf.Video.Src.Empty() {
			return nil, fmt.Errorf("video source is empty")
		}
		src := conf.Video.Src[0]
		vd = dummy.NewDriver(src, &dummy.Options{
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
