package logger

import (
	"github.com/allape/openkvm/envar"
	"io"
	"log"
	"os"
)

var verbose = envar.Getenv(envar.OpenkvmVerbose, "") != ""

func init() {
	if verbose {
		log.Println("[logger] verbose mode enabled")
	}
}

func New(prefix string) *log.Logger {
	return log.New(os.Stdout, prefix+" ", log.LstdFlags)
}

func NewVerboseLogger(prefix string) *log.Logger {
	if verbose {
		return New(prefix)
	}
	return log.New(io.Discard, "", 0)
}
