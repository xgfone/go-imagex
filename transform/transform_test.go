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

package transform

import (
	"context"
	"errors"
	"image"
	"image/color"
	"testing"
)

func testImage() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{10, 20, 30, 255})
	img.SetNRGBA(1, 0, color.NRGBA{40, 50, 60, 255})
	return img
}

func TestTransformFunc(t *testing.T) {
	src := testImage()
	tf := TransformFunc(func(ctx context.Context, img image.Image) (image.Image, error) {
		if ctx.Value("k") != "v" {
			t.Fatal("context value not passed through")
		}
		out := image.NewNRGBA(img.Bounds())
		out.Set(0, 0, img.At(1, 0))
		out.Set(1, 0, img.At(0, 0))
		return out, nil
	})

	got, err := tf.Transform(context.WithValue(context.Background(), "k", "v"), src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c := color.NRGBAModel.Convert(got.At(0, 0)).(color.NRGBA); c.R != 40 {
		t.Fatalf("unexpected transformed pixel: %#v", c)
	}
}

func TestTransformersTransform(t *testing.T) {
	src := testImage()
	called := 0

	first := TransformFunc(func(ctx context.Context, img image.Image) (image.Image, error) {
		called++
		out := image.NewNRGBA(img.Bounds())
		out.Set(0, 0, color.NRGBA{1, 2, 3, 255})
		out.Set(1, 0, img.At(1, 0))
		return out, nil
	})
	second := TransformFunc(func(ctx context.Context, img image.Image) (image.Image, error) {
		called++
		out := image.NewNRGBA(img.Bounds())
		out.Set(0, 0, img.At(0, 0))
		out.Set(1, 0, color.NRGBA{9, 8, 7, 255})
		return out, nil
	})

	got, err := Transformers{first, second}.Transform(context.Background(), src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 2 {
		t.Fatalf("unexpected call count: %d", called)
	}
	if c := color.NRGBAModel.Convert(got.At(0, 0)).(color.NRGBA); c.R != 1 {
		t.Fatalf("unexpected first pixel: %#v", c)
	}
	if c := color.NRGBAModel.Convert(got.At(1, 0)).(color.NRGBA); c.R != 9 {
		t.Fatalf("unexpected second pixel: %#v", c)
	}
}

func TestTransformersTransformStopsOnError(t *testing.T) {
	src := testImage()
	wantErr := errors.New("stop")
	called := 0

	first := TransformFunc(func(ctx context.Context, img image.Image) (image.Image, error) {
		called++
		out := image.NewNRGBA(img.Bounds())
		out.Set(0, 0, color.NRGBA{7, 7, 7, 255})
		out.Set(1, 0, img.At(1, 0))
		return out, nil
	})
	second := TransformFunc(func(ctx context.Context, img image.Image) (image.Image, error) {
		called++
		return nil, wantErr
	})
	third := TransformFunc(func(ctx context.Context, img image.Image) (image.Image, error) {
		called++
		return img, nil
	})

	got, err := Transformers{first, second, third}.Transform(context.Background(), src)
	if !errors.Is(err, wantErr) {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 2 {
		t.Fatalf("expected pipeline to stop after error, got %d calls", called)
	}
	if c := color.NRGBAModel.Convert(got.At(0, 0)).(color.NRGBA); c.R != 7 {
		t.Fatalf("expected last successful image, got %#v", c)
	}
}

func TestTransformersEmpty(t *testing.T) {
	src := testImage()
	got, err := Transformers(nil).Transform(context.Background(), src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != src {
		t.Fatal("empty pipeline should return original image")
	}
}
