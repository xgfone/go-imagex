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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	systemDirsValue    = sync.OnceValue(getSystemDirs)
	systemEntriesValue = sync.OnceValues(getSystemEntries)
)

// GetSystemDirs returns the common system font directories for the OS.
func GetSystemDirs() []string {
	return systemDirsValue()
}

// GetSystemEntries returns the font entries for the system font directories.
func GetSystemEntries() ([]Entry, error) {
	return systemEntriesValue()
}

func getSystemEntries() ([]Entry, error) {
	return ScanDirs(GetSystemDirs())
}

func getSystemDirs() []string {
	dirs := make([]string, 0, 6)
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "windows":
		winDir := os.Getenv("WINDIR")
		if winDir == "" {
			winDir = `C:\Windows`
		}

		if home != "" {
			dir := filepath.Join(home, "AppData", "Local", "Microsoft", "Windows", "Fonts")
			dirs = append(dirs, dir)
		}

		dirs = append(dirs, filepath.Join(winDir, "Fonts"))
		return cleanDirs(dirs)

	case "darwin":
		if home != "" {
			dirs = append(dirs, filepath.Join(home, "Library", "Fonts"))
		}

		dirs = append(dirs, "/Library/Fonts", "/System/Library/Fonts")
		return cleanDirs(dirs)

	default: // linux / unix-like
		if home != "" {
			dirs = append(dirs,
				filepath.Join(home, ".local", "share", "fonts"),
				filepath.Join(home, ".fonts"),
			)
		}

		dirs = append(dirs, "/usr/local/share/fonts", "/usr/share/fonts")
		return cleanDirs(dirs)
	}
}

func cleanDirs(dirs []string) []string {
	out := make([]string, 0, len(dirs))
	seen := make(map[string]struct{}, len(dirs))

	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}

		dir = filepath.Clean(dir)
		if _, ok := seen[dir]; ok {
			continue
		}

		seen[dir] = struct{}{}
		out = append(out, dir)
	}

	return out
}
