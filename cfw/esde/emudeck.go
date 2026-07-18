package esde

import (
	"os"
	"path/filepath"
	"strings"
)

// parseEmuDeckEmulationPath extracts emulationPath from EmuDeck's settings.sh,
// a flat KEY="VALUE" shell file. This is how custom install roots (e.g. SD
// cards) are located without user configuration.
func parseEmuDeckEmulationPath(script string) string {
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimSpace(line)
		value, found := strings.CutPrefix(line, "emulationPath=")
		if !found {
			continue
		}
		value = strings.Trim(value, `"'`)
		value = strings.ReplaceAll(value, "${HOME}", homeDir())
		value = strings.ReplaceAll(value, "$HOME", homeDir())
		return expandHome(value)
	}
	return ""
}

var emuDeckPathCache cached[string]

func emuDeckEmulationPath() string {
	return emuDeckPathCache.get(func() string {
		data, err := os.ReadFile(filepath.Join(homeDir(), "emudeck", "settings.sh"))
		if err != nil {
			return ""
		}
		return parseEmuDeckEmulationPath(string(data))
	})
}
