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

// Package imagex provides image helpers.
package imagex

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"math"
	"net/http"
	"os"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// DecodeDataURI extracts and decodes the base64 image data from a Data URI.
//
// If dataURI doesn't have the "data:image/...;base64," prefix,
// it decodes the whole string as raw base64.
func DecodeDataURI(dataURI string) ([]byte, error) {
	if strings.HasPrefix(dataURI, "data:image/") {
		if index := strings.Index(dataURI, ";base64,"); index > 0 {
			dataURI = dataURI[index+len(";base64,"):]
		}
	}
	return base64.StdEncoding.DecodeString(dataURI)
}

// Load decodes an image from r and optionally applies opacity to it.
func Load(r io.Reader, opacity float64) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("fail to decode the image: %w", err)
	}

	if opacity > 0 && opacity < 1 {
		img = ApplyOpacity(img, opacity)
	}

	return img, nil
}

// LoadFile loads an image from a local path or an HTTP(S) URL.
func LoadFile(path string, opacity float64) (image.Image, error) {
	var f io.ReadCloser
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		resp, err := http.Get(path)
		switch {
		case err != nil:
			return nil, fmt.Errorf("fail to open the image from url: %w", err)

		case resp.StatusCode != 200:
			return nil, fmt.Errorf("fail to open the image from url: statuscode=%d", resp.StatusCode)

		default:
			f = resp.Body
		}
	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("fail to open the image from file: %w", err)
		}
		f = file
	}

	defer f.Close()
	return Load(f, opacity)
}

// ToNRGBA returns a cloned NRGBA image for img.
func ToNRGBA(img image.Image) *image.NRGBA {
	if nrgba, ok := img.(*image.NRGBA); ok {
		return CloneNRGBA(nrgba)
	}

	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), img, b.Min, draw.Src)
	return out
}

// CloneNRGBA returns a deep copy of src.
func CloneNRGBA(src *image.NRGBA) *image.NRGBA {
	if src == nil {
		return nil
	}

	out := image.NewNRGBA(image.Rect(0, 0, src.Bounds().Dx(), src.Bounds().Dy()))
	copy(out.Pix, src.Pix)
	return out
}

// Resize rescales src by scale while keeping at least one pixel per side.
func Resize(src image.Image, scale float64) image.Image {
	newW := max(1, int(float64(src.Bounds().Dx())*scale))
	newH := max(1, int(float64(src.Bounds().Dy())*scale))
	if newW == src.Bounds().Dx() && newH == src.Bounds().Dy() {
		return src
	}

	dst := image.NewNRGBA(image.Rect(0, 0, newW, newH))
	xdraw.NearestNeighbor.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// ApplyOpacity multiplies the alpha channel of src by opacity.
func ApplyOpacity(src image.Image, opacity float64) image.Image {
	if opacity >= 1 {
		return src
	}

	if opacity <= 0 {
		return image.NewNRGBA(src.Bounds())
	}

	out := ToNRGBA(src)
	for i := 3; i < len(out.Pix); i += 4 {
		out.Pix[i] = uint8(math.Round(float64(out.Pix[i]) * opacity))
	}

	return out
}

// DropAlpha flattens images with alpha onto a white background.
func DropAlpha(img image.Image) image.Image {
	switch img.(type) {
	case *image.RGBA, *image.NRGBA, *image.RGBA64, *image.NRGBA64, *image.Alpha, *image.Alpha16:
		return flattenToRGB(img)
	default:
		return img
	}
}

func flattenToRGB(img image.Image) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(out, b, img, b.Min, draw.Over)
	return out
}

// Alpha255 converts an opacity ratio in [0, 1] into an 8-bit alpha value.
func Alpha255(v float64) uint8 {
	if v <= 0 {
		return 0
	}

	if v >= 1 {
		return 255
	}

	return uint8(math.Round(v * 255))
}
