package ui

import (
	"errors"
	"strings"

	"grout/internal"
	"grout/internal/fileutil"
	"grout/romm"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	gabaconst "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

type AddonSelectionScreen struct{}

func NewAddonSelectionScreen() *AddonSelectionScreen {
	return &AddonSelectionScreen{}
}

// AddonSelectionInput carries what the add-on picker needs to show each file's
// download state. Config resolves the ROM/add-on directories and Platform is the
// game's platform (derived from the game when unset).
type AddonSelectionInput struct {
	Game     romm.Rom
	Config   *internal.Config
	Platform romm.Platform
}

// AddonSelectionResult reports the outcome of the add-on picker. Confirmed is
// false when the user backed out (the download should be cancelled).
// SelectedFileIDs holds every file the user chose to (re)download — base files
// included — so an already-downloaded base can be left in place while a new
// add-on is added. It is non-nil whenever the picker ran, even if empty, which
// signals the download planner to fetch exactly this set rather than defaulting
// to the base game.
type AddonSelectionResult struct {
	Confirmed       bool
	SelectedFileIDs []int
}

// addonFileLocations maps each of the game's file IDs to the absolute path it
// would be downloaded to, using the exact same placement logic as the download
// itself (base at the ROM dir, add-ons in their category/custom subfolder). This
// lets the picker check whether each file is already on disk.
func addonFileLocations(game romm.Rom, config *internal.Config, platform romm.Platform, romDir string) map[int]string {
	all := make(map[int]bool, len(game.Files))
	for _, f := range game.Files {
		all[f.ID] = true
	}
	plan := planRomDownloads(game, all, romDir, func(cat romm.RomFileCategory) string {
		return config.AddonDestination(platform, cat, romDir)
	})
	locations := make(map[int]string, len(plan))
	for _, p := range plan {
		locations[p.FileID] = p.Location
	}
	return locations
}

// addonSectionHeader formats a group's label (Base, Update, DLC, …) as a
// non-selectable header row that visually introduces the files beneath it.
func addonSectionHeader(label string) string {
	return strings.ToUpper(label)
}

// addonItemText renders a file row, indented so it reads as nested under its
// section header and prefixing a download icon when the file is already on disk
// so it's clear why it defaults to unchecked. Keeping the category on the header
// (not on every row) also shortens each row, which the list renders far more
// cheaply — long, per-row text is what makes this screen chug.
func addonItemText(fileName string, downloaded bool) string {
	prefix := "  "
	if downloaded {
		prefix += gabaconst.Download + " "
	}
	return prefix + fileName
}

func addonCategoryLabel(c romm.RomFileCategory) string {
	switch c {
	case romm.RomFileUpdate:
		return i18n.Localize(&goi18n.Message{ID: "addon_category_update", Other: "Update"}, nil)
	case romm.RomFileDLC:
		return i18n.Localize(&goi18n.Message{ID: "addon_category_dlc", Other: "DLC"}, nil)
	case romm.RomFilePatch:
		return i18n.Localize(&goi18n.Message{ID: "addon_category_patch", Other: "Patch"}, nil)
	case romm.RomFileHack:
		return i18n.Localize(&goi18n.Message{ID: "addon_category_hack", Other: "Hack"}, nil)
	case romm.RomFileTranslation:
		return i18n.Localize(&goi18n.Message{ID: "addon_category_translation", Other: "Translation"}, nil)
	case romm.RomFileMod:
		return i18n.Localize(&goi18n.Message{ID: "addon_category_mod", Other: "Mod"}, nil)
	default:
		s := string(c)
		if s == "" {
			return s
		}
		return strings.ToUpper(s[:1]) + s[1:]
	}
}

// Draw shows a checklist of the game's files — the base game plus its add-ons
// (updates, DLC, …) — for the user to choose which to download. Files not yet on
// disk are pre-checked; files already downloaded are marked and default to
// unchecked (so the base isn't needlessly re-fetched just to add an add-on) but
// stay toggleable so any file can be re-downloaded. Returns the chosen file IDs.
func (s *AddonSelectionScreen) Draw(input AddonSelectionInput) (AddonSelectionResult, error) {
	game := input.Game

	// Resolve the platform for directory lookups, falling back to the game's own
	// platform fields when the caller didn't carry a fully-populated platform
	// (e.g. from an all-platforms or collection view).
	platform := input.Platform
	if platform.ID == 0 && game.PlatformID != 0 {
		platform = romm.Platform{
			ID:     game.PlatformID,
			FSSlug: game.PlatformFSSlug,
			Name:   game.PlatformDisplayName,
		}
	}

	romDir := input.Config.GetPlatformRomDirectory(platform)
	locations := addonFileLocations(game, input.Config, platform, romDir)
	isDownloaded := func(fileID int) bool {
		loc, ok := locations[fileID]
		return ok && fileutil.FileExists(loc)
	}

	items := make([]gaba.MenuItem, 0)

	// Build the checklist as labeled sections (Base, then Update/DLC/…). Each
	// section gets a non-selectable header row and its files are indented beneath
	// it, so the groups are easy to tell apart at a glance. Header rows carry no
	// Metadata, so they're ignored when collecting the selected file IDs.
	appendSection := func(header string, files []romm.RomFile) {
		if len(files) == 0 {
			return
		}
		items = append(items, gaba.MenuItem{
			Text:               addonSectionHeader(header),
			NotMultiSelectable: true,
		})
		for _, f := range files {
			downloaded := isDownloaded(f.ID)
			items = append(items, gaba.MenuItem{
				Text:     addonItemText(f.FileName, downloaded),
				Selected: !downloaded, // pre-check what isn't downloaded; user unchecks the rest
				Metadata: f.ID,
			})
		}
	}

	// Base game first (now toggleable: leave it unchecked when it's already on
	// disk to add an add-on without re-fetching it, or check it to re-download),
	// then each add-on category in its stable display order.
	baseLabel := i18n.Localize(&goi18n.Message{ID: "addon_category_base", Other: "Base"}, nil)
	appendSection(baseLabel, game.BaseFiles())
	for _, group := range game.AddonGroups() {
		appendSection(addonCategoryLabel(group.Category), group.Files)
	}

	options := gaba.DefaultListOptions(
		i18n.Localize(&goi18n.Message{ID: "addon_selection_title", Other: "Files to Download"}, nil),
		items,
	)
	options.UseSmallTitle = true
	options.InitialMultiSelectMode = true
	// Start focus on the first real file rather than the leading "BASE" header
	// (index 0), which isn't selectable.
	options.SelectedIndex = 1
	// A toggles the highlighted item's checkbox (gabagool's built-in multi-select
	// toggle). We deliberately do NOT bind MultiSelectButton to A: that button
	// toggles multi-select *mode* off, which clears every checkbox and drops the
	// list out of multi-select on the first A press.
	options.MultiSelectConfirmButton = gabaconst.VirtualButtonStart
	options.SelectAllButton = gabaconst.VirtualButtonR1
	options.DeselectAllButton = gabaconst.VirtualButtonL1
	options.FooterHelpItems = []gaba.FooterHelpItem{
		FooterBack(),
		{ButtonName: "A", HelpText: i18n.Localize(&goi18n.Message{ID: "addon_toggle", Other: "Toggle"}, nil)},
		{ButtonName: "L1", HelpText: i18n.Localize(&goi18n.Message{ID: "addon_none", Other: "None"}, nil)},
		{ButtonName: "R1", HelpText: i18n.Localize(&goi18n.Message{ID: "addon_all", Other: "All"}, nil)},
		{ButtonName: "Start", HelpText: i18n.Localize(&goi18n.Message{ID: "button_download", Other: "Download"}, nil)},
	}

	result, err := gaba.List(options)
	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return AddonSelectionResult{Confirmed: false}, nil
		}
		return AddonSelectionResult{}, err
	}

	// Non-nil even when empty: this signals downstream that the picker ran, so the
	// planner fetches exactly this set instead of defaulting to the base game.
	res := AddonSelectionResult{Confirmed: true, SelectedFileIDs: []int{}}
	for _, idx := range result.Selected {
		if idx >= 0 && idx < len(items) {
			if id, ok := items[idx].Metadata.(int); ok {
				res.SelectedFileIDs = append(res.SelectedFileIDs, id)
			}
		}
	}
	return res, nil
}
