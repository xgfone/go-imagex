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
	"hash/crc32"
)

const pngXMPKeyword = "XML:com.adobe.xmp"

func init() {
	RegisterInjectFunc("png", InjectPNG)
}

// InjectPNG injects the XMP metadata into the iTXt chunk of the PNG data.
func InjectPNG(pngData, xmpData []byte) ([]byte, error) {
	if len(xmpData) == 0 {
		return pngData, nil
	}
	return injectPNGiTXtChunk(pngData, pngXMPKeyword, xmpData)
}

func injectPNGiTXtChunk(data []byte, keyword string, text []byte) ([]byte, error) {
	const sigLen = 8
	if len(data) < sigLen+12 || !bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return nil, errors.New("invalid png data")
	}
	ihdrLen := int(binary.BigEndian.Uint32(data[8:12]))
	if len(data) < 8+12+ihdrLen {
		return nil, errors.New("corrupt png data")
	}
	insertAt := 8 + 4 + 4 + ihdrLen + 4

	chunkData := make([]byte, 0, len(keyword)+len(text)+5)
	chunkData = append(chunkData, []byte(keyword)...)
	chunkData = append(chunkData, 0) // keyword terminator
	chunkData = append(chunkData, 0) // compression flag: uncompressed
	chunkData = append(chunkData, 0) // compression method
	chunkData = append(chunkData, 0) // language tag terminator
	chunkData = append(chunkData, 0) // translated keyword terminator
	chunkData = append(chunkData, text...)

	chunk := pngChunk("iTXt", chunkData)
	out := make([]byte, 0, len(data)+len(chunk))
	out = append(out, data[:insertAt]...)
	out = append(out, chunk...)
	out = append(out, data[insertAt:]...)
	return out, nil
}

func pngChunk(kind string, payload []byte) []byte {
	out := make([]byte, 0, len(payload)+12)
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(payload)))
	out = append(out, lenBuf...)
	out = append(out, []byte(kind)...)
	out = append(out, payload...)
	crc := crc32.ChecksumIEEE(append([]byte(kind), payload...))
	crcBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBuf, crc)
	out = append(out, crcBuf...)
	return out
}
