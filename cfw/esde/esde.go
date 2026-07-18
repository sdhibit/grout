// Package esde implements support for EmulationStation Desktop Edition (ES-DE)
// setups: vanilla ES-DE on desktop Linux, EmuDeck, and RetroDECK.
//
// Unlike handheld CFWs, ES-DE installs vary per user: EmuDeck and RetroDECK
// both allow relocating their install root, and each directory can be moved
// individually. Every directory therefore resolves through a precedence chain:
// explicit config override > environment variable > variant discovery
// (retrodeck.json, EmuDeck settings.sh, es_settings.xml) > variant default.
//
// ES-DE also differs from Batocera-style EmulationStation in layout: gamelists
// live under <appdata>/gamelists/<system>/ rather than next to the ROMs, and
// media lives under <media>/<system>/{covers,marquees,...} matched purely by
// ROM basename, so no filename suffixes are used.
package esde

import (
	"embed"
	"grout/internal/jsonutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed data/*.json
var embeddedFiles embed.FS

var (
	Platforms              = jsonutil.MustLoadJSONMap[string, []string](embeddedFiles, "data/platforms.json")
	emuDeckSaveDirectories = jsonutil.MustLoadJSONMap[string, []string](embeddedFiles, "data/save_directories_emudeck.json")
)

type Variant string

const (
	VariantVanilla   Variant = "vanilla"
	VariantEmuDeck   Variant = "emudeck"
	VariantRetroDeck Variant = "retrodeck"
)

// Settings holds the user-configurable ES-DE options persisted in config.json.
// Empty fields fall back to environment variables, variant discovery, and
// finally the variant's static defaults.
type Settings struct {
	Variant    Variant `json:"variant,omitempty"`
	BasePath   string  `json:"base_path,omitempty"`
	RomsDir    string  `json:"roms_dir,omitempty"`
	BiosDir    string  `json:"bios_dir,omitempty"`
	SavesDir   string  `json:"saves_dir,omitempty"`
	AppDataDir string  `json:"appdata_dir,omitempty"`
	MediaDir   string  `json:"media_dir,omitempty"`
}

var settings Settings

// Configure installs the ES-DE settings loaded from config.json. Passing nil
// resets to defaults (vanilla variant, no overrides). It is called from
// LoadConfig/SaveConfig so path resolution always reflects the current config.
func Configure(s *Settings) {
	if s == nil {
		settings = Settings{}
	} else {
		settings = *s
	}
	resetDiscoveryCaches()
}

func CurrentVariant() Variant {
	switch settings.Variant {
	case VariantEmuDeck, VariantRetroDeck:
		return settings.Variant
	default:
		return VariantVanilla
	}
}

var homeDir = sync.OnceValue(func() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return os.Getenv("HOME")
})

func expandHome(path string) string {
	if path == "~" {
		return homeDir()
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir(), path[2:])
	}
	return path
}

// explicitBase returns a base path the user set themselves (config override or
// BASE_PATH env), or "" when only variant defaults apply.
func explicitBase() string {
	if p := expandHome(settings.BasePath); p != "" {
		return p
	}
	return os.Getenv("BASE_PATH")
}

func GetBasePath() string {
	if base := explicitBase(); base != "" {
		return base
	}
	if p := discoveredBasePath(); p != "" {
		return p
	}
	switch CurrentVariant() {
	case VariantEmuDeck:
		return filepath.Join(homeDir(), "Emulation")
	case VariantRetroDeck:
		return filepath.Join(homeDir(), "retrodeck")
	default:
		return homeDir()
	}
}

func GetRomDirectory() string {
	if p := expandHome(settings.RomsDir); p != "" {
		return p
	}
	if p := os.Getenv("GROUT_ESDE_ROMS_DIR"); p != "" {
		return p
	}
	if p := discoveredRomsPath(); p != "" {
		return p
	}
	switch CurrentVariant() {
	case VariantEmuDeck, VariantRetroDeck:
		return filepath.Join(GetBasePath(), "roms")
	default:
		// ES-DE's default ROM directory; follows the base path when the user
		// pointed grout at a custom root.
		if base := explicitBase(); base != "" {
			return filepath.Join(base, "ROMs")
		}
		return filepath.Join(homeDir(), "ROMs")
	}
}

