package gpio

import (
	"errors"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/button"
)

type Button struct {
	button.Driver
	Config config.Button
}

func (b *Button) Open() error {
	return errors.New("not implemented")
}

func (b *Button) Close() error {
	return errors.New("not implemented")
}

func (b *Button) Press(_ button.Type) error {
	return errors.New("not implemented")
}

func (b *Button) Release(_ button.Type) error {
	return errors.New("not implemented")
}
