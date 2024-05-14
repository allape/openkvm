package device

import (
	"fmt"
	"github.com/allape/openkvm/kvm/video"
	"image/jpeg"
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	var err error

	device := NewDevice("0", 30, video.Vertical, nil)
	err = device.Open()
	if err != nil {
		t.Fatal(err)
	}

	if device.GetFrameRate() != 30 {
		t.Fatalf("Expected 30, got %f", device.GetFrameRate())
	}

	// should be the same bytes array
	rects, err := device.GetNextImageRects(4, true)
	if err != nil {
		t.Fatal(err)
	}
	options := &jpeg.Options{Quality: 50}
	for index, rect := range rects {
		file, err := os.Create(fmt.Sprintf("frame.%02d.jpg", index))
		if err != nil {
			t.Fatal(err)
		}
		err = jpeg.Encode(file, rect.Frame, options)
		if err != nil {
			t.Fatal(err)
		}
		err = file.Close()
		if err != nil {
			t.Fatal(err)
		}
	}

	err = device.Close()
	if err != nil {
		t.Fatal(err)
	}
}
