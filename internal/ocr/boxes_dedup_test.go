package ocr

import "testing"

func TestDeduplicateBoxes(t *testing.T) {
	boxes := []TextBox{
		{Text: "+300.00", Box: [4][2]float64{{350, 380}, {430, 380}, {430, 405}, {350, 405}}, Score: 0.99},
		{Text: "+300.00", Box: [4][2]float64{{350, 380}, {430, 380}, {430, 405}, {350, 405}}, Score: 0.95},
		{Text: "+300.00", Box: [4][2]float64{{350, 380}, {430, 380}, {430, 405}, {350, 405}}, Score: 0.98, AmountColumn: true},
	}
	out := DeduplicateBoxes(boxes)
	if len(out) != 2 {
		t.Fatalf("expected 2 boxes (full + amount column), got %d", len(out))
	}
}

func TestBoxNearDuplicate(t *testing.T) {
	a := TextBox{Text: "账单", Box: [4][2]float64{{100, 50}, {200, 50}, {200, 70}, {100, 70}}}
	b := TextBox{Text: "账单", Box: [4][2]float64{{102, 51}, {202, 51}, {202, 71}, {102, 71}}}
	if !boxNearDuplicate(a, b) {
		t.Fatal("expected near duplicate")
	}
}
