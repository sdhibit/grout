package ui

import (
	"errors"
	"grout/cfw/esde"
	"grout/internal"
	"strings"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	icons "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

type ESDESettingsInput struct {
	Config *internal.Config
}

type ESDESettingsOutput struct {
	Action ESDESettingsAction
	Config *internal.Config
}

type ESDESettingsAction int

const (
	ESDESettingsActionSaved ESDESettingsAction = iota
	ESDESettingsActionBack
)

type ESDESettingsScreen struct{}

func NewESDESettingsScreen() *ESDESettingsScreen {
	return &ESDESettingsScreen{}
}

// Draw shows the ES-DE variant selector plus per-directory overrides. Rows
// display the currently effective path; an empty value clears the override so
// the variant default (or discovered path) applies again.
func (s *ESDESettingsScreen) Draw(input ESDESettingsInput) (ESDESettingsOutput, error) {
	config := input.Config
	output := ESDESettingsOutput{Action: ESDESettingsActionBack, Config: config}

	current := esde.Settings{}
	if config.ESDE != nil {
		current = *config.ESDE
	}

	pathRow := func(label *goi18n.Message, override, effective string) gaba.ItemWithOptions {
		display := override
		prompt := override
		if display == "" {
			display = effective
			prompt = effective
		}
		return gaba.ItemWithOptions{
			Item: gaba.MenuItem{Text: i18n.Localize(label, nil)},
			Options: []gaba.Option{
				{
					Type:           gaba.OptionTypeKeyboard,
					DisplayName:    display,
					KeyboardPrompt: prompt,
					Value:          override,
				},
			},
		}
	}

	items := []gaba.ItemWithOptions{
		{
			Item: gaba.MenuItem{Text: i18n.Localize(&goi18n.Message{ID: "esde_settings_variant", Other: "Variant"}, nil)},
			Options: []gaba.Option{
				{DisplayName: "ES-DE", Value: esde.VariantVanilla},
				{DisplayName: "EmuDeck", Value: esde.VariantEmuDeck},
				{DisplayName: "RetroDECK", Value: esde.VariantRetroDeck},
			},
			SelectedOption: variantToIndex(current.Variant),
		},
		pathRow(&goi18n.Message{ID: "esde_settings_base_path", Other: "Base Path"}, current.BasePath, esde.GetBasePath()),
		pathRow(&goi18n.Message{ID: "esde_settings_roms_dir", Other: "ROMs Directory"}, current.RomsDir, esde.GetRomDirectory()),
		pathRow(&goi18n.Message{ID: "esde_settings_bios_dir", Other: "BIOS Directory"}, current.BiosDir, esde.GetBIOSDirectory()),
		pathRow(&goi18n.Message{ID: "esde_settings_saves_dir", Other: "Saves Directory"}, current.SavesDir, esde.GetBaseSavePath()),
		pathRow(&goi18n.Message{ID: "esde_settings_appdata_dir", Other: "ES-DE Data Directory"}, current.AppDataDir, esde.GetAppDataDir()),
		pathRow(&goi18n.Message{ID: "esde_settings_media_dir", Other: "Media Directory"}, current.MediaDir, esde.GetMediaDir()),
	}

	result, err := gaba.OptionsList(
		i18n.Localize(&goi18n.Message{ID: "settings_esde", Other: "ES-DE Settings"}, nil),
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
		gaba.GetLogger().Error("ES-DE settings error", "error", err)
		return output, err
	}

	settings := result.Items

	updated := current
	if variant, ok := settings[0].Options[settings[0].SelectedOption].Value.(esde.Variant); ok {
		updated.Variant = variant
	}
	pathValue := func(item gaba.ItemWithOptions) string {
		if value, ok := item.Value().(string); ok {
			return strings.TrimSpace(value)
		}
		return ""
	}
	updated.BasePath = pathValue(settings[1])
	updated.RomsDir = pathValue(settings[2])
	updated.BiosDir = pathValue(settings[3])
	updated.SavesDir = pathValue(settings[4])
	updated.AppDataDir = pathValue(settings[5])
	updated.MediaDir = pathValue(settings[6])

	config.ESDE = &updated
	if err := internal.SaveConfig(config); err != nil {
		gaba.GetLogger().Error("Error saving ES-DE settings", "error", err)
		return output, err
	}

	output.Config = config
	output.Action = ESDESettingsActionSaved
	return output, nil
}

func variantToIndex(variant esde.Variant) int {
	switch variant {
	case esde.VariantEmuDeck:
		return 1
	case esde.VariantRetroDeck:
		return 2
	default:
		return 0
	}
}
