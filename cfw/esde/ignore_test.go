package esde

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMarkDirectoryIgnoredCreatesMarker(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "update")

	if err := MarkDirectoryIgnored(dir); err != nil {
		t.Fatalf("MarkDirectoryIgnored failed: %v", err)
	}

	marker := filepath.Join(dir, IgnoreMarkerName)
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("expected %s to exist: %v", marker, err)
	}
}

func TestMarkDirectoryIgnoredIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, IgnoreMarkerName)
	// Pre-existing marker with content should be left untouched.
	if err := os.WriteFile(marker, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := MarkDirectoryIgnored(dir); err != nil {
		t.Fatalf("MarkDirectoryIgnored failed: %v", err)
	}

	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Errorf("existing marker should be left as-is, got %q", data)
	}
}
