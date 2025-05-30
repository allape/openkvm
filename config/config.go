package config

import (
	"github.com/allape/gogger"
	"github.com/pelletier/go-toml/v2"
	"os"
)

var l = gogger.New("config")

const DefaultConfigPath = "kvm.toml"

type VideoDriverType string

const (
	VideoUSBDevice   VideoDriverType = "usb"
	VideoShellDevice VideoDriverType = "shell"
	VideoDummyDevice VideoDriverType = "dummy"
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
)

type ClipboardDriverType string

const (
	ClipboardNone       ClipboardDriverType = "none"
	ClipboardSerialPort ClipboardDriverType = "serialport"
)

type Websocket struct {
	Addr    string `toml:"addr"`
	Path    string `toml:"path"`
	Cors    bool   `toml:"cors"`
	Timeout int    `toml:"timeout"` // in sec
}

type Video struct {
	SetupCommands []SetupCommand  `toml:"setup_commands"`
	Type          VideoDriverType `toml:"type"`
	Src           VideoSrc        `toml:"src"`
	Width         int             `toml:"width"`
	Height        int             `toml:"height"`
	FrameRate     float64         `toml:"frame_rate"`
	Quality       int             `toml:"quality"`
	SliceCount    SliceCount      `toml:"slice_count"`
	Ext           ExtMap          `toml:"ext"`
}

type Keyboard struct {
	Type KeyboardDriverType `toml:"type"`
	Src  string             `toml:"src"`
	Ext  SerialPortExt      `toml:"ext"`
}

type Mouse struct {
	Type MouseDriverType `toml:"type"`
	Src  string          `toml:"src"`
	Ext  SerialPortExt   `toml:"ext"`

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
	Ext         ExtMap           `toml:"ext"`
	PowerButton string           `toml:"pwr_btn"`
	ResetButton string           `toml:"rst_btn"`
	ExtraButton string           `toml:"ext_btn"`
}

type Clipboard struct {
	Type ClipboardDriverType `toml:"type"`
	Src  string              `toml:"src"`
	Ext  SerialPortExt       `toml:"ext"`
}

type VNC struct {
	Path     string `toml:"path"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

type Config struct {
	Websocket Websocket `toml:"websocket"`
	Video     Video     `toml:"video"`
	Keyboard  Keyboard  `toml:"keyboard"`
	Mouse     Mouse     `toml:"mouse"`
	Button    Button    `toml:"button"`
	Clipboard Clipboard `toml:"clipboard"`
	VNC       VNC       `toml:"vnc"`
}

func GetConfig() (Config, error) {
	configFile := DefaultConfigPath
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	l.Info().Println("reading config file:", configFile)

	config := Config{
		Websocket: Websocket{
			Addr:    ":8080",
			Path:    "/websockify",
			Timeout: 30,
		},
		Keyboard: Keyboard{
			Type: KeyboardNone,
		},
		Video: Video{
			Type:      "error",
			Width:     1280,
			Height:    720,
			FrameRate: 15,
			Quality:   100,
		},
		Mouse: Mouse{
			Type: MouseNone,
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

	l.Debug().Println("use config:", config)

	return config, nil
}
