package esde

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
)

// parseESSettings reads ES-DE's es_settings.xml, a sequence of
// <string name="..." value="..." /> elements with no root element.
func parseESSettings(data []byte) map[string]string {
	values := map[string]string{}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err != nil {
			return values
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		var name, value string
		for _, attr := range start.Attr {
			switch attr.Name.Local {
			case "name":
				name = attr.Value
			case "value":
				value = attr.Value
			}
		}
		if name != "" {
			values[name] = value
		}
	}
}

var esSettingsCache cached[map[string]string]

// esSettingsValue returns a setting from ES-DE's own configuration (e.g.
// ROMDirectory, MediaDirectory), or "" when unset or the file is absent.
// EmuDeck writes both of these, so they resolve relocated installs.
func esSettingsValue(name string) string {
	values := esSettingsCache.get(func() map[string]string {
		appData := GetAppDataDir()
		for _, candidate := range []string{
			filepath.Join(appData, "settings", "es_settings.xml"),
			filepath.Join(appData, "es_settings.xml"),
		} {
			if data, err := os.ReadFile(candidate); err == nil {
				return parseESSettings(data)
			}
		}
		return map[string]string{}
	})
	return expandHome(values[name])
}
