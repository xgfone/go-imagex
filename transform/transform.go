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

// Package transform provides some composable image transformation helpers.
package transform

import (
	"context"
	"image"
)

// Transformer transforms an image and returns the transformed result.
type Transformer interface {
	Transform(context.Context, image.Image) (image.Image, error)
}

// TransformFunc adapts a function to the Transformer interface.
type TransformFunc func(context.Context, image.Image) (image.Image, error)

// Transform implements the interface Transformer.
func (f TransformFunc) Transform(ctx context.Context, img image.Image) (image.Image, error) {
	return f(ctx, img)
}

// Transformers applies multiple transformers in order.
type Transformers []Transformer

// Transform runs each transformer in sequence and stops at the first error.
func (ts Transformers) Transform(ctx context.Context, img image.Image) (image.Image, error) {
	_img := img

	var err error
	for i := range ts {
		if _img, err = ts[i].Transform(ctx, _img); err != nil {
			break
		} else {
			img = _img
		}
	}

	return img, err
}
