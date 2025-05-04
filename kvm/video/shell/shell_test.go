package shell

import (
	"fmt"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm/video"
	"image/jpeg"
	"os"
	"path"
	"testing"
	"time"
)

const TestData = "testdata"

func TestDriver(t *testing.T) {
	err := os.MkdirAll(TestData, 0755)
	if err != nil {
		t.Fatal(err)
	}

	driver := NewDriver(
		[]string{
			"ffmpeg",
			"-f",
			"avfoundation",
			"-framerate",
			"30",
			"-pixel_format",
			"uyvy422",
			"-video_size",
			"1280x720",
			"-i",
			"default",
			"-f",
			"mjpeg",
			"-",
		},
		&Options{
			Options: video.Options{
				Width:     1280,
				Height:    720,
				FrameRate: 30,
			},
		},
	)

	err = driver.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = driver.Close()
	}()

	if driver.GetFrameRate() != 30 {
		t.Fatalf("Expected 30, got %f", driver.GetFrameRate())
	}

	var lastFrame config.Frame

	for i := 0; i < 10; i++ {
		frame, err := driver.NextFrame()
		if err != nil {
			t.Fatal(err)
		}

		if frame == lastFrame {
			time.Sleep(35 * time.Millisecond)
			continue
		}

		lastFrame = frame

		err = func() error {
			file, err := os.Create(path.Join(TestData, fmt.Sprintf("frame.%04d.jpg", i)))
			if err != nil {
				return err
			}
			defer func() {
				_ = file.Close()
			}()

			err = jpeg.Encode(file, frame, &jpeg.Options{Quality: 50})
			if err != nil {
				return err
			}

			return nil
		}()

		time.Sleep(35 * time.Millisecond)
	}
}
