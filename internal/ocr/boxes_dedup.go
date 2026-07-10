package ocr

import (
	"math"
	"sort"
)

const (
	dupCenterYTol = 4.0
	dupCenterXTol = 8.0
)

// DeduplicateBoxes removes parent/child and repeated detections with same text and position.
func DeduplicateBoxes(boxes []TextBox) []TextBox {
	if len(boxes) <= 1 {
		return boxes
	}
	sorted := append([]TextBox(nil), boxes...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Score != sorted[j].Score {
			return sorted[i].Score > sorted[j].Score
		}
		yi, xi := boxCenter(sorted[i].Box)
		yj, xj := boxCenter(sorted[j].Box)
		if yi != yj {
			return yi < yj
		}
		return xi < xj
	})

	var out []TextBox
	for _, b := range sorted {
		if isDuplicateBox(out, b) {
			continue
		}
		out = append(out, b)
	}
	return out
}

func isDuplicateBox(existing []TextBox, b TextBox) bool {
	cy, cx := boxCenter(b.Box)
	for _, e := range existing {
		if e.Text != b.Text || e.AmountColumn != b.AmountColumn {
			continue
		}
		ey, ex := boxCenter(e.Box)
		if math.Abs(cy-ey) <= dupCenterYTol && math.Abs(cx-ex) <= dupCenterXTol {
			return true
		}
	}
	return false
}

func boxNearDuplicate(a, b TextBox) bool {
	if a.Text != b.Text || a.AmountColumn != b.AmountColumn {
		return false
	}
	ay, ax := boxCenter(a.Box)
	by, bx := boxCenter(b.Box)
	return math.Abs(ay-by) <= dupCenterYTol && math.Abs(ax-bx) <= dupCenterXTol
}

func boxCenter(box [4][2]float64) (float64, float64) {
	return (box[0][0] + box[2][0]) / 2, (box[0][1] + box[2][1]) / 2
}
