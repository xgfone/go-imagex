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

package orientation

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

func testPatternImage() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	colors := []color.NRGBA{
		{255, 0, 0, 255},
		{0, 255, 0, 255},
		{0, 0, 255, 255},
		{255, 255, 0, 255},
		{255, 0, 255, 255},
		{0, 255, 255, 255},
	}
	i := 0
	for y := range 3 {
		for x := range 2 {
			img.SetNRGBA(x, y, colors[i])
			i++
		}
	}
	return img
}

func imgSignature(img image.Image) []color.NRGBA {
	b := img.Bounds()
	out := make([]color.NRGBA, 0, b.Dx()*b.Dy())
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out = append(out, color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA))
		}
	}
	return out
}

func TestApplyOrientation(t *testing.T) {
	src := testPatternImage()
	tests := []struct {
		orientation int
		size        image.Point
		want        []color.NRGBA
	}{
		{1, image.Pt(2, 3), imgSignature(src)},
		{2, image.Pt(2, 3), []color.NRGBA{
			{0, 255, 0, 255},
			{255, 0, 0, 255},
			{255, 255, 0, 255},
			{0, 0, 255, 255},
			{0, 255, 255, 255},
			{255, 0, 255, 255}},
		},
		{3, image.Pt(2, 3), []color.NRGBA{
			{0, 255, 255, 255},
			{255, 0, 255, 255},
			{255, 255, 0, 255},
			{0, 0, 255, 255},
			{0, 255, 0, 255},
			{255, 0, 0, 255},
		}},
		{4, image.Pt(2, 3), []color.NRGBA{
			{255, 0, 255, 255},
			{0, 255, 255, 255},
			{0, 0, 255, 255},
			{255, 255, 0, 255},
			{255, 0, 0, 255},
			{0, 255, 0, 255},
		}},
		{5, image.Pt(3, 2), []color.NRGBA{
			{255, 0, 0, 255},
			{0, 0, 255, 255},
			{255, 0, 255, 255},
			{0, 255, 0, 255},
			{255, 255, 0, 255},
			{0, 255, 255, 255},
		}},
		{6, image.Pt(3, 2), []color.NRGBA{
			{255, 0, 255, 255},
			{0, 0, 255, 255},
			{255, 0, 0, 255},
			{0, 255, 255, 255},
			{255, 255, 0, 255},
			{0, 255, 0, 255},
		}},
		{7, image.Pt(3, 2), []color.NRGBA{
			{0, 255, 255, 255},
			{255, 255, 0, 255},
			{0, 255, 0, 255},
			{255, 0, 255, 255},
			{0, 0, 255, 255},
			{255, 0, 0, 255},
		}},
		{8, image.Pt(3, 2), []color.NRGBA{
			{0, 255, 0, 255},
			{255, 255, 0, 255},
			{0, 255, 255, 255},
			{255, 0, 0, 255},
			{0, 0, 255, 255},
			{255, 0, 255, 255},
		}},
	}

	for _, tt := range tests {
		got := ApplyOrientation(src, tt.orientation)
		if got.Bounds().Size() != tt.size {
			t.Fatalf("orientation %d size=%v want %v", tt.orientation, got.Bounds().Size(), tt.size)
		}
		if sig := imgSignature(got); !equalColors(sig, tt.want) {
			t.Fatalf("orientation %d signature mismatch: %#v", tt.orientation, sig)
		}
	}
}

func equalColors(a, b []color.NRGBA) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func buildEXIFOrientationJPEG(orientation uint16, order binary.ByteOrder) []byte {
	var tiff bytes.Buffer
	if order == binary.LittleEndian {
		tiff.WriteString("II")
	} else {
		tiff.WriteString("MM")
	}

	_ = binary.Write(&tiff, order, uint16(42))
	_ = binary.Write(&tiff, order, uint32(8))
	_ = binary.Write(&tiff, order, uint16(1))
	_ = binary.Write(&tiff, order, uint16(0x0112))
	_ = binary.Write(&tiff, order, uint16(3))
	_ = binary.Write(&tiff, order, uint32(1))
	_ = binary.Write(&tiff, order, orientation)
	_ = binary.Write(&tiff, order, uint16(0))
	_ = binary.Write(&tiff, order, uint32(0))

	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	seg := make([]byte, 4)
	seg[0], seg[1] = 0xFF, 0xE1
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))

	out := []byte{0xFF, 0xD8}
	out = append(out, seg...)
	out = append(out, payload...)
	out = append(out, 0xFF, 0xD9)
	return out
}

func TestExtractOrientationFromJPEG(t *testing.T) {
	if got := ExtractOrientationFromJPEG(buildEXIFOrientationJPEG(6, binary.BigEndian)); got != 6 {
		t.Fatalf("unexpected big-endian orientation: %d", got)
	}
	if got := ExtractOrientationFromJPEG(buildEXIFOrientationJPEG(8, binary.LittleEndian)); got != 8 {
		t.Fatalf("unexpected little-endian orientation: %d", got)
	}
	if got := ExtractOrientationFromJPEG([]byte("bad")); got != 1 {
		t.Fatalf("unexpected fallback orientation: %d", got)
	}
}

func TestParseExifOrientationFailures(t *testing.T) {
	cases := [][]byte{
		nil,
		[]byte("II"),
		append([]byte("ZZ"), make([]byte, 10)...),
		func() []byte {
			buf := append([]byte("II"), 0, 0)
			buf = append(buf, 0, 0, 0, 0)
			return buf
		}(),
	}
	for _, data := range cases {
		if _, ok := parseExifOrientation(data); ok {
			t.Fatalf("expected parse failure for %v", data)
		}
	}

	var buf bytes.Buffer
	buf.WriteString("MM")
	_ = binary.Write(&buf, binary.BigEndian, uint16(42))
	_ = binary.Write(&buf, binary.BigEndian, uint32(8))
	_ = binary.Write(&buf, binary.BigEndian, uint16(1))
	_ = binary.Write(&buf, binary.BigEndian, uint16(0x0112))
	_ = binary.Write(&buf, binary.BigEndian, uint16(1))
	_ = binary.Write(&buf, binary.BigEndian, uint32(1))
	_ = binary.Write(&buf, binary.BigEndian, uint32(0))
	if _, ok := parseExifOrientation(buf.Bytes()); ok {
		t.Fatal("expected invalid type failure")
	}
}
