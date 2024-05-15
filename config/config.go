package config

import (
	"github.com/allape/openkvm/config/sub"
	"github.com/allape/openkvm/kvm/video"
	"github.com/pelletier/go-toml/v2"
	"log"
	"os"
)

const Tag = "[config]"

const DefaultConfigPath = "kvm.toml"

type VideoDriverType string

const (
	VideoUSBDevice VideoDriverType = "usb"
)

type KeyboardDriverType string

const (
	KeyboardNone       KeyboardDriverType = "none"
	KeyboardSerialPort KeyboardDriverType = "serialport"
)

type MouseDriverType string

const (
	MouseNone       MouseDriverType = "none"
	MouseSerialPort MouseDriverType = "serialport"
)

type Websocket struct {
	Addr string
	Path string
	Cors bool
}

type Video struct {
	PreludeCommand sub.PreludeCommand
	Type           VideoDriverType
	Src            string
	FrameRate      float64
	Quality        int
	FlipCode       video.FlipCode
	SliceCount     video.SliceCount
	Ext            sub.TagString
}

type Keyboard struct {
	Type KeyboardDriverType
	Src  string
	Ext  sub.TagString
}

type Mouse struct {
	Type MouseDriverType
	Src  string
	Ext  sub.TagString
}

type VNC struct {
	Path string
}

type Config struct {
	Websocket Websocket `toml:"websocket"`
	Video     Video     `toml:"video"`
	Keyboard  Keyboard  `toml:"keyboard"`
	Mouse     Mouse     `toml:"mouse"`
	VNC       VNC       `toml:"vnc"`
}

func GetConfig() (Config, error) {
	configFile := DefaultConfigPath
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	log.Println(Tag, "reading config file:", configFile)

	config := Config{
		Keyboard: Keyboard{
			Type: KeyboardNone,
		},
		Video: Video{
			PreludeCommand: "",
			Type:           VideoUSBDevice,
			Src:            "0",
			FlipCode:       video.Nothing,
			FrameRate:      30,
			SliceCount:     4,
			Ext:            `placeholder:"width:1920,height:1080"`,
		},
		Mouse: Mouse{
			Type: MouseNone,
		},
		Websocket: Websocket{
			Addr: ":8080",
			Path: "/websockify",
		},
		VNC: VNC{
			Path: "",
		},
	}

	_, err := os.Stat(configFile)
	if err != nil {
		return config, err
	}

	configData, err := os.ReadFile(configFile)
	if err != nil {
		return config, err
	}

	err = toml.Unmarshal(configData, &config)
	if err != nil {
		return config, err
	}

	log.Println(Tag, "use config:", config)

	return config, nil
}
