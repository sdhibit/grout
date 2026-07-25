package ui

import (
	"path/filepath"

	"grout/romm"
)

// plannedDownload is a single file to fetch for a ROM and the absolute path it
// should be written to.
type plannedDownload struct {
	FileID   int
	FileName string
	Location string
	// IsBase marks the base-game file, which the gamelist/artwork entry is
	// anchored to (add-ons don't get their own library entry).
	IsBase bool
}

// planRomDownloads expands a categorized multi-part ROM into the concrete files
// to download given the user's add-on selection.
//
// Base-game files (category empty or "game") always download into romDir.
// Selected add-ons default to a category subfolder under romDir (update/, dlc/,
// …), mirroring RomM's own layout so most emulators find them without extra
// setup. addonDestFor optionally overrides the destination directory for a
// category (used later to redirect Switch updates/DLC to an emulator-specific
// folder); returning "" from it keeps the default. selectedAddons is keyed by
// RomFile ID.
func planRomDownloads(
	rom romm.Rom,
	selectedAddons map[int]bool,
	romDir string,
	addonDestFor func(romm.RomFileCategory) string,
) []plannedDownload {
	planned := make([]plannedDownload, 0, len(rom.Files))
	for _, f := range rom.Files {
		switch {
		case f.IsBase():
			planned = append(planned, plannedDownload{
				FileID:   f.ID,
				FileName: f.FileName,
				Location: filepath.Join(romDir, f.FileName),
				IsBase:   true,
			})
		case f.Category.IsSupplementalAddon() && selectedAddons[f.ID]:
			dest := filepath.Join(romDir, string(f.Category))
			if addonDestFor != nil {
				if override := addonDestFor(f.Category); override != "" {
					dest = override
				}
			}
			planned = append(planned, plannedDownload{
				FileID:   f.ID,
				FileName: f.FileName,
				Location: filepath.Join(dest, f.FileName),
			})
		}
	}
	return planned
}
