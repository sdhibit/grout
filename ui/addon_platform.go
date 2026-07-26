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

// addonGameContentCategories are the supplemental add-on categories offered as
// configurable subfolder rows on a platform's add-on screen, in display order.
// Only content layered on top of the base game (updates, DLC, patches) is placed
// via these mappings; standalone alternate builds (hacks, prototypes, …) are
// downloaded as selectable versions into the ROM directory, not as add-ons.
var addonGameContentCategories = []romm.RomFileCategory{
	romm.RomFileUpdate,
	romm.RomFileDLC,
	romm.RomFilePatch,
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

// Draw configures where one platform's add-ons are downloaded, mirroring the Rom
// Directory Mapping screen's cycle interaction. The Base Folder row cycles between
// the platform's ROM directory (default) and a Custom path. Each add-on category
// row cycles Skip / default-subfolder / Custom: Skip places the files directly in
// the base folder, the default is a subfolder named after the category, and Custom
// is any other subfolder relative to the base.
func (s *AddonPlatformScreen) Draw(input AddonPlatformInput) (AddonPlatformOutput, error) {
	config := input.Config
	output := AddonPlatformOutput{Config: config}

	romDir := config.GetPlatformRomDirectory(input.Platform)
	mapping := config.AddonDirectoryMappings[input.Platform.FSSlug]

	customLabel := i18n.Localize(&goi18n.Message{ID: "platform_mapping_custom", Other: "Custom..."}, nil)

	// Base Folder: a default (the platform's ROM directory) / Custom cycle. There
	// is no Skip here — the platform's add-ons must resolve to some root folder.
	baseSelected := 0
	baseCustom := ""
	if mapping.BaseDir != "" && mapping.BaseDir != romDir {
		baseSelected = 1
		baseCustom = mapping.BaseDir
	}
	baseCustomDisplay := customLabel
	// Prefill the keyboard with the path currently in effect so editing the base
	// folder starts from the real path instead of a blank field. When a custom
	// base is set, that's the starting point; otherwise it's the ROM directory.
	// DisplayName/Value stay as-is, so an unedited confirm still resolves to the
	// default base (non-destructive).
	baseKeyboardPrompt := romDir
	if baseCustom != "" {
		baseCustomDisplay = baseCustom
		baseKeyboardPrompt = baseCustom
	}
	items := []gaba.ItemWithOptions{
		{
			Item: gaba.MenuItem{
				Text:     i18n.Localize(&goi18n.Message{ID: "addon_base_folder", Other: "Base Folder"}, nil),
				Metadata: addonBaseRowKey,
			},
			Options: []gaba.Option{
				{DisplayName: romDir, Value: romDir},
				{
					Type:           gaba.OptionTypeKeyboard,
					DisplayName:    baseCustomDisplay,
					KeyboardPrompt: baseKeyboardPrompt,
					Value:          baseCustom,
				},
			},
			SelectedOption: baseSelected,
		},
	}

	// Each add-on category: a Skip / default-subfolder / Custom cycle, mirroring
	// the Rom Directory Mapping screen. Skip places the files in the base folder
	// (stored as an empty subfolder), the default is a subfolder named after the
	// category, and Custom takes any other subfolder relative to the base.
	skipLabel := i18n.Localize(&goi18n.Message{ID: "common_skip", Other: "Skip"}, nil)
	for _, cat := range addonGameContentCategories {
		defaultSub := string(cat)
		defaultDisplay := i18n.Localize(&goi18n.Message{ID: "platform_mapping_path_prefix", Other: "/{{.Name}}"}, map[string]interface{}{"Name": defaultSub})

		selected := 1 // default subfolder
		custom := ""
		if v, present := mapping.Categories[defaultSub]; present {
			switch v {
			case "":
				selected = 0 // Skip -> base folder
			case defaultSub:
				selected = 1
			default:
				selected = 2 // custom subfolder
				custom = v
			}
		}
		customDisplay := customLabel
		if custom != "" {
			customDisplay = custom
		}

		items = append(items, gaba.ItemWithOptions{
			Item: gaba.MenuItem{
				Text:     addonCategoryLabel(cat),
				Metadata: defaultSub,
			},
			Options: []gaba.Option{
				{DisplayName: skipLabel, Value: ""},
				{DisplayName: defaultDisplay, Value: defaultSub},
				{
					Type:           gaba.OptionTypeKeyboard,
					DisplayName:    customDisplay,
					KeyboardPrompt: custom,
					Value:          custom,
				},
			},
			SelectedOption: selected,
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
			StatusBar:        StatusBar(),
			UseSmallTitle:    true,
			ListPickerButton: icons.VirtualButtonA,
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
