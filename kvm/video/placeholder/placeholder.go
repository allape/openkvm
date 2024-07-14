package placeholder

import (
	_ "embed"
	"github.com/allape/openkvm/config"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"image/color"
	"time"
)

//go:embed Roboto-Regular.ttf
var FontBytes []byte

var Font *truetype.Font

// CreatePlaceholder
// create a placeholder image when failed to capture the frame
func CreatePlaceholder(
	width, height int,
	backgroundColor, color color.Color,
	text string,
	timestamp bool, // put current time in YYYY-MM-dd HH:mm:ss pattern at the right bottom corner
) (config.Frame, error) {
	dc := gg.NewContext(width, height)
	dc.SetColor(backgroundColor)
	dc.DrawRectangle(0, 0, float64(width), float64(height))
	dc.Fill()

	if Font == nil {
		var err error
		Font, err = truetype.Parse(FontBytes)
		if err != nil {
			return nil, err
		}
	}
	dc.SetFontFace(truetype.NewFace(Font, &truetype.Options{Size: 120}))

	dc.SetColor(color)
	dc.DrawStringAnchored(text, float64(width/2), float64(height/2), 0.5, 0.5)

	if timestamp {
		nowStr := time.Now().Format(time.DateTime)
		dc.SetFontFace(truetype.NewFace(Font, &truetype.Options{Size: 32}))
		dc.DrawStringAnchored(nowStr, float64(width-50), float64(height-50), 1, 0)
	}

	return dc.Image(), nil
}
