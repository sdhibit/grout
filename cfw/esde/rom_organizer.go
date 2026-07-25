package esde

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
)

// systemsWithoutM3USupport lists ES-DE systems whose default emulator cannot load
// .m3u playlists, so a multi-disc game must expose a launchable disc image rather
// than a playlist. PS2 (standalone PCSX2) is the main case — disc swapping is done
// from within the emulator. Keyed by RomM fs_slug.
var systemsWithoutM3USupport = map[string]bool{
	"ps2": true,
}

// SupportsM3U reports whether a system's default ES-DE emulator can load an .m3u
// playlist for multi-disc games.
func SupportsM3U(fsSlug string) bool {
	return !systemsWithoutM3USupport[fsSlug]
}

// OrganizeMultiFileRom lays out an extracted multi-disc ROM the way ES-DE expects:
// a single game entry instead of one entry per disc. It returns the path the
// gamelist entry should point at.
//
// ES-DE builds its game list by scanning the system directory for every file with
// a supported extension, so a freshly extracted multi-disc game (its .m3u plus each
// .chd/.iso side by side) shows up as several entries. ES-DE — like the OS — skips
// files and directories whose name starts with a dot, so the disc images are moved
// into a dot-prefixed directory.
//
//   - When the emulator supports .m3u (supportsM3U), the rewritten playlist is left
//     in the system root and the discs are hidden beside it.
//   - When it does not (e.g. PS2/PCSX2), the primary disc is left in the system root
//     as the launchable entry and the remaining discs are hidden; the player swaps
//     discs from within the emulator.
//
// extractDir is <romDirectory>/<gameName>, the flat directory the archive was
// unzipped into. This mirrors muos.OrganizeMultiFileRom, which does the same for
// muOS using a "_" prefix.
func OrganizeMultiFileRom(extractDir, romDirectory, gameName string, supportsM3U bool) (string, error) {
	logger := gaba.GetLogger()

	hiddenName := "." + gameName
	hiddenDir := filepath.Join(romDirectory, hiddenName)

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

	if supportsM3U && m3uName != "" {
		return organizeWithM3U(extractDir, romDirectory, hiddenName, hiddenDir, gameName, m3uName, logger)
	}
	return organizeWithPrimaryDisc(extractDir, romDirectory, hiddenName, hiddenDir, m3uName, discFiles, logger)
}

// organizeWithM3U hides the whole extract directory and exposes only the playlist,
// rewritten so its entries resolve into the now-hidden directory. Returns the path
// of the root .m3u.
func organizeWithM3U(extractDir, romDirectory, hiddenName, hiddenDir, gameName, m3uName string, logger *slog.Logger) (string, error) {
	m3uSrc := filepath.Join(extractDir, m3uName)
	content, err := os.ReadFile(m3uSrc)
	if err != nil {
		return "", fmt.Errorf("failed to read .m3u file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Paths in an .m3u resolve relative to the playlist, which now sits in the
		// system root; point each disc entry into the hidden directory.
		lines[i] = filepath.Join(hiddenName, line)
	}

	m3uDest := filepath.Join(romDirectory, gameName+".m3u")
	if err := os.WriteFile(m3uDest, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return "", fmt.Errorf("failed to write updated .m3u file: %w", err)
	}
	if err := os.Remove(m3uSrc); err != nil {
		logger.Warn("Failed to remove original .m3u file", "path", m3uSrc, "error", err)
	}
	if err := os.Rename(extractDir, hiddenDir); err != nil {
		return "", fmt.Errorf("failed to hide directory %s: %w", hiddenDir, err)
	}
	logger.Debug("Organized multi-disc ROM for ES-DE (m3u)", "m3u", m3uDest, "hiddenDir", hiddenDir)
	return m3uDest, nil
}

// organizeWithPrimaryDisc leaves the first disc in the system root as the launchable
// entry and moves everything else (other discs, any unusable .m3u) into the hidden
// directory. Returns the path of the primary disc. Note: single-file disc images
// (.chd/.iso) are assumed — a .cue that references a separate .bin is not rewritten.
func organizeWithPrimaryDisc(extractDir, romDirectory, hiddenName, hiddenDir, m3uName string, discFiles []string, logger *slog.Logger) (string, error) {
	// Nothing launchable to promote — just hide the directory so its contents don't
	// each become an entry.
	if len(discFiles) == 0 {
		if err := os.Rename(extractDir, hiddenDir); err != nil {
			return "", fmt.Errorf("failed to hide directory %s: %w", hiddenDir, err)
		}
		logger.Debug("No disc images found, hid multi-disc directory for ES-DE", "hiddenDir", hiddenDir)
		return "", nil
	}

	primary := choosePrimaryDisc(extractDir, m3uName, discFiles)

	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create hidden directory %s: %w", hiddenDir, err)
	}

	// Move the primary disc to the system root, everything else into the hidden dir.
	primaryDest := filepath.Join(romDirectory, primary)
	if err := os.Rename(filepath.Join(extractDir, primary), primaryDest); err != nil {
		return "", fmt.Errorf("failed to promote primary disc %s: %w", primary, err)
	}
	remaining, err := os.ReadDir(extractDir)
	if err != nil {
		return "", fmt.Errorf("failed to re-read extracted directory: %w", err)
	}
	for _, entry := range remaining {
		if err := os.Rename(filepath.Join(extractDir, entry.Name()), filepath.Join(hiddenDir, entry.Name())); err != nil {
			return "", fmt.Errorf("failed to hide %s: %w", entry.Name(), err)
		}
	}
	if err := os.Remove(extractDir); err != nil {
		logger.Warn("Failed to remove emptied extract directory", "path", extractDir, "error", err)
	}
	logger.Debug("Organized multi-disc ROM for ES-DE (primary disc)", "primary", primaryDest, "hiddenDir", hiddenDir)
	return primaryDest, nil
}

// choosePrimaryDisc picks the disc to expose as the game entry: the first entry
// listed in the .m3u (disc 1) when it matches an extracted file, otherwise the
// first disc by name.
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
