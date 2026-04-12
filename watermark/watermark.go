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

import "image"

const (
	// PositionCenter centers the watermark on the base image.
	PositionCenter = "center"
	// PositionTopCenter places the watermark at the top center.
	PositionTopCenter = "top-center"
	// PositionBottomCenter places the watermark at the bottom center.
	PositionBottomCenter = "bottom-center"

	// PositionTopLeft places the watermark at the top-left corner.
	PositionTopLeft = "top-left"
	// PositionTopRight places the watermark at the top-right corner.
	PositionTopRight = "top-right"
	// PositionBottomLeft places the watermark at the bottom-left corner.
	PositionBottomLeft = "bottom-left"
	// PositionBottomRight places the watermark at the bottom-right corner.
	PositionBottomRight = "bottom-right"
)

// Position describes where a watermark should be placed.
type Position struct {
	Position string

	LongEdgePaddingRatio  float64 // Such as 0.05 for 5% padding on the long edge
	ShortEdgePaddingRatio float64 // Such as 0.02 for 2% padding on the short edge
}

func (p *Position) calculatePosition(baseSize, elementSize image.Point) image.Point {
	baseWidth := baseSize.X
	baseHeight := baseSize.Y
	elemWidth := max(elementSize.X, 0)
	elemHeight := max(elementSize.Y, 0)
	maxX := max(0, baseWidth-elemWidth)
	maxY := max(0, baseHeight-elemHeight)

	left, right, top, bottom := p.resolveEdgeOffsets(baseSize)
	centerX := (baseWidth - elemWidth) / 2
	centerY := (baseHeight - elemHeight) / 2

	switch p.Position {
	case PositionCenter:
		return image.Pt(clampInt(centerX, 0, maxX), clampInt(centerY, 0, maxY))

	case PositionTopCenter:
		return image.Pt(clampInt(centerX, 0, maxX), clampInt(top, 0, maxY))

	case PositionBottomCenter:
		return image.Pt(clampInt(centerX, 0, maxX), clampInt(baseHeight-elemHeight-bottom, 0, maxY))

	case PositionTopLeft:
		return image.Pt(clampInt(left, 0, maxX), clampInt(top, 0, maxY))

	case PositionTopRight:
		return image.Pt(clampInt(baseWidth-elemWidth-right, 0, maxX), clampInt(top, 0, maxY))

	case PositionBottomLeft:
		return image.Pt(clampInt(left, 0, maxX), clampInt(baseHeight-elemHeight-bottom, 0, maxY))

	case PositionBottomRight:
		return image.Pt(clampInt(baseWidth-elemWidth-right, 0, maxX), clampInt(baseHeight-elemHeight-bottom, 0, maxY))

	default:
		return image.Pt(clampInt(left, 0, maxX), clampInt(top, 0, maxY))
	}
}

func (p *Position) resolveEdgeOffsets(baseSize image.Point) (left, right, top, bottom int) {
	baseWidth := baseSize.X
	baseHeight := baseSize.Y

	var defaultHorizontal, defaultVertical int
	if baseWidth < baseHeight { // Vertical
		defaultHorizontal = int(float64(baseWidth) * p.ShortEdgePaddingRatio)
		defaultVertical = int(float64(baseHeight) * p.LongEdgePaddingRatio)
	} else { // Horizontal
		defaultHorizontal = int(float64(baseWidth) * p.LongEdgePaddingRatio)
		defaultVertical = int(float64(baseHeight) * p.ShortEdgePaddingRatio)
	}

	left = defaultHorizontal
	right = defaultHorizontal
	top = defaultVertical
	bottom = defaultVertical
	return
}
