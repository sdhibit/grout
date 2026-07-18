package esde

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed input_mappings/*.json
var embeddedInputMappings embed.FS

// GetInputMappingBytes returns the controller mapping for ES-DE devices.
//
// ES-DE runs on desktop-class hardware (Steam Deck, Steam Machine) where the
// controller presents as a standard SDL game controller. Because Steam Input
// virtualizes every device — the Deck's built-in controls, the Steam
// Controller, a Steam Machine's pad — to the same Xbox-style layout, one
// mapping keyed on the SDL GameController abstraction covers them all.
//
// gabagool's default mapping is tuned for handhelds and (a) uses a
// Nintendo-style A/B swap that feels reversed on the Xbox layout and (b) binds
// Menu to the Steam (GUIDE) button, which Steam Input intercepts so it never
// reaches Grout. This mapping instead uses direct face buttons and makes Menu
// reachable via the L2 trigger and the left-stick click, so BIOS download works
// out of the box.
func GetInputMappingBytes() ([]byte, error) {
	const filename = "input_mappings/steam-input.json"

	overridePath := filepath.Join("overrides", "cfw", "esde", filename)
	if data, err := os.ReadFile(overridePath); err == nil {
		return data, nil
	}

	data, err := embeddedInputMappings.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded input mapping %s: %w", filename, err)
	}
	return data, nil
}
