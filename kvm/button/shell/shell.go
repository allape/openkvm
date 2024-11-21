package shell

import (
	"errors"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/button"
)

type Button struct {
	button.Driver
	Config config.Button

	openCommand    config.ShellCommand
	pressCommand   config.ShellCommand
	releaseCommand config.ShellCommand
}

func (b *Button) Exec(command config.ShellCommand, btn string) error {
	if btn == "" {
		return errors.New("button not found")
	}

	cmd, err := command.ToCommand()
	if err != nil {
		return err
	}

	for i, segment := range cmd.Args {
		if segment == "$PIN" {
			cmd.Args[i] = btn
		}
	}

	bs, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(bs))
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
	var err error

	openCommand := b.Config.Ext.Get("open")
	if openCommand == "" {
		return errors.New("open command not found")
	}
	pressCommand := b.Config.Ext.Get("press")
	if pressCommand == "" {
		return errors.New("press command not found")
	}
	releaseCommand := b.Config.Ext.Get("release")
	if releaseCommand == "" {
		return errors.New("release command not found")
	}

	b.openCommand = config.NewShellCommand(openCommand)
	b.pressCommand = config.NewShellCommand(pressCommand)
	b.releaseCommand = config.NewShellCommand(releaseCommand)

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
		err = b.Exec(b.openCommand, btn)
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
	err := b.Exec(b.pressCommand, b.GetButton(t))
	if err != nil {
		return err
	}
	return nil
}

func (b *Button) Release(t button.Type) error {
	err := b.Exec(b.releaseCommand, b.GetButton(t))
	if err != nil {
		return err
	}
	return nil
}
