package esde

import (
	"os"
	"path/filepath"
	"testing"
)

// withTestHome points homeDir at a temp dir and clears settings, discovery
// caches, and relevant env vars so each test starts from a clean slate.
func withTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()

	originalHome := homeDir
	homeDir = func() string { return home }
	t.Cleanup(func() {
		homeDir = originalHome
		Configure(nil)
	})

	for _, key := range []string{"BASE_PATH", "GROUT_ESDE_ROMS_DIR", "ESDE_APPDATA_DIR", "XDG_CONFIG_HOME", "RETRODECK_CFG"} {
		t.Setenv(key, "")
	}

	Configure(nil)
	return home
}

func TestVanillaDefaults(t *testing.T) {
	home := withTestHome(t)

	if got, want := GetRomDirectory(), filepath.Join(home, "ROMs"); got != want {
		t.Errorf("GetRomDirectory() = %q, want %q", got, want)
	}
	if got, want := GetBIOSDirectory(), filepath.Join(home, ".config", "retroarch", "system"); got != want {
		t.Errorf("GetBIOSDirectory() = %q, want %q", got, want)
	}
	if got, want := GetBaseSavePath(), filepath.Join(home, ".config", "retroarch", "saves"); got != want {
		t.Errorf("GetBaseSavePath() = %q, want %q", got, want)
	}
	if got, want := GetAppDataDir(), filepath.Join(home, "ES-DE"); got != want {
		t.Errorf("GetAppDataDir() = %q, want %q", got, want)
	}
	if got, want := GetMediaDir(), filepath.Join(home, "ES-DE", "downloaded_media"); got != want {
		t.Errorf("GetMediaDir() = %q, want %q", got, want)
	}
}

func TestEmuDeckDefaults(t *testing.T) {
	home := withTestHome(t)
	Configure(&Settings{Variant: VariantEmuDeck})

	base := filepath.Join(home, "Emulation")
	if got, want := GetRomDirectory(), filepath.Join(base, "roms"); got != want {
		t.Errorf("GetRomDirectory() = %q, want %q", got, want)
	}
	if got, want := GetBIOSDirectory(), filepath.Join(base, "bios"); got != want {
		t.Errorf("GetBIOSDirectory() = %q, want %q", got, want)
	}
	if got, want := GetBaseSavePath(), filepath.Join(base, "saves"); got != want {
		t.Errorf("GetBaseSavePath() = %q, want %q", got, want)
	}
	if got, want := GetMediaDir(), filepath.Join(base, "tools", "downloaded_media"); got != want {
		t.Errorf("GetMediaDir() = %q, want %q", got, want)
	}
	if got, want := GetAppDataDir(), filepath.Join(home, "ES-DE"); got != want {
		t.Errorf("GetAppDataDir() = %q, want %q", got, want)
	}
}

func TestRetroDeckDefaults(t *testing.T) {
	home := withTestHome(t)
	Configure(&Settings{Variant: VariantRetroDeck})

	base := filepath.Join(home, "retrodeck")
	if got, want := GetRomDirectory(), filepath.Join(base, "roms"); got != want {
		t.Errorf("GetRomDirectory() = %q, want %q", got, want)
	}
	flatpakAppData := filepath.Join(home, ".var", "app", "net.retrodeck.retrodeck", "config", "ES-DE")
	if got := GetAppDataDir(); got != flatpakAppData {
		t.Errorf("GetAppDataDir() = %q, want %q", got, flatpakAppData)
	}
}

func TestRetroDeckAppDataHonorsXDGConfigHome(t *testing.T) {
	withTestHome(t)
	Configure(&Settings{Variant: VariantRetroDeck})
	t.Setenv("XDG_CONFIG_HOME", "/sandbox/config")

	if got, want := GetAppDataDir(), filepath.Join("/sandbox/config", "ES-DE"); got != want {
		t.Errorf("GetAppDataDir() = %q, want %q", got, want)
	}
}

