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

// Package encoder provides the image encoder.
package encoder

import (
	"context"
	"image"
	"image/jpeg"
	"image/png"
	"io"
)

// Encoder encodes the image to the writer.
type Encoder interface {
	Encode(context.Context, io.Writer, image.Image) error
}

// EncodeFunc is the function type of the encoder.
type EncodeFunc func(context.Context, io.Writer, image.Image) error

// Encode implements the interface Encoder.
func (f EncodeFunc) Encode(ctx context.Context, w io.Writer, img image.Image) error {
	return f(ctx, w, img)
}

// NewJPEGEncoder returns a new JPEG encoder.
func NewJPEGEncoder(quality int) Encoder {
	return EncodeFunc(func(_ context.Context, w io.Writer, img image.Image) error {
		return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
	})
}

// NewPNGEncoder returns a new PNG encoder.
func NewPNGEncoder() Encoder {
	return EncodeFunc(func(_ context.Context, w io.Writer, img image.Image) error {
		return png.Encode(w, img)
	})
}
