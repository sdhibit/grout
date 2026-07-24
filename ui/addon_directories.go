package ui

import (
	"errors"

	"grout/internal"
	"grout/romm"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

// addonCapableSlugs are the platforms whose games commonly ship updates/DLC and
// whose emulators may want them in dedicated folders. Only these are offered in
// the add-on folder settings, keeping the list focused (Switch is the primary
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

type AddonDirectoriesInput struct {
	Config    *internal.Config
	Platforms []romm.Platform
}

type AddonDirectoriesOutput struct {
	Action           AddonDirectoriesAction
	SelectedPlatform romm.Platform
	Config           *internal.Config
}

type AddonDirectoriesAction int

const (
	AddonDirectoriesActionBack AddonDirectoriesAction = iota
	AddonDirectoriesActionSelectPlatform
)

type AddonDirectoriesScreen struct{}

func NewAddonDirectoriesScreen() *AddonDirectoriesScreen {
	return &AddonDirectoriesScreen{}
}

// Draw lists the add-on-capable platforms in the user's library. Selecting one
// opens its own screen (AddonPlatformScreen) for configuring where that
// platform's add-ons are placed — a base folder plus per-category subfolders —
// mirroring how Directory Mappings gives each system its own area.
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

	items := make([]gaba.ItemWithOptions, 0, len(platforms))
	for _, p := range platforms {
		items = append(items, gaba.ItemWithOptions{
			Item:    gaba.MenuItem{Text: p.Name, Metadata: p},
			Options: []gaba.Option{{Type: gaba.OptionTypeClickable}},
		})
	}

	result, err := gaba.OptionsList(
		i18n.Localize(&goi18n.Message{ID: "settings_addon_dirs", Other: "Add-on Folders"}, nil),
		gaba.OptionListSettings{
			FooterHelpItems: []gaba.FooterHelpItem{FooterBack(), FooterSelect()},
			StatusBar:       StatusBar(),
			UseSmallTitle:   true,
		},
		items,
	)
	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return output, nil
		}
		return output, err
	}

	if result.Action == gaba.ListActionSelected {
		if p, ok := items[result.Selected].Item.Metadata.(romm.Platform); ok {
			output.Action = AddonDirectoriesActionSelectPlatform
			output.SelectedPlatform = p
		}
	}

	return output, nil
}
