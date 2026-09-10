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
	// Download reports whether the file should actually be fetched. Base files
	// are always present in the plan because they anchor the gamelist/artwork
	// entry, but are only fetched when Download is true — e.g. the base is
	// already on disk and the user only wants to add an update or DLC.
	Download bool
}

// autoAddonSelection selects a categorized ROM's base file plus its supplemental
// add-ons for a bulk download (which runs without the add-on picker). All DLC and
// patches are taken; updates follow the platform's update strategy — on a
// cumulative-update platform (Switch) only the newest update is taken, since it
// supersedes the earlier ones. The returned map feeds planRomDownloads exactly
// like a picker result would.
func autoAddonSelection(rom romm.Rom) map[int]bool {
	latestUpdateID, collapseUpdates := 0, false
	if rom.UpdatesCumulative() {
		if latest, ok := rom.LatestUpdateFile(); ok {
			latestUpdateID, collapseUpdates = latest.ID, true
		}
	}

	selected := make(map[int]bool, len(rom.Files))
	for _, f := range rom.Files {
		switch {
		case f.IsBase():
			selected[f.ID] = true
		case f.Category == romm.RomFileUpdate:
			// Skip superseded updates when the platform's updates are cumulative.
			if !collapseUpdates || f.ID == latestUpdateID {
				selected[f.ID] = true
			}
		case f.Category.IsSupplementalAddon(): // DLC, patches: always kept.
			selected[f.ID] = true
		}
	}
	return selected
}

// planRomDownloads expands a categorized multi-part ROM into the concrete files
// to download given the user's add-on selection.
//
// Base-game files (category empty or "game") download into romDir. Selected
// add-ons default to a category subfolder under romDir (update/, dlc/, …),
// mirroring RomM's own layout so most emulators find them without extra setup.
// addonDestFor optionally overrides the destination directory for a category
// (used later to redirect Switch updates/DLC to an emulator-specific folder);
// returning "" from it keeps the default.
//
// selected is keyed by RomFile ID and controls what is fetched:
//   - nil            → no add-on picker was shown (bulk or non-categorized
//     download): fetch the base game and no add-ons.
//   - non-nil (map)  → the exact add-on picker result: every file, base
//     included, is fetched only if its ID is present. This lets an
//     already-downloaded base stay in place while the user adds an add-on.
//
// Base files are always present in the returned plan (flagged IsBase) so the
// caller can anchor the gamelist/artwork entry to them; the Download flag says
// whether to actually fetch each one.
func planRomDownloads(
	rom romm.Rom,
	selected map[int]bool,
	romDir string,
	addonDestFor func(romm.RomFileCategory) string,
) []plannedDownload {
	pickerRan := selected != nil
	planned := make([]plannedDownload, 0, len(rom.Files))
	for _, f := range rom.Files {
		switch {
		case f.IsBase():
			download := true
			if pickerRan {
				download = selected[f.ID]
			}
			planned = append(planned, plannedDownload{
				FileID:   f.ID,
				FileName: f.FileName,
				Location: filepath.Join(romDir, f.FileName),
				IsBase:   true,
				Download: download,
			})
		case f.Category.IsSupplementalAddon() && selected[f.ID]:
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
				Download: true,
			})
		}
	}
	return planned
}