func TestExplicitOverridesWin(t *testing.T) {
	withTestHome(t)
	Configure(&Settings{
		Variant:    VariantEmuDeck,
		BasePath:   "/run/media/sdcard/Emulation",
		RomsDir:    "/custom/roms",
		BiosDir:    "/custom/bios",
		SavesDir:   "/custom/saves",
		AppDataDir: "/custom/ES-DE",
		MediaDir:   "/custom/media",
	})

	if got := GetRomDirectory(); got != "/custom/roms" {
		t.Errorf("GetRomDirectory() = %q, want /custom/roms", got)
	}
	if got := GetBIOSDirectory(); got != "/custom/bios" {
		t.Errorf("GetBIOSDirectory() = %q, want /custom/bios", got)
	}
	if got := GetBaseSavePath(); got != "/custom/saves" {
		t.Errorf("GetBaseSavePath() = %q, want /custom/saves", got)
	}
	if got := GetAppDataDir(); got != "/custom/ES-DE" {
		t.Errorf("GetAppDataDir() = %q, want /custom/ES-DE", got)
	}
	if got := GetMediaDir(); got != "/custom/media" {
		t.Errorf("GetMediaDir() = %q, want /custom/media", got)
	}
}

func TestBasePathOverrideFlowsToChildDirs(t *testing.T) {
	withTestHome(t)
	Configure(&Settings{Variant: VariantEmuDeck, BasePath: "/run/media/sdcard/Emulation"})

	if got, want := GetRomDirectory(), "/run/media/sdcard/Emulation/roms"; got != want {
		t.Errorf("GetRomDirectory() = %q, want %q", got, want)
	}
	if got, want := GetMediaDir(), "/run/media/sdcard/Emulation/tools/downloaded_media"; got != want {
		t.Errorf("GetMediaDir() = %q, want %q", got, want)
	}
}

func TestEnvOverrides(t *testing.T) {
	withTestHome(t)
	t.Setenv("GROUT_ESDE_ROMS_DIR", "/env/roms")
	t.Setenv("ESDE_APPDATA_DIR", "/env/ES-DE")

	if got := GetRomDirectory(); got != "/env/roms" {
		t.Errorf("GetRomDirectory() = %q, want /env/roms", got)
	}
	if got := GetAppDataDir(); got != "/env/ES-DE" {
		t.Errorf("GetAppDataDir() = %q, want /env/ES-DE", got)
	}
	// Config overrides beat environment variables.
	Configure(&Settings{RomsDir: "/config/roms"})
	if got := GetRomDirectory(); got != "/config/roms" {
		t.Errorf("GetRomDirectory() = %q, want /config/roms", got)
	}
}

func TestTildeExpansionInOverrides(t *testing.T) {
	home := withTestHome(t)
	Configure(&Settings{RomsDir: "~/MyROMs"})

	if got, want := GetRomDirectory(), filepath.Join(home, "MyROMs"); got != want {
		t.Errorf("GetRomDirectory() = %q, want %q", got, want)
	}
}

func TestMediaTypeDirs(t *testing.T) {
	home := withTestHome(t)
	romDir := filepath.Join(home, "ROMs", "snes")

	media := filepath.Join(home, "ES-DE", "downloaded_media", "snes")
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"covers", GetArtDirectory(romDir), filepath.Join(media, "covers")},
		{"screenshots", GetScreenshotDirectory(romDir), filepath.Join(media, "screenshots")},
		{"marquees", GetMarqueeDirectory(romDir), filepath.Join(media, "marquees")},
		{"videos", GetVideoDirectory(romDir), filepath.Join(media, "videos")},
		{"manuals", GetManualDirectory(romDir), filepath.Join(media, "manuals")},
		{"backcovers", GetBoxbackDirectory(romDir), filepath.Join(media, "backcovers")},
		{"fanart", GetFanartDirectory(romDir), filepath.Join(media, "fanart")},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s dir = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

func TestGamelistPaths(t *testing.T) {
	home := withTestHome(t)

	romDir := filepath.Join(home, "ROMs", "megadrive")
	if got, want := GetGamelistPath(romDir, "gamelist.xml"), filepath.Join(home, "ES-DE", "gamelists", "megadrive", "gamelist.xml"); got != want {
		t.Errorf("GetGamelistPath() = %q, want %q", got, want)
	}
	if got, want := GetGroutGamelist(), filepath.Join(home, "ES-DE", "gamelists", "ports", "gamelist.xml"); got != want {
		t.Errorf("GetGroutGamelist() = %q, want %q", got, want)
	}
}

