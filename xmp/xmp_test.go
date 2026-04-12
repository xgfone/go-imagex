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
	"image"
	"strings"
	"testing"
)

func xmpTestImage() *image.NRGBA {
	return image.NewNRGBA(image.Rect(0, 0, 2, 2))
}

func TestAIGCBuildXMPPacket(t *testing.T) {
	aigc := AIGC{Label: `A&B`, ContentProducer: "bot"}
	data, err := aigc.BuildXMPPacketData("", "")
	if err != nil {
		t.Fatalf("build packet data: %v", err)
	}

	s := string(data)
	if !strings.Contains(s, "<aigc:AIGC>") || !strings.Contains(s, "A&amp;B") {
		t.Fatalf("unexpected packet: %s", s)
	}

	var buf bytes.Buffer
	if err := aigc.BuildXMPPacket(&buf, "ns1", " https://example.com/ns "); err != nil {
		t.Fatalf("build packet: %v", err)
	}

	if !strings.Contains(buf.String(), `xmlns:ns1="https://example.com/ns"`) {
		t.Fatalf("unexpected namespace: %s", buf.String())
	}
}

func TestAIGCBuildXMPPacketErrors(t *testing.T) {
	aigc := AIGC{}
	if err := aigc.BuildXMPPacket(&bytes.Buffer{}, "xml", "x"); err == nil {
		t.Fatal("expected reserved prefix error")
	}
	if err := aigc.BuildXMPPacket(&bytes.Buffer{}, "bad prefix", "x"); err == nil {
		t.Fatal("expected invalid prefix error")
	}
	if err := aigc.BuildXMPPacket(&bytes.Buffer{}, "ok", " "); err == nil {
		t.Fatal("expected empty namespace error")
	}
	if got := escapeXMLAttr(`'"<>&`); got != "&apos;&quot;&lt;&gt;&amp;" {
		t.Fatalf("unexpected xml escape: %q", got)
	}
	if _, err := marshalCompactJSON(map[string]string{"x": "<>"}); err != nil {
		t.Fatalf("marshal compact json: %v", err)
	}
}

func TestEncodeJPEGWithXMP(t *testing.T) {
	xmpData := []byte("<xmp/>")
	data, err := EncodeJPEGWithXMP(xmpTestImage(), 80, xmpData)
	if err != nil {
		t.Fatalf("encode jpeg with xmp: %v", err)
	}
	if !bytes.Contains(data, append([]byte(jpegXMPHeader), xmpData...)) {
		t.Fatal("expected jpeg xmp payload")
	}

	plain, err := EncodeJPEGWithXMP(xmpTestImage(), 80, nil)
	if err != nil {
		t.Fatalf("encode plain jpeg: %v", err)
	}
	if bytes.Contains(plain, []byte(jpegXMPHeader)) {
		t.Fatal("unexpected xmp header in plain jpeg")
	}

	if _, err := injectJPEGXMP([]byte("bad"), xmpData); err == nil {
		t.Fatal("expected invalid jpeg error")
	}
	tooLarge := bytes.Repeat([]byte("x"), 0xFFFF)
	if _, err := injectJPEGXMP([]byte{0xFF, 0xD8, 0xFF, 0xD9}, tooLarge); err == nil {
		t.Fatal("expected oversized xmp error")
	}
}

func TestEncodePNGWithXMP(t *testing.T) {
	xmpData := []byte("<xmp/>")
	data, err := EncodePNGWithXMP(xmpTestImage(), xmpData)
	if err != nil {
		t.Fatalf("encode png with xmp: %v", err)
	}
	if !bytes.Contains(data, []byte("iTXt")) || !bytes.Contains(data, xmpData) {
		t.Fatal("expected png xmp chunk")
	}

	plain, err := EncodePNGWithXMP(xmpTestImage(), nil)
	if err != nil {
		t.Fatalf("encode plain png: %v", err)
	}
	if bytes.Contains(plain, []byte("iTXt")) {
		t.Fatal("unexpected iTXt chunk in plain png")
	}

	if _, err := injectPNGiTXtChunk([]byte("bad"), pngXMPKeyword, xmpData); err == nil {
		t.Fatal("expected invalid png error")
	}
	if _, err := injectPNGiTXtChunk(append([]byte{137, 80, 78, 71, 13, 10, 26, 10}, make([]byte, 4)...), pngXMPKeyword, xmpData); err == nil {
		t.Fatal("expected corrupt png error")
	}
	if chunk := pngChunk("tEXt", []byte("abc")); len(chunk) != 15 {
		t.Fatalf("unexpected png chunk size: %d", len(chunk))
	}
}

