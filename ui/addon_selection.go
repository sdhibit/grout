package ui

import (
	"errors"
	"fmt"
	"strings"

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

// AddonSelectionResult reports the outcome of the add-on picker. Confirmed is
// false when the user backed out (the download should be cancelled). The base
// game is always downloaded and is not represented here.
type AddonSelectionResult struct {
	Confirmed       bool
	SelectedFileIDs []int
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

// Draw shows a checklist of the game's add-ons (updates, DLC, …), all pre-checked,
// for the user to choose which to download alongside the base game (which is
// always downloaded). Returns the chosen add-on file IDs.
func (s *AddonSelectionScreen) Draw(game romm.Rom) (AddonSelectionResult, error) {
	items := make([]gaba.MenuItem, 0)

	// Show the base game first as an always-included, non-toggleable row so it's
	// obvious which file is the base versus the add-ons. It carries no Metadata,
	// so it's ignored when collecting the selected add-on IDs.
	baseLabel := i18n.Localize(&goi18n.Message{ID: "addon_category_base", Other: "Base"}, nil)
	for _, f := range game.BaseFiles() {
		items = append(items, gaba.MenuItem{
			Text:               fmt.Sprintf("[%s] %s", baseLabel, f.FileName),
			Selected:           true,
			NotMultiSelectable: true,
		})
	}

	addonCount := 0
	for _, group := range game.AddonGroups() {
		label := addonCategoryLabel(group.Category)
		for _, f := range group.Files {
			items = append(items, gaba.MenuItem{
				Text:     fmt.Sprintf("[%s] %s", label, f.FileName),
				Selected: true, // default: download everything; user unchecks what they don't want
				Metadata: f.ID,
			})
			addonCount++
		}
	}

	// No categorized add-ons: nothing to choose, proceed with just the base.
	if addonCount == 0 {
		return AddonSelectionResult{Confirmed: true}, nil
	}

	options := gaba.DefaultListOptions(
		i18n.Localize(&goi18n.Message{ID: "addon_selection_title", Other: "Add-ons to Download"}, nil),
		items,
	)
	options.UseSmallTitle = true
	options.InitialMultiSelectMode = true
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

	res := AddonSelectionResult{Confirmed: true}
	for _, idx := range result.Selected {
		if idx >= 0 && idx < len(items) {
			if id, ok := items[idx].Metadata.(int); ok {
				res.SelectedFileIDs = append(res.SelectedFileIDs, id)
			}
		}
	}
	return res, nil
}