func GetBIOSDirectory() string {
	if p := expandHome(settings.BiosDir); p != "" {
		return p
	}
	if p := discoveredBiosPath(); p != "" {
		return p
	}
	switch CurrentVariant() {
	case VariantEmuDeck, VariantRetroDeck:
		return filepath.Join(GetBasePath(), "bios")
	default:
		if base := explicitBase(); base != "" {
			return filepath.Join(base, "bios")
		}
		// Vanilla ES-DE has no BIOS convention of its own; RetroArch's system
		// directory is the most common target. Overridable in ES-DE Settings.
		return filepath.Join(homeDir(), ".config", "retroarch", "system")
	}
}

func GetBaseSavePath() string {
	if p := expandHome(settings.SavesDir); p != "" {
		return p
	}
	if p := discoveredSavesPath(); p != "" {
		return p
	}
	switch CurrentVariant() {
	case VariantEmuDeck, VariantRetroDeck:
		return filepath.Join(GetBasePath(), "saves")
	default:
		if base := explicitBase(); base != "" {
			return filepath.Join(base, "saves")
		}
		// Vanilla ES-DE has no saves convention; RetroArch's default save
		// directory is the most common target. Overridable in ES-DE Settings.
		return filepath.Join(homeDir(), ".config", "retroarch", "saves")
	}
}

// GetAppDataDir returns ES-DE's application data directory, which holds
// gamelists/ and (by default) downloaded_media/.
func GetAppDataDir() string {
	if p := expandHome(settings.AppDataDir); p != "" {
		return p
	}
	// ES-DE itself honors ESDE_APPDATA_DIR, so respect the same override.
	if p := os.Getenv("ESDE_APPDATA_DIR"); p != "" {
		return p
	}
	if p := discoveredAppDataPath(); p != "" {
		return p
	}
	if CurrentVariant() == VariantRetroDeck {
		// RetroDECK ships ES-DE inside its flatpak, so the appdata dir lives in
		// the sandbox's XDG config dir rather than ~/ES-DE.
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "ES-DE")
		}
		return filepath.Join(homeDir(), ".var", "app", "net.retrodeck.retrodeck", "config", "ES-DE")
	}
	return filepath.Join(homeDir(), "ES-DE")
}

// GetMediaDir returns the root of ES-DE's downloaded media tree
// (<media>/<system>/covers etc.).
func GetMediaDir() string {
	if p := expandHome(settings.MediaDir); p != "" {
		return p
	}
	if p := discoveredMediaPath(); p != "" {
		return p
	}
	if CurrentVariant() == VariantEmuDeck {
		return filepath.Join(GetBasePath(), "tools", "downloaded_media")
	}
	return filepath.Join(GetAppDataDir(), "downloaded_media")
}

// systemName extracts the ES-DE system folder name from a ROM directory path.
func systemName(romDir string) string {
	return filepath.Base(filepath.Clean(romDir))
}

func mediaTypeDir(romDir, mediaType string) string {
	return filepath.Join(GetMediaDir(), systemName(romDir), mediaType)
}

func GetArtDirectory(romDir string) string {
	return mediaTypeDir(romDir, "covers")
}

func GetScreenshotDirectory(romDir string) string {
	return mediaTypeDir(romDir, "screenshots")
}

func GetMarqueeDirectory(romDir string) string {
	return mediaTypeDir(romDir, "marquees")
}

func GetVideoDirectory(romDir string) string {
	return mediaTypeDir(romDir, "videos")
}

func GetManualDirectory(romDir string) string {
	return mediaTypeDir(romDir, "manuals")
}

func GetBoxbackDirectory(romDir string) string {
	return mediaTypeDir(romDir, "backcovers")
}

func GetFanartDirectory(romDir string) string {
	return mediaTypeDir(romDir, "fanart")
}

// GetGamelistPath returns the gamelist file for a system. ES-DE keeps
// gamelists under its appdata dir, keyed by system folder name, never next to
// the ROMs.
func GetGamelistPath(romDir string, filename string) string {
	return filepath.Join(GetAppDataDir(), "gamelists", systemName(romDir), filename)
}

func GetGroutGamelist() string {
	return filepath.Join(GetAppDataDir(), "gamelists", "ports", "gamelist.xml")
}

// SaveDirectories returns the per-platform save folder map. EmuDeck organizes
// saves per emulator; vanilla ES-DE and RetroDECK use per-system folders
// matching the ROM folder names.
func SaveDirectories() map[string][]string {
	if CurrentVariant() == VariantEmuDeck {
		return emuDeckSaveDirectories
	}
	return Platforms
}
