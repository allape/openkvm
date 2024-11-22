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
		config.NewShellCommand("[\"ffmpeg\", \"-f\", \"avfoundation\", \"-framerate\", \"30\", \"-pixel_format\", \"uyvy422\", \"-video_size\", \"1280x720\", \"-i\", \"default\", \"-f\", \"mjpeg\", \"-\"]"),
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

	for i := 0; i < 1000; i++ {
		frame, changed, err := driver.GetFrame()
		if err != nil {
			t.Fatal(err)
		}

		if frame == nil {
			i -= 1
			t.Logf("frame is not ready yet, wait for 3 seconds")
			time.Sleep(3 * time.Second)
			continue
		}

		if !changed {
			continue
		}

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
