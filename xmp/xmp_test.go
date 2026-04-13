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
	"image/jpeg"
	"image/png"
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

func TestAIGCHelpers(t *testing.T) {
	if !(AIGC{}).IsZero() {
		t.Fatal("zero AIGC should be empty")
	}

	aigc := (AIGC{}).WithProduceID("p1").WithPropagatorID("p2")
	if aigc.IsZero() || aigc.ProduceID != "p1" || aigc.PropagatorID != "p2" {
		t.Fatalf("unexpected AIGC helper result: %#v", aigc)
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

func TestRegistryInject(t *testing.T) {
	orig, existed := _registries["test"]
	if existed {
		defer func() { _registries["test"] = orig }()
	} else {
		defer delete(_registries, "test")
	}

	called := false
	RegisterInjectFunc("test", func(imageData, xmpData []byte) ([]byte, error) {
		called = true
		return append(append([]byte(nil), imageData...), xmpData...), nil
	})

	got, err := Inject("test", []byte("img"), []byte("xmp"))
	if err != nil {
		t.Fatalf("inject dispatch: %v", err)
	}
	if !called || string(got) != "imgxmp" {
		t.Fatalf("unexpected dispatch result: %q", got)
	}

	if _, err := Inject("missing", []byte("img"), []byte("xmp")); err == nil {
		t.Fatal("expected missing injector error")
	}
}

func TestRegisterInjectFuncPanics(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "empty type",
			fn: func() {
				RegisterInjectFunc("", func(imageData, xmpData []byte) ([]byte, error) { return nil, nil })
			},
		},
		{
			name: "nil func",
			fn: func() {
				RegisterInjectFunc("test", nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			tt.fn()
		})
	}
}

func TestInjectJPEG(t *testing.T) {
	img := xmpTestImage()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}

	got, err := InjectJPEG(buf.Bytes(), []byte("<xmp/>"))
	if err != nil {
		t.Fatalf("inject jpeg: %v", err)
	}
	if !bytes.Contains(got, []byte(jpegXMPHeader)) {
		t.Fatal("expected jpeg xmp header")
	}

	got, err = InjectJPEG(buf.Bytes(), nil)
	if err != nil {
		t.Fatalf("inject jpeg without xmp: %v", err)
	}
	if !bytes.Equal(got, buf.Bytes()) {
		t.Fatal("empty xmp should return original jpeg data")
	}

	if _, err := InjectJPEG([]byte("bad"), []byte("x")); err == nil {
		t.Fatal("expected invalid jpeg error")
	}
}

func TestInjectJPEGWithAPP0(t *testing.T) {
	jpegData := []byte{
		0xFF, 0xD8,
		0xFF, 0xE0, 0x00, 0x04, 0x00, 0x00,
		0xFF, 0xD9,
	}
	got, err := injectJPEGXMP(jpegData, []byte("x"))
	if err != nil {
		t.Fatalf("inject jpeg with app0: %v", err)
	}
	if !bytes.Equal(got[2:8], jpegData[2:8]) {
		t.Fatal("expected xmp segment to be inserted after APP0")
	}
}

func TestInjectPNG(t *testing.T) {
	img := xmpTestImage()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	got, err := InjectPNG(buf.Bytes(), []byte("<xmp/>"))
	if err != nil {
		t.Fatalf("inject png: %v", err)
	}
	if !bytes.Contains(got, []byte("iTXt")) || !bytes.Contains(got, []byte(pngXMPKeyword)) {
		t.Fatal("expected png iTXt chunk")
	}

	got, err = InjectPNG(buf.Bytes(), nil)
	if err != nil {
		t.Fatalf("inject png without xmp: %v", err)
	}
	if !bytes.Equal(got, buf.Bytes()) {
		t.Fatal("empty xmp should return original png data")
	}

	if _, err := InjectPNG([]byte("bad"), []byte("x")); err == nil {
		t.Fatal("expected invalid png error")
	}
}

func TestInjectPNGiTXtChunk(t *testing.T) {
	img := xmpTestImage()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	got, err := injectPNGiTXtChunk(buf.Bytes(), "keyword", []byte("payload"))
	if err != nil {
		t.Fatalf("inject itxt: %v", err)
	}
	if !bytes.Contains(got, []byte("keyword")) || !bytes.Contains(got, []byte("payload")) {
		t.Fatal("expected keyword and payload in png data")
	}

	if _, err := injectPNGiTXtChunk([]byte("bad"), pngXMPKeyword, []byte("x")); err == nil {
		t.Fatal("expected invalid png error")
	}
	if _, err := injectPNGiTXtChunk(append([]byte{137, 80, 78, 71, 13, 10, 26, 10}, make([]byte, 4)...), pngXMPKeyword, []byte("x")); err == nil {
		t.Fatal("expected corrupt png error")
	}
	if chunk := pngChunk("tEXt", []byte("abc")); len(chunk) != 15 {
		t.Fatalf("unexpected png chunk size: %d", len(chunk))
	}
}

func TestInjectWEBP(t *testing.T) {
	webp := append([]byte{
		'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'E', 'B', 'P',
	}, riffChunk("VP8 ", []byte{0x9d, 0x01, 0x2a, 0x02, 0x00, 0x03, 0x00, 0, 0, 0})...)
	binary.LittleEndian.PutUint32(webp[4:8], uint32(len(webp)-8))

	got, err := InjectWEBP(webp, []byte("meta"))
	if err != nil {
		t.Fatalf("inject webp: %v", err)
	}
	if !bytes.Contains(got, []byte("XMP ")) {
		t.Fatal("expected webp xmp chunk")
	}

	got, err = InjectWEBP(webp, nil)
	if err != nil {
		t.Fatalf("inject webp without xmp: %v", err)
	}
	if !bytes.Equal(got, webp) {
		t.Fatal("empty xmp should return original webp data")
	}

	if _, err := InjectWEBP([]byte("bad"), []byte("x")); err == nil {
		t.Fatal("expected invalid webp error")
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

	vp8Chunk := riffChunk("VP8 ", []byte{0x9d, 0x01, 0x2a, 0x02, 0x00, 0x03, 0x00, 0, 0, 0})
	dims, hasAlpha, err = webpCanvasInfo(vp8Chunk)
	if err != nil || dims != (image.Pt(2, 3)) || hasAlpha {
		t.Fatalf("unexpected VP8 info: %v %v %v", dims, hasAlpha, err)
	}

	if _, _, err := webpCanvasInfo(riffChunk("VP8X", []byte{1})); err == nil {
		t.Fatal("expected invalid VP8X error")
	}
	if _, _, err := webpCanvasInfo(riffChunk("VP8L", []byte{0})); err == nil {
		t.Fatal("expected invalid VP8L error")
	}
	if _, _, err := webpCanvasInfo(riffChunk("VP8 ", []byte{1, 2, 3})); err == nil {
		t.Fatal("expected invalid VP8 error")
	}
	if _, _, err := webpCanvasInfo([]byte("short")); err == nil {
		t.Fatal("expected corrupt chunk error")
	}
	if _, _, err := webpCanvasInfo(riffChunk("TEST", []byte{1, 2})); err == nil {
		t.Fatal("expected canvas size error")
	}
}
