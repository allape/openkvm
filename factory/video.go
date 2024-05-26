package config

import (
	"fmt"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/video"
	"github.com/allape/openkvm/kvm/video/device"
)

func VideoFromConfig(conf config.Config) (vd video.Driver, err error) {
	switch conf.Video.Type {
	case config.VideoUSBDevice:
		vd = device.NewDevice(conf.Video.Src, &device.Options{
			Width:          conf.Video.Width,
			Height:         conf.Video.Height,
			FrameRate:      conf.Video.FrameRate,
			FlipCode:       conf.Video.FlipCode,
			PreludeCommand: conf.Video.PreludeCommand,
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
