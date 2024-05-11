package codec

import (
	"github.com/allape/openkvm/kvm/video"
)

type Codec interface {
	FramebufferUpdate(rects []video.Rect) ([]byte, error)
}
