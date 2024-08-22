package serialport

import (
	"errors"
	"fmt"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/button"
	"github.com/allape/openkvm/kvm/keymouse"
	"strconv"
)

type Button struct {
	button.Driver
	Config config.Button

	powerButtonPin byte
	resetButtonPin byte
	extraButtonPin byte

	km keymouse.Driver
}

func (b *Button) GetButton(t button.Type) byte {
	switch t {
	case button.PowerButton:
		return b.powerButtonPin
	case button.ResetButton:
		return b.resetButtonPin
	case button.ExtraButton:
		return b.extraButtonPin
	}
	return 0
}

func (b *Button) Open() error {
	var err error

	if b.Config.PowerButton == "" {
		return errors.New("power button not found")
	}
	if b.Config.ResetButton == "" {
		return errors.New("reset button not found")
	}

	powerButtonPin, err := strconv.Atoi(b.Config.PowerButton)
	if err != nil {
		return err
	}
	resetButtonPin, err := strconv.Atoi(b.Config.ResetButton)
	if err != nil {
		return err
	}

	extraButtonPin := 0
	if b.Config.ExtraButton != "" {
		extraButtonPin, err = strconv.Atoi(b.Config.ExtraButton)
		if err != nil {
			return err
		}
	}

	b.powerButtonPin = byte(powerButtonPin)
	b.resetButtonPin = byte(resetButtonPin)
	b.extraButtonPin = byte(extraButtonPin)

	buttons := map[string]byte{
		"power": b.powerButtonPin,
		"reset": b.resetButtonPin,
		"extra": b.extraButtonPin,
	}

	for name, btn := range buttons {
		if btn == 0 {
			continue
		}
		_, err = b.km.Write([]byte{
			0xff,
			0x01,
			btn,
			0x00,
		})
		if err != nil {
			return fmt.Errorf("open %s button error: %w", name, err)
		}
	}

	return nil
}

func (b *Button) Close() error {
	return nil
}

func (b *Button) Press(t button.Type) error {
	_, err := b.km.Write([]byte{
		0xff,
		0x02,
		b.GetButton(t),
		0x01,
	})
	return err
}

func (b *Button) Release(t button.Type) error {
	_, err := b.km.Write([]byte{
		0xff,
		0x02,
		b.GetButton(t),
		0x00,
	})
	return err
}
