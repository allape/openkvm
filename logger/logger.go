package logger

import (
	"github.com/allape/openkvm/envar"
	"io"
	"log"
	"os"
)

func New(prefix string) *log.Logger {
	return log.New(os.Stdout, prefix+" ", log.LstdFlags)
}

func NewVerboseLogger(prefix string) *log.Logger {
	if envar.Getenv(envar.OpenkvmVerbose, "") != "" {
		return New(prefix)
	}
	return log.New(io.Discard, "", 0)
}
