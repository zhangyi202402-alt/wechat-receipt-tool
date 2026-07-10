package ocr

import (
	"image"
	"image/color"
	"testing"
)

func TestCropAndScaleAmountColumn(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 50))
	for x := 60; x < 100; x++ {
		for y := 10; y < 20; y++ {
			img.Set(x, y, color.White)
		}
	}
	data, scale, err := cropAndScaleAmountColumn(img, 0.55, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty png")
	}
	if scale != 2 {
		t.Fatalf("scale: got %v", scale)
	}
}

func TestMapAmountColumnBoxes(t *testing.T) {
	boxes := []TextBox{{Text: "+100.00", Box: [4][2]float64{{10, 20}, {50, 20}, {50, 30}, {10, 30}}}}
	mapAmountColumnBoxes(boxes, 200, 2)
	if !boxes[0].AmountColumn {
		t.Fatal("expected AmountColumn")
	}
	if boxes[0].Box[0][0] != 205 {
		t.Fatalf("x offset: got %v want 205", boxes[0].Box[0][0])
	}
	if boxes[0].Box[0][1] != 10 {
		t.Fatalf("y scale: got %v want 10", boxes[0].Box[0][1])
	}
}
