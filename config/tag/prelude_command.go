package tag

import (
	"reflect"
	"strings"
)

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
