package envar

import "os"

const (
	OpenkvmVerbose = "OPENKVM_VERBOSE"
)

func Getenv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}
