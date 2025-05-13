package clipboard

import "io"

type Driver interface {
	io.Closer
	Open() error
	Read(buffer []byte) (int, error)
	Write(buffer []byte) (int, error)
}
