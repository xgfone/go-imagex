// Copyright 2026 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package watermark

import (
	"context"
	"image"
	"image/color"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

func wmBaseImage(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, color.NRGBA{20, 30, 40, 255})
		}
	}
	return img
}

func coloredMark(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func TestPositionHelpers(t *testing.T) {
	pos := Position{Position: PositionBottomRight, LongEdgePaddingRatio: 0.1, ShortEdgePaddingRatio: 0.05}
	if got := pos.calculatePosition(image.Pt(100, 60), image.Pt(20, 10)); got != (image.Pt(70, 47)) {
		t.Fatalf("unexpected position: %v", got)
	}

	if got := pos.calculatePosition(image.Pt(10, 10), image.Pt(30, 30)); got != (image.Pt(0, 0)) {
		t.Fatalf("unexpected clamped position: %v", got)
	}

	left, right, top, bottom := pos.resolveEdgeOffsets(image.Pt(60, 100))
	if left != 3 || right != 3 || top != 10 || bottom != 10 {
		t.Fatalf("unexpected edge offsets: %d %d %d %d", left, right, top, bottom)
	}

	if got := clampInt(5, 0, 3); got != 3 {
		t.Fatalf("unexpected clamp: %d", got)
	}
}

func TestImageWatermarkDraw(t *testing.T) {
	src := wmBaseImage(6, 4)
	mark := coloredMark(2, 1, color.NRGBA{200, 10, 10, 255})
	out, _ := ImageWatermark{Position: Position{Position: PositionTopRight}, MarkImage: mark}.Transform(context.Background(), src)
	got := color.NRGBAModel.Convert(out.At(4, 0)).(color.NRGBA)
	if got.R == 20 {
		t.Fatalf("expected watermark at top-right, got %v", got)
	}
}

func TestImageWatermarkDrawUsesMarkBounds(t *testing.T) {
	src := wmBaseImage(6, 4)
	want := color.NRGBA{R: 200, G: 10, B: 10, A: 255}
	mark := coloredMark(4, 3, want).SubImage(image.Rect(2, 1, 4, 3))

	position := Position{Position: PositionBottomRight}
	out := position.ImageWatermark().Draw(src, mark)
	if got := color.NRGBAModel.Convert(out.At(4, 2)).(color.NRGBA); got != want {
		t.Fatalf("watermark pixel at (4,2) = %v, want %v", got, want)
	}
}

func TestTextWatermarkDraw(t *testing.T) {
	otf, err := opentype.Parse(goregular.TTF)
	if err != nil {
		t.Fatalf("parse font: %v", err)
	}

	face, err := opentype.NewFace(otf, &opentype.FaceOptions{Size: 18, DPI: 72})
	if err != nil {
		t.Fatalf("new face: %v", err)
	}
	defer closeFontFace(face)

	wm := TextWatermark{Position: Position{Position: PositionCenter}}
	landscape := wmBaseImage(120, 60)
	portrait := wmBaseImage(60, 120)

	gotLandscape := wm.DrawHorizontalText(landscape, face, "Hello")
	gotPortrait := wm.DrawVerticalText(portrait, face, "中A*")

	if sameImage(landscape, gotLandscape) {
		t.Fatal("expected horizontal text to modify image")
	}
	if sameImage(portrait, gotPortrait) {
		t.Fatal("expected vertical text to modify image")
	}

	drawn, err := wm.Draw(landscape, otf, "Hello")
	if err != nil {
		t.Fatalf("draw text: %v", err)
	}
	if sameImage(landscape, drawn) {
		t.Fatal("expected Draw to modify image")
	}

	if got := wm.DrawVerticalText(portrait, face, ""); got != portrait {
		t.Fatal("empty text should return original image")
	}
}

func TestTextWatermarkKeepsShadowAtBottomRight(t *testing.T) {
	font, err := opentype.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	face, err := opentype.NewFace(font, &opentype.FaceOptions{Size: 24, DPI: 72})
	if err != nil {
		t.Fatal(err)
	}
	defer closeFontFace(face)

	countVisible := func(img image.Image) (count int) {
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
					count++
				}
			}
		}
		return count
	}

	src := image.NewNRGBA(image.Rect(0, 0, 100, 80))
	center := TextWatermark{Position: Position{Position: PositionCenter}, Opacity: 1}
	corner := center.WithPosition(Position{Position: PositionBottomRight})
	if got, want := countVisible(corner.DrawHorizontalText(src, face, "Ag")), countVisible(center.DrawHorizontalText(src, face, "Ag")); got != want || want == 0 {
		t.Errorf("horizontal bottom-right coverage = %d, centered coverage = %d", got, want)
	}
	if got, want := countVisible(corner.DrawVerticalText(src, face, "Ag*")), countVisible(center.DrawVerticalText(src, face, "Ag*")); got != want || want == 0 {
		t.Errorf("vertical bottom-right coverage = %d, centered coverage = %d", got, want)
	}
}

func sameImage(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}

	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			if color.NRGBAModel.Convert(a.At(x, y)) != color.NRGBAModel.Convert(b.At(x, y)) {
				return false
			}
		}
	}

	return true
}

func TestTextHelpers(t *testing.T) {
	if got := parseTextUnits("中A12#文"); len(got) != 4 {
		t.Fatalf("unexpected units: %#v", got)
	}
	if parseTextUnits("") != nil {
		t.Fatal("empty text should return nil units")
	}
	if !isHalfHeightUnit("*") || isHalfHeightUnit("AB") {
		t.Fatal("unexpected half-height detection")
	}
	if got := textFillColor(0.5); got.A == 0 {
		t.Fatalf("unexpected fill color: %#v", got)
	}
	if got := textShadowColor(0.5); got.A == 0 {
		t.Fatalf("unexpected shadow color: %#v", got)
	}

	wm := &TextWatermark{}
	wm.setDefault()
	if wm.Opacity <= 0 || wm.Scale <= 0 || wm.DPI <= 0 {
		t.Fatalf("defaults not applied: %#v", wm)
	}
}
