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
	"image"
	"image/draw"

	"github.com/xgfone/go-imagex"
)

type ImageWatermark struct {
	Position Position
}

func (wm ImageWatermark) Draw(src, mark image.Image) image.Image {
	out := imagex.ToNRGBA(src)
	pos := wm.Position.calculatePosition(src.Bounds().Size(), mark.Bounds().Size())
	rect := image.Rectangle{Min: pos, Max: pos.Add(mark.Bounds().Size())}
	draw.Draw(out, rect, mark, image.Point{}, draw.Over)
	return out
}
