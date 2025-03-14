package config

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
)

type VideoSrc []string

func (s VideoSrc) Empty() bool {
	return len(s) == 0
}

type VideoShellSrc VideoSrc

func (s VideoShellSrc) Empty() bool {
	return VideoSrc(s).Empty()
}

func (s VideoShellSrc) ToCommand() (*exec.Cmd, error) {
	if len(s) == 0 {
		return nil, nil
	}

	return exec.Command(s[0], s[1:]...), nil
}

type SetupCommand []string

func (s SetupCommand) ToCommand() (*exec.Cmd, error) {
	if len(s) == 0 {
		return nil, nil
	}

	return exec.Command(s[0], s[1:]...), nil
}

type ExtMap map[string]any

type SerialPortExt ExtMap

func (e SerialPortExt) GetBaud(defaultValue int) (int, error) {
	v, ok := e["baud"]
	if !ok {
		return defaultValue, nil
	}

	baud, ok := v.(string)
	if !ok {
		return defaultValue, nil
	}

	return strconv.Atoi(baud)
}

type ButtonShellExt ExtMap

func (e ButtonShellExt) GetCommand(fieldName, pin string) (*exec.Cmd, error) {
	if pin == "" {
		return nil, errors.New("pin is empty")
	}

	v, ok := e[fieldName]
	if !ok {
		return nil, nil
	}

	anyStrings, ok := v.([]any)
	if !ok {
		return nil, nil
	}

	cmd := make([]string, len(anyStrings))
	for i, v := range anyStrings {
		cmd[i] = fmt.Sprintf("%v", v)
		if cmd[i] == "$PIN" {
			cmd[i] = pin
		}
	}

	return exec.Command(cmd[0], cmd[1:]...), nil
}

func (e ButtonShellExt) GetOpenCommand(pin string) (*exec.Cmd, error) {
	return e.GetCommand("open", pin)
}

func (e ButtonShellExt) GetPressCommand(pin string) (*exec.Cmd, error) {
	return e.GetCommand("press", pin)
}

func (e ButtonShellExt) GetReleaseCommand(pin string) (*exec.Cmd, error) {
	return e.GetCommand("release", pin)
}
