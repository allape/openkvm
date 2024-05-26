package codec

import (
	"github.com/allape/openkvm/config"
)

type Codec interface {
	FramebufferUpdate(rects []config.Rect) ([]byte, error)
}
