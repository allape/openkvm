package config

import (
	"reflect"
	"strconv"
	"strings"
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

// PreludeCommand
// A command run before capturing the video device
// Example: shell:"bash" args:"-c" cmd:"ls -al"
type PreludeCommand TagString

func (d PreludeCommand) Get() (string, []string) {
	if d == "" {
		return "", nil
	}

	tag := reflect.StructTag(d)

	shell := tag.Get("shell")
	if shell == "" {
		return "", nil
	}

	args := strings.Split(tag.Get("args"), " ")

	cmd := tag.Get("cmd")
	if cmd == "" {
		return "", nil
	}

	return shell, append(args, cmd)
}
