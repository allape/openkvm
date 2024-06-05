package factory

import (
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/keymouse"
	"github.com/allape/openkvm/kvm/keymouse/serialport"
)

func KeyboardFromConfig(conf config.Config) (kd keymouse.KeyboardMouseDriver, err error) {
	switch conf.Keyboard.Type {
	case config.KeyboardNone:
		log.Println("keyboard driver is none, no keyboard output")
	case config.KeyboardSerialPort:
		log.Println("keyboard driver is serial port:", conf.Keyboard.Src)
		baud, err := conf.Mouse.Ext.GetInt("baud", 9600)
		if err != nil {
			return nil, err
		}
		kd = serialport.New(conf.Keyboard.Src, baud)
	}

	if kd != nil {
		err = kd.Open()
		if err != nil {
			log.Println("open keyboard driver:", err)
			//return kd, err
		}
	}

	return kd, nil
}

func MouseFromConfigOrUseKeyboard(kd keymouse.KeyboardMouseDriver, conf config.Config) (md keymouse.KeyboardMouseDriver, err error) {
	if string(conf.Mouse.Type) != string(conf.Keyboard.Type) ||
		conf.Mouse.Src != conf.Keyboard.Src ||
		conf.Mouse.Ext != conf.Keyboard.Ext {
		switch conf.Mouse.Type {
		case config.MouseNone:
			log.Println("mouse driver is none, no mouse output")
		case config.MouseSerialPort:
			log.Println("mouse driver is serial port:", conf.Mouse.Src)
			baud, err := conf.Mouse.Ext.GetInt("baud", 9600)
			if err != nil {
				return nil, err
			}
			md = serialport.New(conf.Mouse.Src, baud)
		}

		if md != nil {
			err = md.Open()
			if err != nil {
				log.Println("open mouse driver:", err)
				//return nil, err
			}
		}
	} else {
		log.Println("mouse driver is same as keyboard driver")
		md = kd
	}

	return md, err
}
