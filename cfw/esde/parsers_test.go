package esde

import (
	"path/filepath"
	"testing"
)

func TestParseRetroDeckConfigWrapped(t *testing.T) {
	data := []byte(`{"version": "0.9.0", "paths": {
		"rd_home_path": "/home/deck/retrodeck",
		"roms_path": "/home/deck/retrodeck/roms",
		"saves_path": "/home/deck/retrodeck/saves",
		"bios_path": "/home/deck/retrodeck/bios",
		"downloaded_media_path": "/home/deck/retrodeck/ES-DE/downloaded_media"
	}}`)

	paths := parseRetroDeckConfig(data)
	if paths.RDHomePath != "/home/deck/retrodeck" {
		t.Errorf("RDHomePath = %q", paths.RDHomePath)
	}
	if paths.RomsPath != "/home/deck/retrodeck/roms" {
		t.Errorf("RomsPath = %q", paths.RomsPath)
	}
	if paths.DownloadedMediaPath != "/home/deck/retrodeck/ES-DE/downloaded_media" {
		t.Errorf("DownloadedMediaPath = %q", paths.DownloadedMediaPath)
	}
}

func TestParseRetroDeckConfigFlat(t *testing.T) {
	data := []byte(`{"rd_home_path": "/home/deck/retrodeck", "roms_path": "/home/deck/retrodeck/roms"}`)

	paths := parseRetroDeckConfig(data)
	if paths.RomsPath != "/home/deck/retrodeck/roms" {
		t.Errorf("RomsPath = %q", paths.RomsPath)
	}
}

func TestParseRetroDeckConfigInvalid(t *testing.T) {
	if paths := parseRetroDeckConfig([]byte("not json")); paths != (retroDeckPathsData{}) {
		t.Errorf("expected zero value for invalid JSON, got %+v", paths)
	}
}

func TestParseEmuDeckEmulationPath(t *testing.T) {
	home := withTestHome(t)

	cases := []struct {
		name   string
		script string
		want   string
	}{
		{"plain", "emulationPath=/run/media/sdcard/Emulation\n", "/run/media/sdcard/Emulation"},
		{"double_quoted", `emulationPath="/run/media/sdcard/Emulation"` + "\n", "/run/media/sdcard/Emulation"},
		{"single_quoted", "emulationPath='/run/media/sdcard/Emulation'\n", "/run/media/sdcard/Emulation"},
		{"home_var", `emulationPath="$HOME/Emulation"` + "\n", filepath.Join(home, "Emulation")},
		{"home_braces", `emulationPath="${HOME}/Emulation"` + "\n", filepath.Join(home, "Emulation")},
		{"tilde", "emulationPath=~/Emulation\n", filepath.Join(home, "Emulation")},
		{"among_other_lines", "#!/bin/bash\nfoo=bar\nemulationPath=/x/Emulation\nother=1\n", "/x/Emulation"},
		{"missing", "foo=bar\n", ""},
	}
	for _, tc := range cases {
		if got := parseEmuDeckEmulationPath(tc.script); got != tc.want {
			t.Errorf("%s: parseEmuDeckEmulationPath() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestParseESSettings(t *testing.T) {
	data := []byte(`<?xml version="1.0"?>
<string name="ROMDirectory" value="/home/deck/Emulation/roms" />
<string name="MediaDirectory" value="" />
<bool name="ShowHiddenFiles" value="false" />
<int name="MaxVRAM" value="512" />
`)

	values := parseESSettings(data)
	if got := values["ROMDirectory"]; got != "/home/deck/Emulation/roms" {
		t.Errorf("ROMDirectory = %q", got)
	}
	if got := values["MediaDirectory"]; got != "" {
		t.Errorf("MediaDirectory = %q, want empty", got)
	}
	if got := values["ShowHiddenFiles"]; got != "false" {
		t.Errorf("ShowHiddenFiles = %q", got)
	}
}

func TestParseESSettingsMalformed(t *testing.T) {
	// Truncated XML must not panic and returns what was parsed so far.
	data := []byte(`<string name="ROMDirectory" value="/roms" /><string name=`)
	values := parseESSettings(data)
	if got := values["ROMDirectory"]; got != "/roms" {
		t.Errorf("ROMDirectory = %q", got)
	}
}
