package config

import (
	"github.com/allape/openkvm/logger"
	"github.com/pelletier/go-toml/v2"
	"os"
)

var log = logger.New("[config]")

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

type ButtonDriverType string

const (
	ButtonNone       ButtonDriverType = "none"
	ButtonSerialPort ButtonDriverType = "serialport"
	ButtonShell      ButtonDriverType = "shell"
	ButtonGPIO       ButtonDriverType = "gpio" // TODO not supported yet
)

type Websocket struct {
	Addr string `toml:"addr"`
	Path string `toml:"path"`
	Cors bool   `toml:"cors"`
}

type Video struct {
	PreludeCommand PreludeCommand  `toml:"prelude_command"`
	Width          int             `toml:"width"`
	Height         int             `toml:"height"`
	Type           VideoDriverType `toml:"type"`
	Src            string          `toml:"src"`
	FrameRate      float64         `toml:"frame_rate"`
	Quality        int             `toml:"quality"`
	FlipCode       FlipCode        `toml:"flip_code"`
	SliceCount     SliceCount      `toml:"slice_count"`
	Ext            TagString       `toml:"ext"`
}

type Keyboard struct {
	Type KeyboardDriverType `toml:"type"`
	Src  string             `toml:"src"`
	Ext  TagString          `toml:"ext"`
}

type Mouse struct {
	Type MouseDriverType `toml:"type"`
	Src  string          `toml:"src"`
	Ext  TagString       `toml:"ext"`

	// CursorXScale
	// A factor to adjust the cursor move distance when the video is scaled.
	// Example:
	//  If the VNC cursor move distance is 10 pixels, and the CursorXScale is 0.5, the actual cursor will move 5 pixel.
	CursorXScale float64 `toml:"cursor_x_scale"`
	// CursorYScale: see CursorXScale
	CursorYScale float64 `toml:"cursor_y_scale"`
}

type Button struct {
	Type        ButtonDriverType `toml:"type"`
	Src         string           `toml:"src"`
	Ext         TagString        `toml:"ext"`
	PowerButton string           `toml:"pwr_btn"`
	ResetButton string           `toml:"rst_btn"`
	ExtraButton string           `toml:"ext_btn"`
}

type VNC struct {
	Path string `toml:"path"`
}

type Config struct {
	Websocket Websocket `toml:"websocket"`
	Video     Video     `toml:"video"`
	Keyboard  Keyboard  `toml:"keyboard"`
	Mouse     Mouse     `toml:"mouse"`
	Button    Button    `toml:"button"`
	VNC       VNC       `toml:"vnc"`
}

func GetConfig() (Config, error) {
	configFile := DefaultConfigPath
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	log.Println("reading config file:", configFile)

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
			FlipCode:       NoFlip,
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
		Button: Button{
			Type: ButtonNone,
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

	log.Println("use config:", config)

	return config, nil
}
