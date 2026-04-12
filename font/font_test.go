package font

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

func writeTempFont(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "Go Regular.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0o600); err != nil {
		t.Fatalf("write font: %v", err)
	}
	return path
}

func TestIsSupportedFontFile(t *testing.T) {
	if !IsSupportedFontFile("demo.TTF") {
		t.Fatal("expected supported font")
	}
	if IsSupportedFontFile("demo.png") {
		t.Fatal("unexpected supported extension")
	}
}

func TestNewEntryMatchAndLoad(t *testing.T) {
	path := writeTempFont(t)
	entries, err := ScanFile(path, nil)
	if err != nil {
		t.Fatalf("scan font: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("unexpected entry count: %d", len(entries))
	}

	entry := entries[0]
	if !entry.Match("GoRegular") || !entry.Match(path) || entry.Match("") {
		t.Fatal("unexpected match result")
	}

	loaded, err := entry.Load()
	if err != nil {
		t.Fatalf("load entry: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected font")
	}
}

func TestEntryLoadErrors(t *testing.T) {
	entry := Entry{Path: filepath.Join(t.TempDir(), "missing.ttf"), Index: 0}
	if _, err := entry.Load(); err == nil {
		t.Fatal("expected missing file error")
	}

	path := writeTempFont(t)
	entry = Entry{Path: path, Index: 10}
	if _, err := entry.Load(); err == nil {
		t.Fatal("expected invalid index error")
	}
}

func TestScanDirsAndHelpers(t *testing.T) {
	dir := t.TempDir()
	fontPath := filepath.Join(dir, "sub", "regular.ttf")
	if err := os.MkdirAll(filepath.Dir(fontPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fontPath, goregular.TTF, 0o600); err != nil {
		t.Fatalf("write font: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write text file: %v", err)
	}

	entries, err := ScanDirs([]string{"", filepath.Join(dir, "missing"), filepath.Join(dir, "ignore.txt"), dir})
	if err != nil {
		t.Fatalf("scan dirs: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("unexpected dir scan count: %d", len(entries))
	}

	got := cleanDirs([]string{" ", dir, dir, filepath.Clean(dir + "/sub/..")})
	if len(got) != 1 || got[0] != filepath.Clean(dir) {
		t.Fatalf("unexpected cleaned dirs: %v", got)
	}

	sysDirs := GetSystemDirs()
	if len(sysDirs) == 0 {
		t.Fatal("expected at least one system dir")
	}
}

func TestScanFileErrorsAndFallbacks(t *testing.T) {
	if _, err := ScanFile(filepath.Join(t.TempDir(), "missing.ttf"), nil); err == nil {
		t.Fatal("expected read error")
	}

	path := filepath.Join(t.TempDir(), "bad.ttf")
	if err := os.WriteFile(path, []byte("bad"), 0o600); err != nil {
		t.Fatalf("write bad font: %v", err)
	}
	if _, err := ScanFile(path, nil); err == nil {
		t.Fatal("expected parse error")
	}

	if got := normalizeLookupName(" /tmp/Go-Regular.TTF "); got != "goregular" {
		t.Fatalf("unexpected normalized name: %q", got)
	}
	if got := squashName(" Go-Regular 123 "); got != "goregular123" {
		t.Fatalf("unexpected squashed name: %q", got)
	}
}

func TestNewEntryWithoutFont(t *testing.T) {
	entry := NewEntry("/tmp/My Font.ttf", 0, nil)
	if entry.stemName != "My Font" || entry.normStemName != "myfont" {
		t.Fatalf("unexpected entry stem data: %#v", entry)
	}
}

func TestPreferName(t *testing.T) {
	otf, err := opentype.Parse(goregular.TTF)
	if err != nil {
		t.Fatalf("parse font: %v", err)
	}
	entry := NewEntry("Go-Regular.ttf", 0, otf)
	if entry.family == "" || entry.fullName == "" {
		t.Fatalf("expected font names: %#v", entry)
	}
}
