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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var xmlPrefixPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9._-]*$`)

type AIGC struct {
	Label string `json:",omitempty"`

	ContentProducer string `json:",omitempty"`
	ProduceID       string `json:",omitempty"`
	ReserveCode1    string `json:",omitempty"`

	Propagator   string `json:",omitempty"`
	PropagatorID string `json:",omitempty"`
	ReserveCode2 string `json:",omitempty"`
}

func (aigc AIGC) BuildXMPPacketData(nsPrefix, nsURI string) (data []byte, err error) {
	var buf bytes.Buffer
	buf.Grow(512)

	err = aigc.BuildXMPPacket(&buf, nsPrefix, nsURI)
	if err != nil {
		return
	}

	data = buf.Bytes()
	return
}

func (aigc AIGC) BuildXMPPacket(w io.Writer, nsPrefix, nsURI string) (err error) {
	if nsPrefix == "" {
		nsPrefix = "aigc"
	} else if nsPrefix, err = validateXMLPrefix(nsPrefix); err != nil {
		return fmt.Errorf("invalid namespace prefix: %w", err)
	}

	if nsURI == "" {
		nsURI = "https://www.aigc.com/ns/xmp/1.0/"
	} else if nsURI = strings.TrimSpace(nsURI); nsURI == "" {
		return errors.New("namespace URI is required")
	} else {
		nsURI = escapeXMLAttr(nsURI)
	}

	aigcJSON, err := marshalCompactJSON(aigc)
	if err != nil {
		return fmt.Errorf("marshal AIGC metadata: %w", err)
	}
	aigcJSON = escapeXMLAttr(aigcJSON)

	xmpdatas := []string{
		`<x:xmpmeta xmlns:x="adobe:ns:meta/">` + "\n",
		` <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` + "\n",
		`  <rdf:Description rdf:about="" xmlns:`, nsPrefix, `="`, nsURI, `">` + "\n",
		`   <`, nsPrefix, `:AIGC>`, aigcJSON, `</`, nsPrefix, `:AIGC>` + "\n",
		`  </rdf:Description>` + "\n",
		` </rdf:RDF>` + "\n",
		`</x:xmpmeta>` + "\n",
	}

	for _, data := range xmpdatas {
		_, err = io.WriteString(w, data)
		if err != nil {
			return err
		}
	}

	return
}

func validateXMLPrefix(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("prefix is empty")
	}

	if strings.EqualFold(s, "xml") || strings.EqualFold(s, "xmlns") {
		return "", fmt.Errorf("reserved prefix %q is not allowed", s)
	}

	if !xmlPrefixPattern.MatchString(s) {
		return "", fmt.Errorf("prefix %q does not match XML namespace prefix rules", s)
	}

	return s, nil
}

func marshalCompactJSON(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

var _xmpescaper = strings.NewReplacer(
	`&`, `&amp;`,
	`"`, `&quot;`,
	`<`, `&lt;`,
	`>`, `&gt;`,
	`'`, `&apos;`,
)

func escapeXMLAttr(s string) string {
	return _xmpescaper.Replace(s)
}
