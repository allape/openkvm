package config

import (
	"encoding/hex"
	"encoding/json"
	"os/exec"
	"reflect"
	"strconv"
)

type TagString reflect.StructTag

func (d TagString) GetInt(key string, defaultValue int) (int, error) {
	value := reflect.StructTag(d).Get(key)
	if value == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(value)
}

func (d TagString) Get(key string) string {
	return reflect.StructTag(d).Get(key)
}

// ShellCommand
// A command run before capturing the video device
// Example: ["ls", "-al"]
type ShellCommand struct {
	decoded []string

	Command string
}

func (d ShellCommand) ToStringArray() ([]string, error) {
	if d.Command == "" {
		return nil, nil
	}

	if d.decoded != nil {
		return d.decoded, nil
	}

	var args []string

	err := json.Unmarshal([]byte(d.Command), &args)
	if err != nil {
		return nil, err
	}

	d.decoded = args

	return args, nil
}

func (d ShellCommand) ToCommandParams() (string, []string, error) {
	args, err := d.ToStringArray()
	if err != nil {
		return "", nil, err
	}

	if len(args) == 0 {
		return "", nil, nil
	}

	return args[0], args[1:], nil
}

func (d ShellCommand) ToCommand() (*exec.Cmd, error) {
	cmd, args, err := d.ToCommandParams()
	if err != nil {
		return nil, err
	}

	if cmd == "" {
		return nil, nil
	}

	return exec.Command(cmd, args...), nil
}

func NewShellCommand(cmd string) ShellCommand {
	return ShellCommand{Command: cmd}
}

type HexStringMarker string

func (d HexStringMarker) ToByteArray() ([]byte, error) {
	if d == "" {
		return nil, nil
	}
	return hex.DecodeString(string(d))
}
