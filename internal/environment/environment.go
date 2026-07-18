package environment

import (
	"os"
	"strings"
)

func IsDevelopment() bool {
	return os.Getenv("ENVIRONMENT") == "DEV"
}

func IsMiyoo() bool {
	return os.Getenv("IS_MIYOO") == "1"
}

// IsESDE reports whether Grout is running under the ES-DE CFW (vanilla ES-DE,
// EmuDeck, or RetroDECK). Used for input-hint labels; the CFW env var is set by
// the launcher.
func IsESDE() bool {
	return strings.EqualFold(os.Getenv("CFW"), "ESDE")
}
