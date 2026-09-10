package main

import (
	"bytes"
	"image"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func fixture() ([]fertility, []population) {
	f := make([]fertility, 0, 19)
	p := make([]population, 0, 19*19*2)
	for y := startYear; y <= endYear; y++ {
		value := 2.0 + float64(y-startYear)*.02
		if y > 2014 {
			value = 2.14 - float64(y-2014)*.04
		}
		f = append(f, fertility{Year: y, Value: value, Raw: "1.5", Status: "A"})
		for _, a := range ages {
			for _, s := range []struct{ code, label string }{{"1", "Erkek"}, {"2", "Kadın"}} {
				p = append(p, population{Year: y, AgeCode: a.Code, AgeLabel: a.Label, SexCode: s.code, SexLabel: s.label, Value: 1_000_000, Raw: "1000000"})
			}
		}
	}
	return f, p
}

func TestValidateRejectsDuplicate(t *testing.T) {
	f, p := fixture()
	if err := validate(f, p); err != nil {
		t.Fatal(err)
	}
	p[1] = p[0]
	if err := validate(f, p); err == nil {
		t.Fatal("expected duplicate-cell error")
	}
}

func TestValidateProtects2014Annotation(t *testing.T) {
	f, p := fixture()
	f[2015-startYear].Value = f[2014-startYear].Value
	if err := validate(f, p); err == nil {
		t.Fatal("expected invalid sustained-decline annotation error")
	}
}

func TestYearlyPNGsAreDeterministic(t *testing.T) {
	f, p := fixture()
	d := t.TempDir()
	a, b := filepath.Join(d, "a"), filepath.Join(d, "b")
	if err := writeFrames(a, f, p); err != nil {
		t.Fatal(err)
	}
	if err := writeFrames(b, f, p); err != nil {
		t.Fatal(err)
	}
	for year := startYear; year <= endYear; year++ {
		name := strconv.Itoa(year) + ".png"
		aa, err := os.ReadFile(filepath.Join(a, name))
		if err != nil {
			t.Fatal(err)
		}
		bb, err := os.ReadFile(filepath.Join(b, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(aa, bb) {
			t.Fatalf("identical inputs produced different bytes for %s", name)
		}
	}
	first, err := os.Open(filepath.Join(a, "2007.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	decoded, err := png.Decode(first)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds() != image.Rect(0, 0, 1200, 675) {
		t.Fatalf("unexpected PNG geometry: %v", decoded.Bounds())
	}
}

func TestAnimatedGIFIsDeterministic(t *testing.T) {
	f, p := fixture()
	d := t.TempDir()
	frames := filepath.Join(d, "frames")
	if err := writeFrames(frames, f, p); err != nil {
		t.Fatal(err)
	}
	a, b := filepath.Join(d, "a.gif"), filepath.Join(d, "b.gif")
	if err := writeGIF(a, frames, f); err != nil {
		t.Fatal(err)
	}
	if err := writeGIF(b, frames, f); err != nil {
		t.Fatal(err)
	}
	aa, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	bb, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(aa, bb) {
		t.Fatal("identical inputs produced different GIF bytes")
	}
	decoded, err := gif.DecodeAll(bytes.NewReader(aa))
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Image) != len(f) || len(decoded.Delay) != len(f) {
		t.Fatalf("unexpected GIF frame counts: images=%d delays=%d", len(decoded.Image), len(decoded.Delay))
	}
	if decoded.Config.Width != 1200 || decoded.Config.Height != 675 || decoded.LoopCount != 0 {
		t.Fatalf("unexpected GIF configuration: %+v loop=%d", decoded.Config, decoded.LoopCount)
	}
}

func TestRobotoSupportsAndRendersTurkishGlyphs(t *testing.T) {
	r, err := newTextRenderer(20)
	if err != nil {
		t.Fatal(err)
	}
	defer r.close()
	for _, ch := range "İıĞğŞşÇçÖöÜü" {
		if _, ok := r.faces[20].GlyphAdvance(ch); !ok {
			t.Fatalf("Roboto lacks required Turkish glyph %q", ch)
		}
	}
	im := image.NewPaletted(image.Rect(0, 0, 300, 40), pal)
	r.text(im, 2, 2, 20, "TÜİK: doğurganlık, kişi", 1)
	ink := 0
	for _, px := range im.Pix {
		if px != 0 {
			ink++
		}
	}
	if ink < 100 {
		t.Fatalf("Turkish text did not render as expected: %d ink pixels", ink)
	}
}
