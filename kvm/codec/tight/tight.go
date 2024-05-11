package tight

import (
	"bytes"
	"github.com/allape/openkvm/kvm/codec"
	"github.com/allape/openkvm/kvm/video"
	"image/jpeg"
)

type JPEGEncoder struct {
	codec.Codec
	Quality int
}

func (e *JPEGEncoder) FramebufferUpdate(rects []video.Rect) ([]byte, error) {
	count := len(rects)
	var payload = []byte{
		0,                             // FramebufferUpdate
		0,                             // padding
		byte(count >> 8), byte(count), // Number of rectangles
	}

	options := &jpeg.Options{Quality: e.Quality}
	if options.Quality == 0 {
		options.Quality = 75
	}

	for _, rect := range rects {
		size := rect.Frame.Bounds().Size()
		payload = append(payload, []byte{
			byte(rect.X >> 8), byte(rect.X), // x-position
			byte(rect.Y >> 8), byte(rect.Y), // y-position
			byte(size.X >> 8), byte(size.X), // width
			byte(size.Y >> 8), byte(size.Y), // height
			0, 0, 0, 7, // encoding-type, tight
			0x90, // jpeg rect
		}...)
		buffer := bytes.NewBuffer(nil)
		err := jpeg.Encode(buffer, rect.Frame, options)
		if err != nil {
			return nil, err
		}
		payload = append(payload, encodeLength(buffer.Len())...) // size
		payload = append(payload, buffer.Bytes()...)             // data
	}
	return payload, nil
}

func decodeLength(aob []byte) (int, int) {
	b := aob[0]
	consumedLength := 1
	l := uint(b & 0x7f)
	if b&0x80 != 0 {
		b = aob[1]
		l |= uint(b&0x7f) << 7
		consumedLength = 2
		if b&0x80 != 0 {
			b = aob[2]
			l |= uint(b) << 14
			consumedLength = 3
		}
	}
	return int(l), consumedLength
}

func encodeLength(length int) []byte {
	size := 1
	bs := make([]byte, 3)

	bs[0] = byte(length & 0x7f)
	if length > 0x7f {
		bs[0] |= 0x80
		bs[1] = byte((length >> 7) & 0x7f)
		size = 2
		if length > 0x3fff {
			bs[1] |= 0x80
			bs[2] = byte((length >> 14) & 0x7f)
			size = 3
		}
	}
	return bs[:size]
}
