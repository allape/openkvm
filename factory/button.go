package factory

import (
	"fmt"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/button"
	"github.com/allape/openkvm/kvm/button/serialport"
	"github.com/allape/openkvm/kvm/button/shell"
	"github.com/allape/openkvm/kvm/keymouse"
	keymouseSP "github.com/allape/openkvm/kvm/keymouse/serialport"
)

func ButtonFromConfig(conf config.Config, keyboard keymouse.Driver, mouse keymouse.Driver) (bd button.Driver, err error) {
	switch conf.Button.Type {
	case config.ButtonNone:
		l.Warn().Println("button driver is none, no button output")
		return nil, err
	case config.ButtonSerialPort:
		var km keymouse.Driver

		switch conf.Button.Src {
		case conf.Keyboard.Src:
			l.Info().Println("button driver is the same as keyboard driver")
			km = keyboard
		case conf.Mouse.Src:
			l.Info().Println("button driver is the same as mouse driver")
			km = mouse
		default:
			l.Info().Println("button driver is serial port:", conf.Button.Src)
			baud, err := conf.Button.Ext.GetInt("baud", 9600)
			if err != nil {
				return nil, err
			}
			km = keymouseSP.New(conf.Button.Src, baud)
		}

		bd = &serialport.Button{
			Config:        conf.Button,
			KeyboardMouse: km,
		}
	case config.ButtonShell:
		l.Info().Println("button driver is shell:", conf.Button.Ext)
		bd = &shell.Button{
			Config: conf.Button,
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
