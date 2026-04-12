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

package font

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

type Entry struct {
	Path  string
	Index int

	fullName string
	stemName string
	psName   string
	family   string

	normFullName string
	normStemName string
	normPsName   string
	normFamily   string

	font *opentype.Font
}

func NewEntry(path string, index int, font *opentype.Font) Entry {
	fileName := filepath.Base(path)

	entry := Entry{Path: path, Index: index}
	entry.stemName = strings.TrimSuffix(fileName, filepath.Ext(fileName))

	if font != nil {
		var buf sfnt.Buffer
		entry.family = preferName(font, &buf, sfnt.NameIDTypographicFamily, sfnt.NameIDWWSFamily, sfnt.NameIDFamily)
		entry.fullName = preferName(font, &buf, sfnt.NameIDFull, sfnt.NameIDCompatibleFull)
		entry.psName = preferName(font, &buf, sfnt.NameIDPostScript)
	}

	entry.normFullName = normalizeLookupName(entry.fullName)
	entry.normStemName = normalizeLookupName(entry.stemName)
	entry.normPsName = normalizeLookupName(entry.psName)
	entry.normFamily = normalizeLookupName(entry.family)

	return entry
}

func (e *Entry) Load() (*opentype.Font, error) {
	data, err := os.ReadFile(e.Path)
	if err != nil {
		return nil, fmt.Errorf("fail to read the font file: %w", err)
	}

	coll, err := opentype.ParseCollection(data)
	if err != nil {
		return nil, fmt.Errorf("fail to parse font collection: %w", err)
	}

	if e.Index < 0 || e.Index >= coll.NumFonts() {
		return nil, errors.New("invalid font entry: invalid font index")
	}

	font, err := coll.Font(e.Index)
	if err != nil {
		return nil, fmt.Errorf("invalid font: %w", err)
	}

	return font, nil
}

func (e *Entry) Match(name string) bool {
	name = normalizeLookupName(name)
	switch name {
	case "":
		return false

	case e.normFamily, e.normPsName, e.normStemName, e.normFullName:
		return true

	default:
		return false
	}
}

func preferName(f *opentype.Font, buf *sfnt.Buffer, ids ...sfnt.NameID) string {
	for _, id := range ids {
		v, err := f.Name(buf, id)
		if err == nil && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func normalizeLookupName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	s = filepath.Base(s)
	if IsSupportedFontFile(s) {
		s = strings.TrimSuffix(s, filepath.Ext(s))
	}

	return squashName(s)
}

func squashName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}

	return b.String()
}
