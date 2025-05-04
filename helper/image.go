package helper

import (
	"errors"
	"fmt"
	"github.com/allape/openkvm/config"
	"image"
	"sync"
)

func ImageChanged(img1, img2 image.Image, size image.Point, offsetX, offsetY, width, height int) bool {
	rectMaxWidth := offsetX + width
	if rectMaxWidth > size.X {
		rectMaxWidth = size.X
	}
	rectMaxHeight := offsetY + height
	if rectMaxHeight > size.Y {
		rectMaxHeight = size.Y
	}

	for x := offsetX; x < rectMaxWidth; x++ {
		for y := offsetY; y < rectMaxHeight; y++ {
			r1, b1, g1, _ := img1.At(x, y).RGBA()
			r2, b2, g2, _ := img2.At(x, y).RGBA()
			if r1 != r2 || b1 != b2 || g1 != g2 {
				return true
			}
		}
	}

	return false
}

type SubImager interface {
	image.Image
	SubImage(r image.Rectangle) image.Image
}

func CalcNextImageRects(previewImage, nextImage image.Image, sliceCount config.SliceCount) ([]config.Rect, error) {
	if previewImage == nextImage {
		return nil, nil
	}

	sc := int(sliceCount)

	if nextImage == nil {
		return nil, errors.New("next image is nil")
	}

	img, ok := nextImage.(SubImager)
	if !ok {
		return nil, errors.New("image does not support sub-imaging")
	}

	imageSize := img.Bounds().Size()

	if imageSize.X%sc != 0 {
		return nil, fmt.Errorf("image width should be divisible by slice count: %d %% %d != 0", imageSize.X, sc)
	} else if imageSize.Y%sc != 0 {
		return nil, fmt.Errorf("image height should be divisible by slice count: %d %% %d != 0", imageSize.Y, sc)
	}

	rectSize := image.Point{X: imageSize.X / sc, Y: imageSize.Y / sc}

	if imageSize.X%rectSize.X != 0 {
		return nil, fmt.Errorf("image width should be divisible by rect width: %d %% %d != 0", imageSize.X, rectSize.X)
	} else if imageSize.Y%rectSize.Y != 0 {
		return nil, fmt.Errorf("image height should be divisible by rect height: %d %% %d != 0", imageSize.Y, rectSize.Y)
	}

	colCount := imageSize.X / rectSize.X
	rowCount := imageSize.Y / rectSize.Y

	rectChangedMarks := make([][]bool, colCount)
	for i := range rectChangedMarks {
		rectChangedMarks[i] = make([]bool, rowCount)
	}

	if previewImage == nil {
		for i := 0; i < colCount; i++ {
			for j := 0; j < rowCount; j++ {
				rectChangedMarks[i][j] = true
			}
		}
	} else {
		var wait sync.WaitGroup
		for i := 0; i < colCount; i++ {
			for j := 0; j < rowCount; j++ {
				wait.Add(1)
				go func(x, y int) {
					defer wait.Done()
					rectChangedMarks[x][y] = ImageChanged(previewImage, nextImage, imageSize, x*rectSize.X, y*rectSize.Y, rectSize.X, rectSize.Y)
				}(i, j)
			}
		}
		wait.Wait()
	}

	rects := make([]config.Rect, 0)
	for i := 0; i < colCount; i++ {
		for j := 0; j < rowCount; j++ {
			if !rectChangedMarks[i][j] {
				continue
			}
			rects = append(rects, config.Rect{
				X:     uint64(i * rectSize.X),
				Y:     uint64(j * rectSize.Y),
				Frame: img.SubImage(image.Rect(i*rectSize.X, j*rectSize.Y, (i+1)*rectSize.X, (j+1)*rectSize.Y)),
			})
		}
	}

	return rects, nil
}
