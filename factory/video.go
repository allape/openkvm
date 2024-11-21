package factory

import (
	"fmt"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/video"
	"github.com/allape/openkvm/kvm/video/clt"
	"github.com/allape/openkvm/kvm/video/usb"
)

func VideoFromConfig(conf config.Config) (vd video.Driver, err error) {
	vos := video.Options{
		Width:          conf.Video.Width,
		Height:         conf.Video.Height,
		FrameRate:      conf.Video.FrameRate,
		FlipCode:       conf.Video.FlipCode,
		PreludeCommand: conf.Video.PreludeCommand,
	}

	switch conf.Video.Type {
	case config.VideoUSBDevice:
		vd = usb.NewDevice(conf.Video.Src, &usb.Options{
			Options: vos,
		})
	case config.VideoCltDevice:
		if conf.Video.Src == "" {
			return nil, fmt.Errorf("video source is empty")
		}

		startMaker, err := config.HexStringMarker(conf.Video.Ext.Get("startmarker")).ToByteArray()
		if err != nil {
			return nil, err
		}
		endMarker, err := config.HexStringMarker(conf.Video.Ext.Get("endmarker")).ToByteArray()
		if err != nil {
			return nil, err
		}

		vd = clt.NewClt(config.NewShellCommand(conf.Video.Src), &clt.Options{
			Options:    vos,
			StartMaker: startMaker,
			EndMarker:  endMarker,
		})
	default:
		return nil, fmt.Errorf("unknown video driver: %s", conf.Video.Type)
	}

	err = vd.Open()
	if err != nil {
		return nil, err
	}

	return vd, err
}
