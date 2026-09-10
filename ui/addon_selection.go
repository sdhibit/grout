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

// addonLabelMaxRunes caps the visible length of a file row's name. gabagool's
// list re-rasterizes and re-truncates every visible row's text every frame (and
// scrolls the focused one), so long names tank the framerate. Keeping rows short
// avoids that per-frame work; the full name is a button-press away (see the file-
// names view). The budget is conservative so labels fit without scrolling on the
// narrower target screens.
const addonLabelMaxRunes = 40

// addonItemText renders a file row, indented so it reads as nested under its
// section header and prefixing a download icon when the file is already on disk
// so it's clear why it defaults to unchecked.
func addonItemText(fileName string, downloaded bool) string {
	prefix := "  "
	if downloaded {
		prefix += gabaconst.Download + " "
	}
	return prefix + fileName
}

// middleEllipsize shortens s to at most max runes by dropping the middle and
// joining the head and tail with "…", keeping both ends visible so files that
// differ only near the end (e.g. "(DLC Pack 2)") stay distinguishable.
func middleEllipsize(s string, max int) string {
	r := []rune(s)
	if max < 5 || len(r) <= max {
		return s
	}
	const ell = "..."
	keep := max - len(ell)
	head := (keep + 1) / 2
	tail := keep - head
	return string(r[:head]) + ell + string(r[len(r)-tail:])
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

	// The game's files as labeled sections (Base first, then each add-on category
	// in its stable order), computed once. Each section becomes a non-selectable
	// header row with its files indented beneath.
	type section struct {
		header string
		files  []romm.RomFile
	}
	var sections []section
	baseLabel := i18n.Localize(&goi18n.Message{ID: "addon_category_base", Other: "Base"}, nil)
	if bf := game.BaseFiles(); len(bf) > 0 {
		sections = append(sections, section{baseLabel, bf})
	}
	for _, group := range game.AddonGroups() {
		sections = append(sections, section{addonCategoryLabel(group.Category), group.Files})
	}

	// On cumulative-update platforms (Switch) the newest update supersedes the
	// older ones, so only it should default to checked. Older updates stay listed
	// and toggleable, just unchecked.
	latestUpdateID, collapseUpdates := 0, false
	if game.UpdatesCumulative() {
		if latest, ok := game.LatestUpdateFile(); ok {
			latestUpdateID, collapseUpdates = latest.ID, true
		}
	}

	// Checkbox state keyed by file ID, so it survives rebuilding the list when the
	// user pops the file-names view. Defaults to checking what isn't downloaded yet,
	// minus superseded updates on a cumulative-update platform.
	checked := make(map[int]bool)
	for _, sec := range sections {
		for _, f := range sec.files {
			defaultOn := !isDownloaded(f.ID)
			if collapseUpdates && f.Category == romm.RomFileUpdate {
				defaultOn = defaultOn && f.ID == latestUpdateID
			}
			checked[f.ID] = defaultOn
		}
	}

	// showFileNames opens a read-only, scrollable view of the untruncated file
	// names — the way to see a full name the picker rows ellipsize away.
	showFileNames := func() {
		items := make([]gaba.MenuItem, 0)
		for _, sec := range sections {
			items = append(items, gaba.MenuItem{Text: addonSectionHeader(sec.header), NotMultiSelectable: true})
			for _, f := range sec.files {
				items = append(items, gaba.MenuItem{Text: "  " + f.FileName})
			}
		}
		opts := gaba.DefaultListOptions(
			i18n.Localize(&goi18n.Message{ID: "addon_file_names_title", Other: "File Names"}, nil),
			items,
		)
		opts.UseSmallTitle = true
		opts.SelectedIndex = 1
		opts.FooterHelpItems = []gaba.FooterHelpItem{FooterBack()}
		gaba.List(opts) // read-only: any exit returns to the picker
	}

	for {
		// Build the checklist fresh each pass so it reflects the preserved checkbox
		// state. Names are middle-ellipsized to keep every row short — long rows are
		// what make gabagool's list re-render endlessly and drop frames.
		items := make([]gaba.MenuItem, 0)
		for _, sec := range sections {
			items = append(items, gaba.MenuItem{Text: addonSectionHeader(sec.header), NotMultiSelectable: true})
			for _, f := range sec.files {
				items = append(items, gaba.MenuItem{
					Text:     addonItemText(middleEllipsize(f.FileName, addonLabelMaxRunes), isDownloaded(f.ID)),
					Selected: checked[f.ID],
					Metadata: f.ID,
				})
			}
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
		// X opens the full file-names view (the picker rows are ellipsized).
		options.ActionButton = gabaconst.VirtualButtonX
		// Assign explicit footer groups so gabagool renders every hint. In its
		// default (auto) mode the footer keeps only the first 2 items per side and
		// silently drops the rest — with this many hints that would hide All and
		// Download. The A-to-toggle hint is dropped on purpose: checking items with
		// A is self-evident in a checklist, and cutting it keeps the footer narrow
		// enough to fit the remaining hints on smaller screens.
		options.FooterHelpItems = []gaba.FooterHelpItem{
			FooterBack(),
			{ButtonName: "X", HelpText: i18n.Localize(&goi18n.Message{ID: "addon_view_names", Other: "Names"}, nil), Group: gaba.FooterGroupLeft},
			{ButtonName: "L1", HelpText: i18n.Localize(&goi18n.Message{ID: "addon_none", Other: "None"}, nil), Group: gaba.FooterGroupRight},
			{ButtonName: "R1", HelpText: i18n.Localize(&goi18n.Message{ID: "addon_all", Other: "All"}, nil), Group: gaba.FooterGroupRight},
			{ButtonName: "Start", HelpText: i18n.Localize(&goi18n.Message{ID: "button_download", Other: "Download"}, nil), Group: gaba.FooterGroupRight},
		}

		result, err := gaba.List(options)
		if err != nil {
			if errors.Is(err, gaba.ErrCancelled) {
				return AddonSelectionResult{Confirmed: false}, nil
			}
			return AddonSelectionResult{}, err
		}

		// Sync the preserved checkbox state from what's currently checked.
		for id := range checked {
			checked[id] = false
		}
		for _, idx := range result.Selected {
			if idx >= 0 && idx < len(items) {
				if id, ok := items[idx].Metadata.(int); ok {
					checked[id] = true
				}
			}
		}

		// X: peek at the full names, then reopen the picker with checks intact.
		if result.Action == gaba.ListActionTriggered {
			showFileNames()
			continue
		}

		// Confirmed. Collect selected file IDs in section order. Non-nil even when
		// empty: this signals downstream that the picker ran, so the planner fetches
		// exactly this set instead of defaulting to the base game.
		res := AddonSelectionResult{Confirmed: true, SelectedFileIDs: []int{}}
		for _, sec := range sections {
			for _, f := range sec.files {
				if checked[f.ID] {
					res.SelectedFileIDs = append(res.SelectedFileIDs, f.ID)
				}
			}
		}
		return res, nil
	}
}
