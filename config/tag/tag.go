package tag

import (
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
