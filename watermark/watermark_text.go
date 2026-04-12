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
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"

	"github.com/xgfone/go-imagex"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type TextWatermark struct {
	Position Position
	Opacity  float64
	Scale    float64
	DPI      float64
}

func (wm *TextWatermark) setDefault() {
	if wm.Opacity <= 0 {
		wm.Opacity = 0.7
	}

	if wm.Scale <= 0 {
		wm.Scale = 0.05
	}

	if wm.DPI <= 0 {
		wm.DPI = 72
	}
}

// Font is required to create the font face based on the font size.
func (wm TextWatermark) Draw(src image.Image, otfont *opentype.Font, text string) (dst image.Image, err error) {
	wm.setDefault()

	isLandscape := src.Bounds().Dx() > src.Bounds().Dy()
	baseSize := src.Bounds().Dx()
	if !isLandscape {
		baseSize = src.Bounds().Dy()
	}

	fontSize := math.Max(16, float64(int(float64(baseSize)*wm.Scale*1.2)))
	faceOptions := opentype.FaceOptions{Size: fontSize, DPI: wm.DPI, Hinting: font.HintingFull}
	face, err := opentype.NewFace(otfont, &faceOptions)
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}
	defer closeFontFace(face)

	if isLandscape {
		return wm.drawHorizontalText(src, face, text), nil
	}
	return wm.drawVerticalText(src, face, text), nil
}

func (wm TextWatermark) DrawHorizontalText(src image.Image, face font.Face, text string) image.Image {
	return wm.drawHorizontalText(src, face, text)
}

func (wm TextWatermark) DrawVerticalText(src image.Image, face font.Face, text string) image.Image {
	return wm.drawVerticalText(src, face, text)
}

func (wm *TextWatermark) drawHorizontalText(src image.Image, face font.Face, text string) image.Image {
	wm.setDefault()

	w, hgt := measureText(face, text)
	pos := wm.Position.calculatePosition(src.Bounds().Size(), image.Pt(w, hgt))
	patch := renderTextPatch(face, text, textFillColor(wm.Opacity), textShadowColor(wm.Opacity), image.Pt(1, 1))

	out := imagex.ToNRGBA(src)
	rect := image.Rectangle{Min: pos, Max: pos.Add(patch.Bounds().Size())}
	draw.Draw(out, rect, patch, image.Point{}, draw.Over)
	return out
}

func (wm *TextWatermark) drawVerticalText(src image.Image, face font.Face, text string) image.Image {
	wm.setDefault()

	units := parseTextUnits(text)
	if len(units) == 0 {
		return src
	}

	type unitMetric struct {
		rawWidth   int
		effectiveH int
		patch      *image.NRGBA
	}

	metrics := make([]unitMetric, 0, len(units))
	maxUnitWidth := 0
	heightSum := 0
	for _, unit := range units {
		rawW, rawH := measureText(face, unit)
		effectiveH := rawH
		if isHalfHeightUnit(unit) {
			effectiveH = int(math.Round(float64(effectiveH) / 2.0))
		}

		patch := renderTextPatch(face, unit, textFillColor(wm.Opacity), textShadowColor(wm.Opacity), image.Pt(1, 1))
		maxUnitWidth = max(maxUnitWidth, rawW)
		heightSum += rawH
		metrics = append(metrics, unitMetric{rawWidth: rawW, effectiveH: effectiveH, patch: patch})
	}

	avgUnitHeight := 0
	if len(metrics) > 0 {
		avgUnitHeight = heightSum / len(metrics)
	}

	unitSpacing := int(float64(avgUnitHeight) * 0.2)
	totalHeight := 0
	for _, m := range metrics {
		totalHeight += m.effectiveH
	}

	if len(metrics) > 1 {
		totalHeight += unitSpacing * (len(metrics) - 1)
	}

	pos := wm.Position.calculatePosition(src.Bounds().Size(), image.Pt(maxUnitWidth, totalHeight))
	centerX := pos.X + maxUnitWidth/2
	y := pos.Y

	out := imagex.ToNRGBA(src)
	for _, m := range metrics {
		x := centerX - m.rawWidth/2
		rect := image.Rectangle{Min: image.Pt(x, y), Max: image.Pt(x+m.patch.Bounds().Dx(), y+m.patch.Bounds().Dy())}
		draw.Draw(out, rect, m.patch, image.Point{}, draw.Over)
		y += m.effectiveH + unitSpacing
	}

	return out
}

func clampInt(v, minV, maxV int) int {
	if v < minV {
		return minV
	}

	if v > maxV {
		return maxV
	}

	return v
}

func textFillColor(opacity float64) color.NRGBA {
	return color.NRGBA{255, 255, 255, imagex.Alpha255(min(1.0, opacity*1.2))}
}

func textShadowColor(opacity float64) color.NRGBA {
	return color.NRGBA{0, 0, 0, imagex.Alpha255(opacity * 0.8)}
}

func renderTextPatch(face font.Face, text string, fill, shadow color.NRGBA, shadowOffset image.Point) *image.NRGBA {
	bounds, w, h := textBounds(face, text)
	padLeft := max(0, -shadowOffset.X)
	padTop := max(0, -shadowOffset.Y)
	padRight := max(0, shadowOffset.X)
	padBottom := max(0, shadowOffset.Y)
	patch := image.NewNRGBA(image.Rect(0, 0, w+padLeft+padRight, h+padTop+padBottom))

	baseDot := fixed.Point26_6{
		X: -bounds.Min.X + fixed.I(padLeft),
		Y: -bounds.Min.Y + fixed.I(padTop),
	}

	d := &font.Drawer{Dst: patch, Face: face}
	if shadow.A > 0 {
		d.Src = image.NewUniform(shadow)
		d.Dot = fixed.Point26_6{X: baseDot.X + fixed.I(shadowOffset.X), Y: baseDot.Y + fixed.I(shadowOffset.Y)}
		d.DrawString(text)
	}

	if fill.A > 0 {
		d.Src = image.NewUniform(fill)
		d.Dot = baseDot
		d.DrawString(text)
	}

	return patch
}

func textBounds(face font.Face, s string) (fixed.Rectangle26_6, int, int) {
	b, _ := font.BoundString(face, s)
	w := (b.Max.X - b.Min.X).Ceil()
	h := (b.Max.Y - b.Min.Y).Ceil()
	return b, w, h
}

func measureText(face font.Face, s string) (int, int) {
	_, w, h := textBounds(face, s)
	return w, h
}

func parseTextUnits(text string) []string {
	if text == "" {
		return nil
	}

	units := make([]string, 0, len([]rune(text)))
	var current []rune
	inEnglish := false
	flushChinese := func(rs []rune) {
		for _, r := range rs {
			units = append(units, string(r))
		}
	}

	for _, r := range text {
		isEnglish := r <= 127 && ((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
		if isEnglish {
			if !inEnglish && len(current) > 0 {
				flushChinese(current)
				current = current[:0]
			}

			current = append(current, r)
			inEnglish = true
			continue
		}

		if inEnglish && len(current) > 0 {
			units = append(units, string(current))
			current = current[:0]
		}

		current = append(current, r)
		inEnglish = false
	}

	if len(current) > 0 {
		if inEnglish {
			units = append(units, string(current))
		} else {
			flushChinese(current)
		}
	}

	return units
}

func isHalfHeightUnit(s string) bool {
	runes := []rune(s)
	return len(runes) == 1 && strings.ContainsRune("*@#$%^&*", runes[0])
}

func closeFontFace(face font.Face) {
	if c, ok := face.(interface{ Close() error }); ok {
		_ = c.Close()
	}
}
