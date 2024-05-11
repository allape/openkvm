package placeholder

import (
	"image/color"
	"image/jpeg"
	"os"
	"testing"
)

func TestCreatePlaceholder(t *testing.T) {
	img, err := CreatePlaceholder(
		1920, 1080,
		color.RGBA{A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
		"Hello, World!",
		true,
	)
	if err != nil {
		t.Fatal(err)
	}

	frameFile, err := os.Create("frame.jpg")
	if err != nil {
		t.Fatal(err)
	}

	err = jpeg.Encode(frameFile, img, nil)
	if err != nil {
		t.Fatal(err)
	}
}
