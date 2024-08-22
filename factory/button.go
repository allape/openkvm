package factory

import (
	"fmt"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/button"
	"github.com/allape/openkvm/kvm/button/gpio"
	"github.com/allape/openkvm/kvm/button/serialport"
	"github.com/allape/openkvm/kvm/button/shell"
	"github.com/allape/openkvm/kvm/keymouse"
	serialport2 "github.com/allape/openkvm/kvm/keymouse/serialport"
)

func ButtonFromConfig(conf config.Config, keyboard keymouse.Driver, mouse keymouse.Driver) (bd button.Driver, err error) {
	switch conf.Button.Type {
	case config.ButtonNone:
		log.Println("button driver is none, no button output")
		return nil, err
	case config.ButtonSerialPort:
		log.Println("button driver is serial port:", conf.Button.Src)
		var km keymouse.Driver

		switch conf.Button.Src {
		case conf.Keyboard.Src:
			km = keyboard
		case conf.Mouse.Src:
			km = mouse
		default:
			baud, err := conf.Button.Ext.GetInt("baud", 9600)
			if err != nil {
				return nil, err
			}
			km = serialport2.New(conf.Button.Src, baud)
		}

		bd = &serialport.Button{
			Config:        conf.Button,
			KeyboardMouse: km,
		}
	case config.ButtonShell:
		log.Println("button driver is shell:", conf.Button.Ext)
		bd = &shell.Button{
			Config: conf.Button,
		}
	case config.ButtonGPIO:
		log.Println("button driver is gpio")
		bd = &gpio.Button{
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
