package config

import (
	"fmt"
	"github.com/allape/openkvm/kvm/codec"
	"github.com/allape/openkvm/kvm/codec/tight"
	"github.com/allape/openkvm/kvm/keymouse"
	"github.com/allape/openkvm/kvm/keymouse/serialport"
	"github.com/allape/openkvm/kvm/video"
	"github.com/allape/openkvm/kvm/video/device"
	"log"
)

func KeyboardFromConfig(conf Config) (kd keymouse.KeyboardMouseDriver, err error) {
	switch conf.Keyboard.Type {
	case KeyboardNone:
		log.Println(Tag, "keyboard driver is none, no keyboard output")
	case KeyboardSerialPort:
		log.Println(Tag, "keyboard driver is serial port:", conf.Keyboard.Src)
		baud, err := conf.Mouse.Ext.GetInt("baud", 9600)
		if err != nil {
			return nil, err
		}
		kd = serialport.New(conf.Keyboard.Src, baud)
	}

	if kd != nil {
		err = kd.Open()
		if err != nil {
			log.Println(Tag, "open keyboard driver:", err)
			//return kd, err
		}
	}

	return kd, nil
}

func VideoFromConfig(conf Config) (vd video.Driver, err error) {
	switch conf.Video.Type {
	case VideoUSBDevice:
		phw, err := conf.Video.Ext.GetInt("phw", 1920)
		if err != nil {
			return nil, err
		}
		phh, err := conf.Video.Ext.GetInt("phh", 1080)
		if err != nil {
			return nil, err
		}
		vd = device.NewDevice(conf.Video.Src, conf.Video.FrameRate, conf.Video.FlipCode, &device.PlaceholderOptions{
			Width:  phw,
			Height: phh,
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

func MouseFromConfigOrUseKeyboard(kd keymouse.KeyboardMouseDriver, conf Config) (md keymouse.KeyboardMouseDriver, err error) {
	if string(conf.Mouse.Type) != string(conf.Keyboard.Type) ||
		conf.Mouse.Src != conf.Keyboard.Src ||
		conf.Mouse.Ext != conf.Keyboard.Ext {
		switch conf.Mouse.Type {
		case MouseNone:
			log.Println(Tag, "mouse driver is none, no mouse output")
		case MouseSerialPort:
			log.Println(Tag, "mouse driver is serial port:", conf.Mouse.Src)
			baud, err := conf.Mouse.Ext.GetInt("baud", 9600)
			if err != nil {
				return nil, err
			}
			md = serialport.New(conf.Mouse.Src, baud)
		}

		if md != nil {
			err = md.Open()
			if err != nil {
				log.Println(Tag, "open mouse driver:", err)
				//return nil, err
			}
		}
	} else {
		log.Println(Tag, "mouse driver is same as keyboard driver")
		md = kd
	}

	return md, err
}

func VideoCodecFromConfig(conf Config) (codec.Codec, error) {
	return &tight.JPEGEncoder{Quality: conf.Video.Quality}, nil
}
