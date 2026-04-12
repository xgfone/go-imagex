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
)

func ExtractOrientationFromJPEG(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}

	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			break
		}

		marker := data[i+1]
		if marker == 0xDA || marker == 0xD9 {
			break
		}

		if i+4 > len(data) {
			break
		}

		segLen := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if segLen < 2 || i+2+segLen > len(data) {
			break
		}

		if marker == 0xE1 {
			payload := data[i+4 : i+2+segLen]
			if len(payload) >= 6 && bytes.Equal(payload[:6], []byte{'E', 'x', 'i', 'f', 0, 0}) {
				if o, ok := parseExifOrientation(payload[6:]); ok {
					return o
				}
			}
		}

		i += 2 + segLen
	}

	return 1
}

func parseExifOrientation(tiff []byte) (int, bool) {
	if len(tiff) < 8 {
		return 0, false
	}

	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian

	case "MM":
		order = binary.BigEndian

	default:
		return 0, false
	}

	if order.Uint16(tiff[2:4]) != 42 {
		return 0, false
	}

	ifd0 := int(order.Uint32(tiff[4:8]))
	if ifd0 < 0 || ifd0+2 > len(tiff) {
		return 0, false
	}

	count := int(order.Uint16(tiff[ifd0 : ifd0+2]))
	entryBase := ifd0 + 2
	for i := range count {
		off := entryBase + i*12
		if off+12 > len(tiff) {
			return 0, false
		}

		tag := order.Uint16(tiff[off : off+2])
		if tag != 0x0112 {
			continue
		}

		typ := order.Uint16(tiff[off+2 : off+4])
		cnt := order.Uint32(tiff[off+4 : off+8])
		if typ != 3 || cnt < 1 {
			return 0, false
		}

		val := order.Uint16(tiff[off+8 : off+10])
		return int(val), true
	}

	return 0, false
}
