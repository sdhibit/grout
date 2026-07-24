package ui

import (
	"errors"
	"strings"

	"grout/internal"
	"grout/romm"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	icons "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

// addonGameContentCategories are the add-on categories offered as configurable
// subfolder rows on a platform's add-on screen, in display order.
var addonGameContentCategories = []romm.RomFileCategory{
	romm.RomFileUpdate,
	romm.RomFileDLC,
	romm.RomFilePatch,
	romm.RomFileHack,
	romm.RomFileMod,
	romm.RomFileTranslation,
	romm.RomFileDemo,
	romm.RomFilePrototype,
}

// addonBaseRowKey marks the Base Folder row's metadata (vs a category value).
const addonBaseRowKey = "\x00base"

type AddonPlatformInput struct {
	Config   *internal.Config
	Platform romm.Platform
}

type AddonPlatformOutput struct {
	Config *internal.Config
}

type AddonPlatformScreen struct{}

func NewAddonPlatformScreen() *AddonPlatformScreen {
	return &AddonPlatformScreen{}
}

// Draw configures where one platform's add-ons are downloaded: a Base Folder
// (default: the platform's ROM directory) plus, per add-on category, a subfolder
// relative to that base. Clearing a category's subfolder places its files
// directly in the base folder. Mirrors the platform directory-mapping screen.
func (s *AddonPlatformScreen) Draw(input AddonPlatformInput) (AddonPlatformOutput, error) {
	config := input.Config
	output := AddonPlatformOutput{Config: config}

	romDir := config.GetPlatformRomDirectory(input.Platform)
	mapping := config.AddonDirectoryMappings[input.Platform.FSSlug]

	base := romDir
	if mapping.BaseDir != "" {
		base = mapping.BaseDir
	}

	items := []gaba.ItemWithOptions{
		{
			Item: gaba.MenuItem{
				Text:     i18n.Localize(&goi18n.Message{ID: "addon_base_folder", Other: "Base Folder"}, nil),
				Metadata: addonBaseRowKey,
			},
			Options: []gaba.Option{{
				Type:           gaba.OptionTypeKeyboard,
				DisplayName:    base,
				KeyboardPrompt: base,
				Value:          base,
			}},
		},
	}

	baseFolderLabel := i18n.Localize(&goi18n.Message{ID: "addon_subfolder_base", Other: "(base folder)"}, nil)
	for _, cat := range addonGameContentCategories {
		sub := string(cat) // default subfolder = the category name
		if v, present := mapping.Categories[string(cat)]; present {
			sub = v
		}
		display := sub
		if display == "" {
			display = baseFolderLabel
		}
		items = append(items, gaba.ItemWithOptions{
			Item: gaba.MenuItem{
				Text:     addonCategoryLabel(cat),
				Metadata: string(cat),
			},
			Options: []gaba.Option{{
				Type:           gaba.OptionTypeKeyboard,
				DisplayName:    display,
				KeyboardPrompt: sub,
				Value:          sub,
			}},
		})
	}

	title := i18n.Localize(&goi18n.Message{ID: "addon_platform_title", Other: "{{.Name}} Add-ons"}, map[string]interface{}{"Name": input.Platform.Name})
	result, err := gaba.OptionsList(
		title,
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

	// Rebuild this platform's mapping from the rows. Values equal to their
	// default (base = ROM dir, category subfolder = category name) are dropped so
	// the config stays minimal and tracks the ROM directory.
	newMapping := internal.AddonPlatformMapping{Categories: make(map[string]string)}
	for _, item := range result.Items {
		key, _ := item.Item.Metadata.(string)
		val := ""
		if v, ok := item.Value().(string); ok {
			val = strings.TrimSpace(v)
		}
		if key == addonBaseRowKey {
			if val != "" && val != romDir {
				newMapping.BaseDir = val
			}
			continue
		}
		if val != key { // key is the category value, which is also its default subfolder
			newMapping.Categories[key] = val
		}
	}

	if newMapping.BaseDir == "" && len(newMapping.Categories) == 0 {
		delete(config.AddonDirectoryMappings, input.Platform.FSSlug)
	} else {
		if config.AddonDirectoryMappings == nil {
			config.AddonDirectoryMappings = make(map[string]internal.AddonPlatformMapping)
		}
		config.AddonDirectoryMappings[input.Platform.FSSlug] = newMapping
	}

	if err := internal.SaveConfig(config); err != nil {
		gaba.GetLogger().Error("Error saving add-on folders", "error", err)
		return output, err
	}

	output.Config = config
	return output, nil
}