func TestInjectWebPXMP(t *testing.T) {
	vp8 := append([]byte{
		'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'E', 'B', 'P',
	}, riffChunk("VP8 ", []byte{0x9d, 0x01, 0x2a, 0x02, 0x00, 0x03, 0x00, 0, 0, 0})...)
	binary.LittleEndian.PutUint32(vp8[4:8], uint32(len(vp8)-8))

	out, err := injectWebPXMP(vp8, []byte("meta"))
	if err != nil {
		t.Fatalf("inject webp xmp: %v", err)
	}
	if !bytes.Contains(out, []byte("VP8X")) || !bytes.Contains(out, []byte("XMP ")) {
		t.Fatal("expected synthesized VP8X and XMP chunks")
	}

	withVP8X := append([]byte{
		'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'E', 'B', 'P',
	}, riffChunk("VP8X", []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 2})...)
	withVP8X = append(withVP8X, riffChunk("VP8 ", []byte{0x9d, 0x01, 0x2a, 0x02, 0x00, 0x03, 0x00, 0, 0, 0})...)
	binary.LittleEndian.PutUint32(withVP8X[4:8], uint32(len(withVP8X)-8))
	out, err = injectWebPXMP(withVP8X, []byte("meta"))
	if err != nil {
		t.Fatalf("inject webp xmp with VP8X: %v", err)
	}
	if out[20]&0x04 == 0 {
		t.Fatal("expected XMP flag to be enabled")
	}

	if _, err := injectWebPXMP([]byte("bad"), []byte("x")); err == nil {
		t.Fatal("expected invalid webp error")
	}
}

func TestWebPHelpers(t *testing.T) {
	if got := riffChunk("TEST", []byte("abc")); len(got) != 12 {
		t.Fatalf("unexpected riff chunk len: %d", len(got))
	}

	dst := make([]byte, 3)
	write24LE(dst, 0x030201)
	if got := read24LE(dst); got != 0x030201 {
		t.Fatalf("unexpected 24-bit roundtrip: %#x", got)
	}

	dims, hasAlpha, err := webpCanvasInfo(riffChunk("VP8L", []byte{0x2f, 1, 0, 0, 0}))
	if err != nil || dims != (image.Pt(2, 1)) || hasAlpha {
		t.Fatalf("unexpected VP8L info: %v %v %v", dims, hasAlpha, err)
	}

	dims, hasAlpha, err = webpCanvasInfo(append(riffChunk("ALPH", []byte{1}), riffChunk("VP8X", []byte{0x10, 0, 0, 0, 1, 0, 0, 2, 0, 0})...))
	if err != nil || dims != (image.Pt(2, 3)) || !hasAlpha {
		t.Fatalf("unexpected VP8X info: %v %v %v", dims, hasAlpha, err)
	}

	if _, _, err := webpCanvasInfo(riffChunk("VP8X", []byte{1})); err == nil {
		t.Fatal("expected invalid VP8X error")
	}
	if _, _, err := webpCanvasInfo(riffChunk("VP8L", []byte{0})); err == nil {
		t.Fatal("expected invalid VP8L error")
	}
	if _, _, err := webpCanvasInfo([]byte("short")); err == nil {
		t.Fatal("expected corrupt chunk error")
	}
	if _, _, err := webpCanvasInfo(riffChunk("TEST", []byte{1, 2})); err == nil {
		t.Fatal("expected canvas size error")
	}
}
