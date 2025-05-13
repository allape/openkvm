package factory

import (
	"fmt"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/clipboard"
	"github.com/allape/openkvm/kvm/clipboard/serialport"
	"github.com/allape/openkvm/kvm/keymouse"
)

func ClipboardFromConfig(conf config.Config, keyboard keymouse.Driver, mouse keymouse.Driver) (cd clipboard.Driver, err error) {
	switch conf.Clipboard.Type {
	case config.ClipboardNone:
		l.Warn().Println("clipboard driver is none, no clipboard support")
		return nil, err
	case config.ClipboardSerialPort:
		km, err := KeymouseSerialDriverFromConfig(
			conf, keyboard, mouse,
			"clipboard", conf.Clipboard.Src, conf.Clipboard.Ext,
		)
		if err != nil {
			return nil, err
		}

		cd = &serialport.Clipboard{
			Config:        conf.Clipboard,
			KeyboardMouse: km,
		}
	default:
		return nil, fmt.Errorf("unknown clipboard driver: %s", conf.Clipboard.Type)
	}

	err = cd.Open()
	if err != nil {
		return nil, err
	}

	return cd, err
}
