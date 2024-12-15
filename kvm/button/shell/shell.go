package shell

import (
	"errors"
	"github.com/allape/gogger"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/button"
	"os/exec"
)

var l = gogger.New("button.shell")

type Button struct {
	button.Driver
	Config    config.Button
	Commander config.ButtonShellExt
}

func (b *Button) Exec(cmd *exec.Cmd) error {
	bs, err := cmd.CombinedOutput()

	output := string(bs)

	l.Verbose().Println("executing command:", cmd.String())
	l.Verbose().Println("output:", output)

	if err != nil {
		return errors.New(output)
	}

	return nil
}

func (b *Button) GetButton(t button.Type) string {
	switch t {
	case button.PowerButton:
		return b.Config.PowerButton
	case button.ResetButton:
		return b.Config.ResetButton
	case button.ExtraButton:
		return b.Config.ExtraButton
	}
	return ""
}

func (b *Button) Open() error {
	buttons := map[string]string{
		"power": b.Config.PowerButton,
		"reset": b.Config.ResetButton,
		"extra": b.Config.ExtraButton,
	}

	for name, btn := range buttons {
		if btn == "" {
			if name != "extra" {
				return errors.New(name + " button not found")
			}
			continue
		}

		cmd, err := b.Commander.GetOpenCommand(btn)
		if err != nil {
			return err
		}
		err = b.Exec(cmd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Button) Close() error {
	return nil
}

func (b *Button) Press(t button.Type) error {
	cmd, err := b.Commander.GetPressCommand(b.GetButton(t))
	if err != nil {
		return err
	}
	err = b.Exec(cmd)
	if err != nil {
		return err
	}
	return nil
}

func (b *Button) Release(t button.Type) error {
	cmd, err := b.Commander.GetReleaseCommand(b.GetButton(t))
	if err != nil {
		return err
	}
	err = b.Exec(cmd)
	if err != nil {
		return err
	}
	return nil
}
