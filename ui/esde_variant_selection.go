package ui

import (
	"grout/cfw/esde"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	icons "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

type ESDEVariantSelectionScreen struct{}

func NewESDEVariantSelectionScreen() *ESDEVariantSelectionScreen {
	return &ESDEVariantSelectionScreen{}
}

func (s *ESDEVariantSelectionScreen) Draw() (esde.Variant, error) {
	options := []gaba.SelectionOption{
		{
			DisplayName: "ES-DE",
			Description: i18n.Localize(&goi18n.Message{ID: "esde_variant_vanilla_desc", Other: "Standalone ES-DE (ROMs in ~/ROMs)"}, nil),
			Value:       esde.VariantVanilla,
		},
		{
			DisplayName: "EmuDeck",
			Description: i18n.Localize(&goi18n.Message{ID: "esde_variant_emudeck_desc", Other: "EmuDeck install (~/Emulation)"}, nil),
			Value:       esde.VariantEmuDeck,
		},
		{
			DisplayName: "RetroDECK",
			Description: i18n.Localize(&goi18n.Message{ID: "esde_variant_retrodeck_desc", Other: "RetroDECK flatpak (~/retrodeck)"}, nil),
			Value:       esde.VariantRetroDeck,
		},
	}

	initialSelection := 0
	detected := esde.DetectVariant()
	for i, option := range options {
		if option.Value == detected {
			initialSelection = i
		}
	}

	result, err := gaba.SelectionMessage(
		i18n.Localize(&goi18n.Message{ID: "esde_variant_title", Other: "Which ES-DE setup are you using?"}, nil),
		options,
		[]gaba.FooterHelpItem{
			{ButtonName: icons.LeftRight, HelpText: i18n.Localize(&goi18n.Message{ID: "button_select", Other: "Select"}, nil)},
			{ButtonName: "A", HelpText: i18n.Localize(&goi18n.Message{ID: "button_confirm", Other: "Confirm"}, nil)},
		},
		gaba.SelectionMessageSettings{
			DisableBackButton: true,
			InitialSelection:  initialSelection,
		},
	)

	if err != nil {
		return esde.VariantVanilla, err
	}

	return result.SelectedValue.(esde.Variant), nil
}
