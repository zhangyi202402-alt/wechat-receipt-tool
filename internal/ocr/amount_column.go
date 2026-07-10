package ocr

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"image/png"

	"golang.org/x/image/draw"
)

// cropAndScaleAmountColumn returns PNG bytes of the right-hand amount column, scaled up.
func cropAndScaleAmountColumn(img image.Image, startRatio, scale float64) ([]byte, float64, error) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil, 0, nil
	}
	if startRatio < 0 {
		startRatio = 0
	}
	if startRatio > 0.85 {
		startRatio = 0.85
	}
	if scale < 1 {
		scale = 1
	}

	cropX := int(float64(w) * startRatio)
	if cropX >= w-1 {
		cropX = w / 2
	}
	sub := img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(image.Rect(bounds.Min.X+cropX, bounds.Min.Y, bounds.Max.X, bounds.Max.Y))

	if scale <= 1.01 {
		data, err := encodePNG(sub)
		return data, 1, err
	}

	newW := int(float64(w-cropX) * scale)
	newH := int(float64(h) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), sub, sub.Bounds(), draw.Over, nil)
	data, err := encodePNG(dst)
	return data, scale, err
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func mapAmountColumnBoxes(boxes []TextBox, cropX, scale float64) {
	for i := range boxes {
		boxes[i].AmountColumn = true
		for j := range boxes[i].Box {
			boxes[i].Box[j][0] = boxes[i].Box[j][0]/scale + cropX
			boxes[i].Box[j][1] = boxes[i].Box[j][1] / scale
		}
	}
}

func decodeImage(data []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}