func TestSaveDirectoriesPerVariant(t *testing.T) {
	withTestHome(t)

	if got := SaveDirectories()["snes"]; len(got) == 0 || got[0] != "snes" {
		t.Errorf("vanilla SaveDirectories()[snes] = %v, want ES-DE system folders", got)
	}

	Configure(&Settings{Variant: VariantEmuDeck})
	if got := SaveDirectories()["snes"]; len(got) == 0 || got[0] != "retroarch/saves" {
		t.Errorf("emudeck SaveDirectories()[snes] = %v, want [retroarch/saves]", got)
	}
}

func TestDiscoveryFromRetroDeckConfig(t *testing.T) {
	home := withTestHome(t)

	configPath := filepath.Join(home, "retrodeck.json")
	configJSON := `{"paths": {
		"rd_home_path": "/run/media/sdcard/retrodeck",
		"roms_path": "/run/media/sdcard/retrodeck/roms",
		"saves_path": "/run/media/sdcard/retrodeck/saves",
		"bios_path": "/run/media/sdcard/retrodeck/bios",
		"downloaded_media_path": "/run/media/sdcard/retrodeck/ES-DE/downloaded_media"
	}}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RETRODECK_CFG", configPath)
	Configure(&Settings{Variant: VariantRetroDeck})

	if got, want := GetRomDirectory(), "/run/media/sdcard/retrodeck/roms"; got != want {
		t.Errorf("GetRomDirectory() = %q, want %q", got, want)
	}
	if got, want := GetBIOSDirectory(), "/run/media/sdcard/retrodeck/bios"; got != want {
		t.Errorf("GetBIOSDirectory() = %q, want %q", got, want)
	}
	if got, want := GetBaseSavePath(), "/run/media/sdcard/retrodeck/saves"; got != want {
		t.Errorf("GetBaseSavePath() = %q, want %q", got, want)
	}
	if got, want := GetMediaDir(), "/run/media/sdcard/retrodeck/ES-DE/downloaded_media"; got != want {
		t.Errorf("GetMediaDir() = %q, want %q", got, want)
	}

	// Explicit overrides still win over discovered paths.
	Configure(&Settings{Variant: VariantRetroDeck, RomsDir: "/custom/roms"})
	if got := GetRomDirectory(); got != "/custom/roms" {
		t.Errorf("GetRomDirectory() = %q, want /custom/roms", got)
	}
}

func TestDiscoveryFromESSettings(t *testing.T) {
	home := withTestHome(t)

	settingsDir := filepath.Join(home, "ES-DE", "settings")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatal(err)
	}
	esSettings := `<?xml version="1.0"?>
<string name="ROMDirectory" value="/run/media/sdcard/Emulation/roms" />
<string name="MediaDirectory" value="/run/media/sdcard/Emulation/tools/downloaded_media" />
`
	if err := os.WriteFile(filepath.Join(settingsDir, "es_settings.xml"), []byte(esSettings), 0644); err != nil {
		t.Fatal(err)
	}
	Configure(&Settings{Variant: VariantEmuDeck})

	if got, want := GetRomDirectory(), "/run/media/sdcard/Emulation/roms"; got != want {
		t.Errorf("GetRomDirectory() = %q, want %q", got, want)
	}
	if got, want := GetMediaDir(), "/run/media/sdcard/Emulation/tools/downloaded_media"; got != want {
		t.Errorf("GetMediaDir() = %q, want %q", got, want)
	}
}

func TestDetectVariant(t *testing.T) {
	home := withTestHome(t)

	if got := DetectVariant(); got != VariantVanilla {
		t.Errorf("DetectVariant() = %q, want vanilla on empty home", got)
	}

	if err := os.MkdirAll(filepath.Join(home, "Emulation"), 0755); err != nil {
		t.Fatal(err)
	}
	if got := DetectVariant(); got != VariantEmuDeck {
		t.Errorf("DetectVariant() = %q, want emudeck when ~/Emulation exists", got)
	}

	if err := os.MkdirAll(filepath.Join(home, "retrodeck"), 0755); err != nil {
		t.Fatal(err)
	}
	if got := DetectVariant(); got != VariantRetroDeck {
		t.Errorf("DetectVariant() = %q, want retrodeck when ~/retrodeck exists", got)
	}
}
