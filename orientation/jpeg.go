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

// ExtractOrientationFromJPEG returns the EXIF orientation value from JPEG data.
// It falls back to 1 when no valid orientation can be found.
func ExtractOrientationFromJPEG(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}

	// Walk the JPEG segments until Start Of Scan or End Of Image.
	for i := 2; i < len(data) && data[i] == 0xFF; {
		// A marker may be preceded by any number of 0xFF fill bytes.
		for i < len(data) && data[i] == 0xFF {
			i++
		}
		if i == len(data) {
			break
		}

		marker := data[i]
		i++
		switch {
		case marker == 0xDA || marker == 0xD9:
			return 1
		case marker == 0x01 || (0xD0 <= marker && marker <= 0xD7):
			continue // Standalone markers have no length field.
		}
		if len(data)-i < 2 {
			break
		}

		segLen := int(binary.BigEndian.Uint16(data[i : i+2]))
		if segLen < 2 || segLen > len(data)-i {
			break
		}

		if marker == 0xE1 {
			payload := data[i+2 : i+segLen]
			// Parse the EXIF payload when the APP1 segment starts with "Exif\0\0".
			if len(payload) >= 6 && bytes.Equal(payload[:6], []byte{'E', 'x', 'i', 'f', 0, 0}) {
				if o, ok := parseExifOrientation(payload[6:]); ok {
					return o
				}
			}
		}

		i += segLen
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

	offset := order.Uint32(tiff[4:8])
	if uint64(offset) > uint64(len(tiff)-2) {
		return 0, false
	}
	ifd0 := int(offset)

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
		val := order.Uint16(tiff[off+8 : off+10])
		if typ != 3 || cnt != 1 || val < 1 || val > 8 {
			return 0, false
		}

		return int(val), true
	}

	return 0, false
}
