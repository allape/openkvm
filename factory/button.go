package factory

import (
	"fmt"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/button"
	"github.com/allape/openkvm/kvm/button/gpio"
	"github.com/allape/openkvm/kvm/button/serialport"
	"github.com/allape/openkvm/kvm/button/shell"
)

func ButtonFromConfig(conf config.Config) (bd button.Driver, err error) {
	switch conf.Button.Type {
	case config.ButtonNone:
		return nil, err
	case config.ButtonSerialPort:
		bd = &serialport.Button{
			Config: conf.Button,
		}
	case config.ButtonShell:
		bd = &shell.Button{
			Config: conf.Button,
		}
	case config.ButtonGPIO:
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
