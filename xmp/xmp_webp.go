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

package xmp

import (
	"encoding/binary"
	"errors"
	"image"
	"math"

	"github.com/eringen/gowebper"
)

// EncodeWebPWithXMP encodes img as WebP and injects xmp into the RIFF payload.
func EncodeWebPWithXMP(img image.Image, quality float32, xmp []byte) ([]byte, error) {
	opts := &gowebper.Options{Level: gowebper.LevelDefault}
	q := int(math.Round(float64(quality)))
	if q > 0 {
		if q > 100 {
			q = 100
		}
		opts.Quality = q
	}

	payload, err := gowebper.EncodeToBytes(img, opts)
	if err != nil {
		return nil, err
	}

	if len(xmp) == 0 {
		return payload, nil
	}

	return injectWebPXMP(payload, xmp)
}

func injectWebPXMP(data, xmp []byte) ([]byte, error) {
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return nil, errors.New("invalid webp data")
	}

	chunks := data[12:]
	hasVP8X := false
	vp8xIndex := -1
	for off := 0; off+8 <= len(chunks); {
		fourcc := string(chunks[off : off+4])
		sz := int(binary.LittleEndian.Uint32(chunks[off+4 : off+8]))
		chunkTotal := 8 + sz + (sz % 2)
		if off+chunkTotal > len(chunks) {
			return nil, errors.New("corrupt webp chunks")
		}

		if fourcc == "VP8X" {
			hasVP8X = true
			vp8xIndex = off
			break
		}

		off += chunkTotal
	}

	newChunk := riffChunk("XMP ", xmp)
	out := make([]byte, 0, len(data)+len(newChunk)+18)
	out = append(out, data[:12]...)

	if hasVP8X {
		// Reuse the existing VP8X chunk and only enable the XMP feature bit.
		updated := append([]byte(nil), chunks...)
		updated[vp8xIndex+8] |= 0x04 // XMP flag
		out = append(out, updated...)
		out = append(out, newChunk...)
		binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
		return out, nil
	}

	// Older/simple files may not have VP8X, so synthesize one from the canvas info.
	dims, hasAlpha, err := webpCanvasInfo(chunks)
	if err != nil {
		return nil, err
	}

	vp8x := make([]byte, 18)
	copy(vp8x[:4], []byte("VP8X"))
	binary.LittleEndian.PutUint32(vp8x[4:8], 10)

	flags := byte(0x04)
	if hasAlpha {
		flags |= 0x10
	}

	vp8x[8] = flags
	write24LE(vp8x[12:15], uint32(dims.X-1))
	write24LE(vp8x[15:18], uint32(dims.Y-1))

	out = append(out, vp8x...)
	out = append(out, chunks...)
	out = append(out, newChunk...)
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
	return out, nil
}

func riffChunk(fourcc string, payload []byte) []byte {
	out := make([]byte, 0, 8+len(payload)+1)
	out = append(out, []byte(fourcc)...)
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(payload)))
	out = append(out, lenBuf...)
	out = append(out, payload...)
	if len(payload)%2 == 1 {
		out = append(out, 0)
	}
	return out
}

func webpCanvasInfo(chunks []byte) (image.Point, bool, error) {
	hasAlpha := false
	for off := 0; off+8 <= len(chunks); {
		fourcc := string(chunks[off : off+4])
		sz := int(binary.LittleEndian.Uint32(chunks[off+4 : off+8]))
		chunkTotal := 8 + sz + (sz % 2)
		if off+chunkTotal > len(chunks) {
			return image.Point{}, false, errors.New("corrupt webp data")
		}

		data := chunks[off+8 : off+8+sz]
		switch fourcc {
		case "VP8X":
			if len(data) < 10 {
				return image.Point{}, false, errors.New("invalid VP8X chunk")
			}

			hasAlpha = data[0]&0x10 != 0
			return image.Pt(int(read24LE(data[4:7]))+1, int(read24LE(data[7:10]))+1), hasAlpha, nil

		case "VP8 ":
			if len(data) < 10 {
				return image.Point{}, false, errors.New("invalid VP8 chunk")
			}

			for i := 0; i+9 < len(data); i++ {
				if data[i] == 0x9d && data[i+1] == 0x01 && data[i+2] == 0x2a {
					w := int(binary.LittleEndian.Uint16(data[i+3:i+5]) & 0x3FFF)
					h := int(binary.LittleEndian.Uint16(data[i+5:i+7]) & 0x3FFF)
					return image.Pt(w, h), hasAlpha, nil
				}
			}

		case "VP8L":
			if len(data) < 5 || data[0] != 0x2f {
				return image.Point{}, false, errors.New("invalid VP8L chunk")
			}

			bits := uint32(data[1]) | uint32(data[2])<<8 | uint32(data[3])<<16 | uint32(data[4])<<24
			w := int(bits&0x3FFF) + 1
			h := int((bits>>14)&0x3FFF) + 1
			hasAlpha = (bits>>28)&1 == 1
			return image.Pt(w, h), hasAlpha, nil

		case "ALPH":
			hasAlpha = true
		}
		off += chunkTotal
	}
	return image.Point{}, false, errors.New("unable to determine webp canvas size")
}

func write24LE(dst []byte, v uint32) {
	dst[0] = byte(v)
	dst[1] = byte(v >> 8)
	dst[2] = byte(v >> 16)
}

func read24LE(src []byte) uint32 {
	return uint32(src[0]) | uint32(src[1])<<8 | uint32(src[2])<<16
}
