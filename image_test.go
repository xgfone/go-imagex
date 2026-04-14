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

package imagex

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func solidNRGBA(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func TestLoadAndLoadFile(t *testing.T) {
	src := solidNRGBA(2, 1, color.NRGBA{10, 20, 30, 200})

	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	img, err := Load(bytes.NewReader(buf.Bytes()), 0.5)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA).A; got != 100 {
		t.Fatalf("unexpected alpha after Load: %d", got)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "img.png")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write image: %v", err)
	}
	if _, err := LoadFile(path, 1); err != nil {
		t.Fatalf("load file: %v", err)
	}
}

func TestLoadErrors(t *testing.T) {
	if _, err := Load(bytes.NewReader([]byte("bad")), 1); err == nil {
		t.Fatal("expected decode error")
	}

	if _, err := LoadFile(filepath.Join(t.TempDir(), "missing.png"), 1); err == nil {
		t.Fatal("expected missing file error")
	}

	if _, err := LoadFile("http://127.0.0.1:0/unreachable.png", 1); err == nil {
		t.Fatal("expected url open error")
	}
}

func TestImageHelpers(t *testing.T) {
	src := solidNRGBA(2, 2, color.NRGBA{100, 110, 120, 200})

	cloned := ToNRGBA(src)
	if cloned == src || cloned.Pix[3] != 200 {
		t.Fatal("ToNRGBA should clone source")
	}

	if CloneNRGBA(nil) != nil {
		t.Fatal("CloneNRGBA(nil) should return nil")
	}

	copied := CloneNRGBA(src)
	copied.Pix[0] = 1
	if src.Pix[0] == 1 {
		t.Fatal("CloneNRGBA should deep copy")
	}

	resizedSame := Resize(src, 1)
	if resizedSame != src {
		t.Fatal("Resize(1) should return original")
	}

	resized := Resize(src, 0.2)
	if got := resized.Bounds().Size(); got != (image.Pt(1, 1)) {
		t.Fatalf("unexpected resized size: %v", got)
	}

	if got := ApplyOpacity(src, 2); got != src {
		t.Fatal("opacity >= 1 should return original")
	}

	transparent := ApplyOpacity(src, 0)
	if got := color.NRGBAModel.Convert(transparent.At(0, 0)).(color.NRGBA).A; got != 0 {
		t.Fatalf("unexpected transparent alpha: %d", got)
	}

	flattened := DropAlpha(src)
	if _, ok := flattened.(*image.RGBA); !ok {
		t.Fatal("DropAlpha should flatten alpha images")
	}

	jpgImg := image.NewGray(image.Rect(0, 0, 1, 1))
	if got := DropAlpha(jpgImg); got != jpgImg {
		t.Fatal("DropAlpha should keep opaque-only types")
	}
}

func TestAlpha255(t *testing.T) {
	tests := []struct {
		in   float64
		want uint8
	}{
		{-1, 0},
		{0, 0},
		{0.5, 128},
		{1, 255},
		{2, 255},
	}
	for _, tt := range tests {
		if got := Alpha255(tt.in); got != tt.want {
			t.Fatalf("Alpha255(%v)=%d want %d", tt.in, got, tt.want)
		}
	}
}

func TestLoadSupportsJPEG(t *testing.T) {
	src := solidNRGBA(1, 1, color.NRGBA{200, 100, 50, 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	if _, err := Load(bytes.NewReader(buf.Bytes()), 1); err != nil {
		t.Fatalf("load jpeg: %v", err)
	}
}

func TestDecodeDataURI(t *testing.T) {
	tests := []struct {
		name    string
		dataURI string
		want    string // expected decoded string for easy comparison
		wantErr bool
	}{
		{
			name:    "valid DataURI with jpeg",
			dataURI: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte("hello")),
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "valid DataURI with png",
			dataURI: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("world")),
			want:    "world",
			wantErr: false,
		},
		{
			name:    "raw base64 without prefix",
			dataURI: base64.StdEncoding.EncodeToString([]byte("raw data")),
			want:    "raw data",
			wantErr: false,
		},
		{
			name:    "invalid base64 string",
			dataURI: "not base64!!!",
			want:    "",
			wantErr: true,
		},
		{
			name:    "DataURI with extra params after image type (ignored)",
			dataURI: "data:image/webp;base64," + base64.StdEncoding.EncodeToString([]byte("webp")),
			want:    "webp",
			wantErr: false,
		},
		{
			name:    "DataURI with missing base64 marker",
			dataURI: "data:image/jpeg;charset=utf8,abc",
			want:    "abc", // falls back to decoding the whole string as raw base64 -> will error because "abc" is not valid base64
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeDataURI(tt.dataURI)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeDataURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if string(got) != tt.want {
					t.Errorf("DecodeDataURI() got = %q, want %q", string(got), tt.want)
				}
			}
		})
	}
}
