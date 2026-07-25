package esde

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
)

// systemsWithoutM3USupport lists ES-DE systems whose default emulator cannot load
// .m3u playlists, so a multi-disc game must launch a disc image directly. PS2
// (standalone PCSX2) is the main case — disc swapping is done from within the
// emulator. Keyed by RomM fs_slug.
var systemsWithoutM3USupport = map[string]bool{
	"ps2": true,
}

// SupportsM3U reports whether a system's default ES-DE emulator can load an .m3u
// playlist for multi-disc games.
func SupportsM3U(fsSlug string) bool {
	return !systemsWithoutM3USupport[fsSlug]
}

// OrganizeMultiFileRom presents an extracted multi-disc ROM as a single ES-DE
// entry instead of one entry per disc, and returns the path the gamelist entry
// should point at.
//
// It uses ES-DE's "directories interpreted as files" feature: a directory whose
// name has a supported file extension is shown as one game entry (not a folder),
// and on launch ES-DE passes the file inside it whose name matches the directory
// name to the emulator. So the extracted folder is simply renamed to the launch
// file's name:
//
//   - m3u-capable systems -> "<game>.m3u", launching the playlist (which handles
//     disc swapping);
//   - systems without .m3u support (e.g. PS2/PCSX2) -> the primary disc's file
//     name, launching disc 1. All discs stay together inside the directory so the
//     player can swap discs from within the emulator.
//
// extractDir is <romDirectory>/<gameName>, the flat directory the archive was
// unzipped into. This is the ES-DE counterpart to muos.OrganizeMultiFileRom.
func OrganizeMultiFileRom(extractDir, romDirectory, gameName string, supportsM3U bool) (string, error) {
	logger := gaba.GetLogger()

	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", fmt.Errorf("failed to read extracted directory: %w", err)
	}

	var m3uName string
	var discFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.EqualFold(filepath.Ext(name), ".m3u") {
			if m3uName == "" {
				m3uName = name
			}
			continue
		}
		discFiles = append(discFiles, name)
	}

	// Decide which inner file ES-DE should launch, and hence what to name the
	// interpreted directory.
	var launchFile string
	switch {
	case supportsM3U && m3uName != "":
		// Give the playlist a predictable name matching the game so the directory
		// (and thus the ES-DE entry basename) is clean.
		launchFile = gameName + ".m3u"
		if m3uName != launchFile {
			if err := os.Rename(filepath.Join(extractDir, m3uName), filepath.Join(extractDir, launchFile)); err != nil {
				return "", fmt.Errorf("failed to rename playlist to %s: %w", launchFile, err)
			}
		}
	case len(discFiles) > 0:
		launchFile = choosePrimaryDisc(extractDir, m3uName, discFiles)
	default:
		// Nothing launchable inside: hide the directory (ES-DE, like the OS, skips
		// dot-prefixed names) so its contents don't each become an entry.
		hidden := filepath.Join(romDirectory, "."+gameName)
		if err := os.Rename(extractDir, hidden); err != nil {
			return "", fmt.Errorf("failed to hide directory %s: %w", hidden, err)
		}
		logger.Debug("ES-DE multi-disc: no launchable file, hid directory", "dir", hidden)
		return "", nil
	}

	interpreted := filepath.Join(romDirectory, launchFile)
	if err := os.Rename(extractDir, interpreted); err != nil {
		return "", fmt.Errorf("failed to create interpreted directory %s: %w", interpreted, err)
	}
	logger.Debug("ES-DE multi-disc: interpreted directory as file", "dir", interpreted, "launch", launchFile)
	return interpreted, nil
}

// choosePrimaryDisc picks the disc to launch: the first entry listed in the .m3u
// (disc 1) when it matches an extracted file, otherwise the first disc by name.
func choosePrimaryDisc(extractDir, m3uName string, discFiles []string) string {
	if m3uName != "" {
		if data, err := os.ReadFile(filepath.Join(extractDir, m3uName)); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				if base := filepath.Base(line); slices.Contains(discFiles, base) {
					return base
				}
				break // first playlist entry didn't match a file; fall back to sort
			}
		}
	}
	sorted := append([]string(nil), discFiles...)
	slices.Sort(sorted)
	return sorted[0]
}
