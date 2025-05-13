package factory

import (
	"fmt"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/button"
	"github.com/allape/openkvm/kvm/button/serialport"
	"github.com/allape/openkvm/kvm/button/shell"
	"github.com/allape/openkvm/kvm/keymouse"
)

func ButtonFromConfig(conf config.Config, keyboard keymouse.Driver, mouse keymouse.Driver) (bd button.Driver, err error) {
	switch conf.Button.Type {
	case config.ButtonNone:
		l.Warn().Println("button driver is none, no button output")
		return nil, err
	case config.ButtonSerialPort:
		km, err := KeymouseSerialDriverFromConfig(
			conf, keyboard, mouse,
			"button", conf.Button.Src, config.SerialPortExt(conf.Button.Ext),
		)
		if err != nil {
			return nil, err
		}

		bd = &serialport.Button{
			Config:        conf.Button,
			KeyboardMouse: km,
		}
	case config.ButtonShell:
		l.Info().Println("button driver is shell:", conf.Button.Ext)
		bd = &shell.Button{
			Config:    conf.Button,
			Commander: config.ButtonShellExt(conf.Button.Ext),
		}
	default:
		return nil, fmt.Errorf("unknown button driver: %s", conf.Button.Type)
	}

	err = bd.Open()
	if err != nil {
		return nil, err
	}

	return bd, err
}
