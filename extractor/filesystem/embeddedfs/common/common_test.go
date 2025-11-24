package common

import (
	"archive/tar"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	scalibrfs "github.com/google/osv-scalibr/fs"
	"www.velocidex.com/golang/go-ntfs/parser"
)

func TestDetectFilesystem(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		offset   int64
		expected string
	}{
		{
			name: "ext4",
			data: func() []byte {
				b := make([]byte, 4096)
				b[0x438] = 0x53
				b[0x439] = 0xEF
				return b
			}(),
			expected: "ext4",
		},
		{
			name: "NTFS",
			data: func() []byte {
				b := make([]byte, 4096)
				copy(b[3:], "NTFS    ")
				return b
			}(),
			expected: "NTFS",
		},
		{
			name: "FAT32",
			data: func() []byte {
				b := make([]byte, 4096)
				copy(b[0x52:], "FAT32   ")
				return b
			}(),
			expected: "FAT32",
		},
		{
			name: "exFAT",
			data: func() []byte {
				b := make([]byte, 4096)
				copy(b[3:], "EXFAT   ")
				return b
			}(),
			expected: "exFAT",
		},
		{
			name:     "unknown",
			data:     make([]byte, 4096),
			expected: "unknown",
		},
		{
			name:     "short buffer",
			data:     make([]byte, 10),
			expected: "read error: EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			if got := DetectFilesystem(r, 0); got != tt.expected {
				t.Errorf("DetectFilesystem() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEmbeddedDirFS(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "embeddedfs-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	dummyFile, err := os.CreateTemp("", "dummy")
	if err != nil {
		t.Fatalf("failed to create dummy file: %v", err)
	}
	defer os.Remove(dummyFile.Name())

	refCount := int32(1)
	refMu := &sync.Mutex{}
	efs := &EmbeddedDirFS{
		FS:       scalibrfs.DirFS(tmpDir),
		File:     dummyFile,
		TmpPaths: []string{tmpDir},
		RefCount: &refCount,
		RefMu:    refMu,
	}

	// Test Open
	f, err := efs.Open("test.txt")
	if err != nil {
		t.Errorf("Open() error = %v", err)
	} else {
		f.Close()
	}

	// Test ReadDir
	entries, err := efs.ReadDir(".")
	if err != nil {
		t.Errorf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "test.txt" {
		t.Errorf("ReadDir() unexpected entries: %v", entries)
	}

	// Test Stat
	fi, err := efs.Stat("test.txt")
	if err != nil {
		t.Errorf("Stat() error = %v", err)
	}
	if fi.Name() != "test.txt" {
		t.Errorf("Stat() unexpected name: %v", fi.Name())
	}

	// Test Stat root
	fi, err = efs.Stat("/")
	if err != nil {
		t.Errorf("Stat(/) error = %v", err)
	}
	if !fi.IsDir() {
		t.Errorf("Stat(/) should be directory")
	}

	// Test TempPaths
	if len(efs.TempPaths()) != 1 || efs.TempPaths()[0] != tmpDir {
		t.Errorf("TempPaths() unexpected paths: %v", efs.TempPaths())
	}

	// Test Close
	if err := efs.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	if refCount != 0 {
		t.Errorf("Close() refCount = %v, want 0", refCount)
	}
}

func TestTARToTempDir(t *testing.T) {
	// Create a tar archive in memory
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add a file
	content := []byte("hello world")
	hdr := &tar.Header{
		Name: "test.txt",
		Mode: 0600,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}

	// Add a directory
	hdr = &tar.Header{
		Name:     "dir/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write header: %v", err)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}

	// Test extraction
	tmpDir, err := TARToTempDir(&buf)
	if err != nil {
		t.Fatalf("TARToTempDir() error = %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Verify file exists
	if _, err := os.Stat(filepath.Join(tmpDir, "test.txt")); err != nil {
		t.Errorf("file test.txt not found: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(filepath.Join(tmpDir, "dir")); err != nil {
		t.Errorf("directory dir not found: %v", err)
	}
}

func TestFileInfo(t *testing.T) {
	fi := &fileInfo{
		name:    "root",
		isDir:   true,
		modTime: time.Now(),
	}

	if fi.Name() != "root" {
		t.Errorf("Name() = %v, want root", fi.Name())
	}
	if fi.Size() != 0 {
		t.Errorf("Size() = %v, want 0", fi.Size())
	}
	if fi.Mode() != os.ModeDir|0755 {
		t.Errorf("Mode() = %v, want %v", fi.Mode(), os.ModeDir|0755)
	}
	if !fi.IsDir() {
		t.Errorf("IsDir() = false, want true")
	}
	if fi.Sys() != nil {
		t.Errorf("Sys() = %v, want nil", fi.Sys())
	}
	if fi.ModTime().IsZero() {
		t.Errorf("ModTime() is zero")
	}
}

func TestTARToTempDir_Invalid(t *testing.T) {
	// Create a tar archive with invalid entry (outside root)
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	hdr := &tar.Header{
		Name: "../test.txt",
		Mode: 0600,
		Size: 0,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write header: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}

	_, err := TARToTempDir(&buf)
	if err == nil {
		t.Error("TARToTempDir() expected error for invalid entry")
	}
}

func TestTARToTempDir_ReadError(t *testing.T) {
	// Create a reader that fails
	r := &errorReader{}
	_, err := TARToTempDir(r)
	if err == nil {
		t.Error("TARToTempDir() expected error for read error")
	}
}

type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"path/to/file", "/path/to/file"},
		{"/path/to/file", "/path/to/file"},
		{"path\\to\\file", "/path/to/file"},
		{"\\path\\to\\file", "/path/to/file"},
	}

	for _, tt := range tests {
		if got := normalizePath(tt.input); got != tt.expected {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

type mockFileInfo struct {
	name string
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0 }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() any           { return nil }

func TestFilterEntriesFat32(t *testing.T) {
	entries := []os.FileInfo{
		&mockFileInfo{name: "."},
		&mockFileInfo{name: ".."},
		&mockFileInfo{name: "lost+found"},
		&mockFileInfo{name: "file.txt"},
	}

	filtered := filterEntriesFat32(entries)
	if len(filtered) != 1 || filtered[0].Name() != "file.txt" {
		t.Errorf("filterEntriesFat32() = %v, want [file.txt]", filtered)
	}
}

type mockDirEntry struct {
	name string
}

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return false }
func (m *mockDirEntry) Type() os.FileMode          { return 0 }
func (m *mockDirEntry) Info() (os.FileInfo, error) { return nil, nil }

func TestFilterEntriesExt(t *testing.T) {
	entries := []fs.DirEntry{
		&mockDirEntry{name: "."},
		&mockDirEntry{name: ".."},
		&mockDirEntry{name: "lost+found"},
		&mockDirEntry{name: "file.txt"},
	}

	filtered := filterEntriesExt(entries)
	if len(filtered) != 1 || filtered[0].Name() != "file.txt" {
		t.Errorf("filterEntriesExt() = %v, want [file.txt]", filtered)
	}
}

func TestFilterEntriesNtfs(t *testing.T) {
	entries := []*parser.FileInfo{
		{Name: "."},
		{Name: ".."},
		{Name: "$MFT"},
		{Name: "file.txt"},
		{Name: ""},
	}

	filtered := filterEntriesNtfs(entries)
	if len(filtered) != 1 || filtered[0].Name != "file.txt" {
		t.Errorf("filterEntriesNtfs() = %v, want [file.txt]", filtered)
	}
}
