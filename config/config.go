package config

import (
	"github.com/allape/openkvm/config/tag"
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
	Addr string `toml:"addr"`
	Path string `toml:"path"`
	Cors bool   `toml:"cors"`
}

type Video struct {
	PreludeCommand tag.PreludeCommand `toml:"prelude_command"`
	Width          int                `toml:"width"`
	Height         int                `toml:"height"`
	Type           VideoDriverType    `toml:"type"`
	Src            string             `toml:"src"`
	FrameRate      float64            `toml:"frame_rate"`
	Quality        int                `toml:"quality"`
	FlipCode       video.FlipCode     `toml:"flip_code"`
	SliceCount     video.SliceCount   `toml:"slice_count"`
	Ext            tag.TagString      `toml:"ext"`
}

type Keyboard struct {
	Type KeyboardDriverType `toml:"type"`
	Src  string             `toml:"src"`
	Ext  tag.TagString      `toml:"ext"`
}

type Mouse struct {
	Type MouseDriverType `toml:"type"`
	Src  string          `toml:"src"`
	Ext  tag.TagString   `toml:"ext"`
}

type VNC struct {
	Path string `toml:"path"`
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
			Width:          1920,
			Height:         1080,
			Type:           VideoUSBDevice,
			Src:            "0",
			FlipCode:       video.NoFlip,
			FrameRate:      30,
			SliceCount:     4,
			Ext:            "",
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
