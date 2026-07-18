package esde

import (
	"os"
	"path/filepath"
)

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// DetectVariant guesses which ES-DE flavor this machine runs, used only to
// pre-select the first-run variant screen; the user always confirms.
func DetectVariant() Variant {
	if FindRetroDeckConfig() != "" || dirExists(filepath.Join(homeDir(), "retrodeck")) {
		return VariantRetroDeck
	}
	if _, err := os.Stat(filepath.Join(homeDir(), "emudeck", "settings.sh")); err == nil {
		return VariantEmuDeck
	}
	if dirExists(filepath.Join(homeDir(), "Emulation")) {
		return VariantEmuDeck
	}
	return VariantVanilla
}
