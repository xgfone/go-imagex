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
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/jpeg"
)

const jpegXMPHeader = "http://ns.adobe.com/xap/1.0/\x00"

// EncodeJPEGWithXMP encodes img as JPEG and injects xmp into an APP1 segment.
func EncodeJPEGWithXMP(img image.Image, quality int, xmp []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	if len(xmp) == 0 {
		return buf.Bytes(), nil
	}
	return injectJPEGXMP(buf.Bytes(), xmp)
}

func injectJPEGXMP(data, xmp []byte) ([]byte, error) {
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil, errors.New("invalid jpeg data")
	}
	payload := append([]byte(jpegXMPHeader), xmp...)
	if len(payload)+2 > 0xFFFF {
		return nil, errors.New("jpeg APP1 XMP payload too large")
	}

	seg := make([]byte, 0, len(payload)+4)
	seg = append(seg, 0xFF, 0xE1)
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(payload)+2))
	seg = append(seg, lenBuf...)
	seg = append(seg, payload...)

	insertAt := 2
	if len(data) >= 6 && data[2] == 0xFF && data[3] == 0xE0 {
		app0Len := int(binary.BigEndian.Uint16(data[4:6]))
		if len(data) >= 4+app0Len {
			insertAt = 4 + app0Len
		}
	}
	out := make([]byte, 0, len(data)+len(seg))
	out = append(out, data[:insertAt]...)
	out = append(out, seg...)
	out = append(out, data[insertAt:]...)
	return out, nil
}
