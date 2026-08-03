package filetil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		size int64
		want string
	}{
		{0, "0.00 B"},
		{512, "512.00 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
	}
	for _, tc := range cases {
		got := FormatBytes(tc.size)
		if got != tc.want {
			t.Fatalf("FormatBytes(%d): got %q want %q", tc.size, got, tc.want)
		}
	}
}

func TestIsImageExt(t *testing.T) {
	cases := []struct {
		file string
		want bool
	}{
		{"photo.jpg", true},
		{"photo.JPEG", true},
		{"x.png", true},
		{"doc.pdf", false},
		{"noext", false},
	}
	for _, tc := range cases {
		got := IsImageExt(tc.file)
		if got != tc.want {
			t.Fatalf("IsImageExt(%q): got %v want %v", tc.file, got, tc.want)
		}
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.txt")
	if FileExists(missing) {
		t.Fatal("FileExists should be false for missing file")
	}
	existing := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(existing, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if !FileExists(existing) {
		t.Fatal("FileExists should be true for existing file")
	}
}

func TestRound(t *testing.T) {
	cases := []struct {
		val    float64
		places int
		want   float64
	}{
		{1.234, 2, 1.23},
		{1.235, 2, 1.24},
		{-1.235, 2, -1.24},
		{0, 2, 0},
	}
	for _, tc := range cases {
		got := Round(tc.val, tc.places)
		if got != tc.want {
			t.Fatalf("Round(%v, %d): got %v want %v", tc.val, tc.places, got, tc.want)
		}
	}
}

func TestCopyFileAndFileExists(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "sub", "dst.txt")
	content := []byte("copy me")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	if !FileExists(dst) {
		t.Fatal("destination should exist after CopyFile")
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(content) {
		t.Fatalf("copied content: got %q want %q", data, content)
	}
}

func TestReadFileAndIgnoreUTF8BOM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bom.txt")
	body := []byte{0xef, 0xbb, 0xbf, 'h', 'i'}
	if err := os.WriteFile(path, body, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFileAndIgnoreUTF8BOM(path)
	if err != nil {
		t.Fatalf("ReadFileAndIgnoreUTF8BOM: %v", err)
	}
	if string(got) != "hi" {
		t.Fatalf("BOM stripped: got %q want %q", got, "hi")
	}
	noBOM := filepath.Join(dir, "plain.txt")
	if err := os.WriteFile(noBOM, []byte("plain"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err = ReadFileAndIgnoreUTF8BOM(noBOM)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "plain" {
		t.Fatalf("no BOM: got %q want %q", got, "plain")
	}
}
