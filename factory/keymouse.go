package factory

import (
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/keymouse"
	"github.com/allape/openkvm/kvm/keymouse/serialport"
)

const DefaultBaud = 9600

func KeyboardFromConfig(conf config.Config) (kd keymouse.Driver, err error) {
	switch conf.Keyboard.Type {
	case config.KeyboardNone:
		l.Warn().Println("keyboard driver is none, no keyboard output")
	case config.KeyboardSerialPort:
		l.Info().Println("keyboard driver is serial port:", conf.Keyboard.Src)
		baud, err := conf.Keyboard.Ext.GetBaud(DefaultBaud)
		if err != nil {
			return nil, err
		}
		kd = serialport.New(conf.Keyboard.Src, baud)
	}

	if kd != nil {
		err = kd.Open()
		if err != nil {
			l.Error().Println("open keyboard driver:", err)
			//return kd, err
		}
	}

	return kd, nil
}

func MouseFromConfigOrUseKeyboard(kd keymouse.Driver, conf config.Config) (md keymouse.Driver, err error) {
	if string(conf.Mouse.Type) != string(conf.Keyboard.Type) ||
		conf.Mouse.Src != conf.Keyboard.Src {
		switch conf.Mouse.Type {
		case config.MouseNone:
			l.Warn().Println("mouse driver is none, no mouse output")
		case config.MouseSerialPort:
			l.Info().Println("mouse driver is serial port:", conf.Mouse.Src)
			baud, err := conf.Mouse.Ext.GetBaud(DefaultBaud)
			if err != nil {
				return nil, err
			}
			md = serialport.New(conf.Mouse.Src, baud)
		}

		if md != nil {
			err = md.Open()
			if err != nil {
				l.Error().Println("open mouse driver:", err)
				//return nil, err
			}
		}
	} else {
		l.Info().Println("mouse driver is same as keyboard driver")
		md = kd
	}

	return md, err
}

func KeymouseSerialDriverFromConfig(conf config.Config, keyboard keymouse.Driver, mouse keymouse.Driver, name, src string, ext config.SerialPortExt) (keymouse.Driver, error) {
	if conf.Keyboard.Type == config.KeyboardSerialPort && src == conf.Keyboard.Src {
		l.Info().Printf("%s driver is the same as keyboard driver", name)
		return keyboard, nil
	}

	if conf.Mouse.Type == config.MouseSerialPort && src == conf.Mouse.Src {
		l.Info().Printf("%s driver is the same as mouse driver", name)
		return mouse, nil
	}

	l.Info().Printf("%s driver is serial port: %s", name, src)

	baud, err := ext.GetBaud(DefaultBaud)
	if err != nil {
		return nil, err
	}

	return serialport.New(src, baud), err
}
