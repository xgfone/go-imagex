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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/image/font/opentype"
)

func ScanDirs(dirs []string) ([]Entry, error) {
	fontfiles := make([]Entry, 0, 128)

	for _, root := range dirs {
		if root == "" {
			continue
		}

		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fontfiles, err
		}

		if !info.IsDir() {
			continue
		}

		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !IsSupportedFontFile(path) {
				return err
			}

			fontfiles, err = ScanFile(path, fontfiles)
			return err
		})

		if err != nil {
			return fontfiles, err
		}
	}

	return fontfiles, nil
}

func ScanFile(path string, fontEntries []Entry) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fontEntries, err
	}

	// 用 ParseCollection 统一兼容 TTF/OTF/TTC/OTC。
	coll, err := opentype.ParseCollection(data)
	if err != nil {
		fmt.Println("+++111", path)
		return fontEntries, err
	}

	num := coll.NumFonts()
	if num <= 0 {
		fontEntries = append(fontEntries, NewEntry(path, 0, nil))
		return fontEntries, nil
	}

	for i := range num {
		if font, err := coll.Font(i); err == nil {
			fontEntries = append(fontEntries, NewEntry(path, i, font))
		}
	}

	return fontEntries, nil
}
