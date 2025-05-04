package codec

import (
	"github.com/allape/openkvm/config"
)

type Codec interface {
	FramebufferUpdate(previewFrame, nextFrame config.Frame) ([]byte, error)
}
