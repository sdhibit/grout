package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"grout/internal"
	"grout/romm"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	icons "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

// addonCapableSlugs are the platforms whose games commonly ship updates/DLC and
// whose emulators may want them in a dedicated folder. Only these are offered in
// the add-on directory settings, keeping the list focused (Switch is the primary
// case). Add-ons for any other platform still download to the default location.
var addonCapableSlugs = map[string]bool{
	"switch":  true,
	"ps3":     true,
	"ps4":     true,
	"psvita":  true,
	"3ds":     true,
	"wiiu":    true,
	"wii":     true,
	"xbox360": true,
}

// addonConfigurableCategories are the add-on types that get their own
// configurable destination row per platform.
var addonConfigurableCategories = []romm.RomFileCategory{romm.RomFileUpdate, romm.RomFileDLC}

type AddonDirectoriesInput struct {
	Config    *internal.Config
	Platforms []romm.Platform
}

type AddonDirectoriesOutput struct {
	Action AddonDirectoriesAction
	Config *internal.Config
}

type AddonDirectoriesAction int

const (
	AddonDirectoriesActionSaved AddonDirectoriesAction = iota
	AddonDirectoriesActionBack
)

// addonRow identifies which platform + category a settings row configures, and
// carries the effective default path so a value matching it clears the override.
type addonRow struct {
	Slug     string
	Category romm.RomFileCategory
	Default  string
}

type AddonDirectoriesScreen struct{}

func NewAddonDirectoriesScreen() *AddonDirectoriesScreen {
	return &AddonDirectoriesScreen{}
}

// Draw lets the user set, per add-on-capable platform, separate folders for that
// platform's updates and DLC (e.g. a Switch emulator's watched folders). Each
// row is pre-filled with the effective path; entering the default path (or
// clearing it) restores the default of a category subfolder next to the ROMs.
func (s *AddonDirectoriesScreen) Draw(input AddonDirectoriesInput) (AddonDirectoriesOutput, error) {
	config := input.Config
	output := AddonDirectoriesOutput{Action: AddonDirectoriesActionBack, Config: config}

	var platforms []romm.Platform
	seen := make(map[string]bool)
	for _, p := range input.Platforms {
		if addonCapableSlugs[p.FSSlug] && !seen[p.FSSlug] {
			platforms = append(platforms, p)
			seen[p.FSSlug] = true
		}
	}

	if len(platforms) == 0 {
		gaba.ConfirmationMessage(
			i18n.Localize(&goi18n.Message{ID: "addon_dirs_none", Other: "No add-on-capable platforms found in your library."}, nil),
			[]gaba.FooterHelpItem{FooterBack()},
			gaba.MessageOptions{},
		)
		return output, nil
	}

	items := make([]gaba.ItemWithOptions, 0, len(platforms)*len(addonConfigurableCategories))
	for _, p := range platforms {
		romDir := config.GetPlatformRomDirectory(p)
		for _, cat := range addonConfigurableCategories {
			def := filepath.Join(romDir, string(cat))

			// Effective path: the override if set, otherwise the default.
			effective := def
			if byCat := config.AddonDirectoryMappings[p.FSSlug]; byCat != nil {
				if v := byCat[string(cat)]; v != "" {
					effective = v
				}
			}

			items = append(items, gaba.ItemWithOptions{
				Item: gaba.MenuItem{
					Text:     fmt.Sprintf("%s — %s", p.Name, addonCategoryLabel(cat)),
					Metadata: addonRow{Slug: p.FSSlug, Category: cat, Default: def},
				},
				Options: []gaba.Option{{
					Type:           gaba.OptionTypeKeyboard,
					DisplayName:    effective,
					KeyboardPrompt: effective, // pre-fill the keyboard with a concrete path
					Value:          effective,
				}},
			})
		}
	}

	result, err := gaba.OptionsList(
		i18n.Localize(&goi18n.Message{ID: "settings_addon_dirs", Other: "Add-on Folders"}, nil),
		gaba.OptionListSettings{
			FooterHelpItems: []gaba.FooterHelpItem{
				FooterBack(),
				{ButtonName: icons.LeftRight, HelpText: i18n.Localize(&goi18n.Message{ID: "button_cycle", Other: "Cycle"}, nil)},
				FooterSave(),
			},
			StatusBar:     StatusBar(),
			UseSmallTitle: true,
		},
		items,
	)
	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return output, nil
		}
		return output, err
	}

	for _, item := range result.Items {
		row, ok := item.Item.Metadata.(addonRow)
		if !ok {
			continue
		}
		val := ""
		if v, ok := item.Value().(string); ok {
			val = strings.TrimSpace(v)
		}

		// A blank value or the default path means "no override" — keep config
		// clean and let the destination track the ROM directory dynamically.
		if val == "" || val == row.Default {
			if byCat := config.AddonDirectoryMappings[row.Slug]; byCat != nil {
				delete(byCat, string(row.Category))
				if len(byCat) == 0 {
					delete(config.AddonDirectoryMappings, row.Slug)
				}
			}
			continue
		}

		if config.AddonDirectoryMappings == nil {
			config.AddonDirectoryMappings = make(map[string]map[string]string)
		}
		if config.AddonDirectoryMappings[row.Slug] == nil {
			config.AddonDirectoryMappings[row.Slug] = make(map[string]string)
		}
		config.AddonDirectoryMappings[row.Slug][string(row.Category)] = val
	}

	if err := internal.SaveConfig(config); err != nil {
		gaba.GetLogger().Error("Error saving add-on directories", "error", err)
		return output, err
	}

	output.Config = config
	output.Action = AddonDirectoriesActionSaved
	return output, nil
}
