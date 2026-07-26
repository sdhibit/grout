package esde

import (
	"fmt"
	"os"
	"path/filepath"
)

// IgnoreMarkerName is ES-DE's per-directory "don't scan this folder" marker. When
// a directory contains a file with this name, ES-DE skips it entirely during its
// game scan — regardless of the "Show hidden files and folders" setting and the
// directory's name. It's the documented way to keep support files (here: add-on
// folders such as update/ and dlc/) inside the ROM tree without them showing up
// as separate game entries.
const IgnoreMarkerName = "noload.txt"

// MarkDirectoryIgnored drops an empty noload.txt into dir so ES-DE skips it when
// scanning for games. It creates dir if needed and is a no-op when the marker is
// already present.
func MarkDirectoryIgnored(dir string) error {
	marker := filepath.Join(dir, IgnoreMarkerName)
	if _, err := os.Stat(marker); err == nil {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	if err := os.WriteFile(marker, nil, 0644); err != nil {
		return fmt.Errorf("failed to write ES-DE ignore marker %s: %w", marker, err)
	}
	return nil
}
