package review

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

const maxSnippetEdge = 320

// SaveBandSnippet crops the transaction band from the source image and writes a PNG.
// RelPath is relative to dateDir (e.g. review/门店/001_xxx.png).
func SaveBandSnippet(srcImagePath, dateDir, store string, serial int, sourceHint string, bandBox [4][2]float64) (relPath string, err error) {
	f, err := os.Open(srcImagePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}
	bounds := img.Bounds()
	left := int(bandBox[0][0])
	top := int(bandBox[0][1])
	right := int(bandBox[1][0])
	bottom := int(bandBox[2][1])
	if right <= left {
		right = bounds.Max.X
		left = bounds.Min.X
	}
	if left < bounds.Min.X {
		left = bounds.Min.X
	}
	if top < bounds.Min.Y {
		top = bounds.Min.Y
	}
	if right > bounds.Max.X {
		right = bounds.Max.X
	}
	if bottom > bounds.Max.Y {
		bottom = bounds.Max.Y
	}
	if bottom <= top || right <= left {
		// BandBox 无效时退化为整图底部一条带，保证待核对仍有片段可看
		h := bounds.Dy()
		bandH := h / 12
		if bandH < 40 {
			bandH = 40
		}
		if bandH > 120 {
			bandH = 120
		}
		bottom = bounds.Max.Y
		top = bottom - bandH
		if top < bounds.Min.Y {
			top = bounds.Min.Y
		}
		left = bounds.Min.X
		right = bounds.Max.X
	}
	if bottom <= top || right <= left {
		return "", fmt.Errorf("invalid band box")
	}

	rect := image.Rect(left, top, right, bottom)
	sub, ok := img.(interface {
		SubImage(r image.Rectangle) image.Image
	})
	if !ok {
		return "", fmt.Errorf("image type does not support SubImage")
	}
	cropped := sub.SubImage(rect)
	thumb := resizeMaxEdge(cropped, maxSnippetEdge)

	dir := filepath.Join(dateDir, "review", store)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%03d_%s.png", serial, sanitizeName(sourceHint))
	abs := filepath.Join(dir, name)
	out, err := os.Create(abs)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if err := png.Encode(out, thumb); err != nil {
		return "", err
	}
	relPath = filepath.ToSlash(filepath.Join("review", store, name))
	return relPath, nil
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "item"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 0x4e00 && r <= 0x9fff:
			b.WriteRune(r)
		case r == '*' || r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
		if b.Len() >= 24 {
			break
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "item"
	}
	return out
}

func resizeMaxEdge(src image.Image, maxEdge int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return src
	}
	if w <= maxEdge && h <= maxEdge {
		return src
	}
	scale := float64(maxEdge) / float64(w)
	if h > w {
		scale = float64(maxEdge) / float64(h)
	}
	nw := int(float64(w) * scale)
	nh := int(float64(h) * scale)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}
