package internal

import (
	"path/filepath"
	"testing"

	"grout/romm"
)

func TestAddonDestination(t *testing.T) {
	sw := romm.Platform{FSSlug: "switch"}
	romDir := "/roms/switch"

	tests := []struct {
		name    string
		mapping *AddonPlatformMapping // nil = no entry for the platform
		cat     romm.RomFileCategory
		want    string
	}{
		{
			name: "no mapping uses a category subfolder under the rom dir",
			cat:  romm.RomFileUpdate,
			want: filepath.Join(romDir, "update"),
		},
		{
			name:    "base override keeps the default category subfolder",
			mapping: &AddonPlatformMapping{BaseDir: "/mnt/switch"},
			cat:     romm.RomFileDLC,
			want:    filepath.Join("/mnt/switch", "dlc"),
		},
		{
			name:    "custom subfolder under the default base",
			mapping: &AddonPlatformMapping{Categories: map[string]string{"update": "load"}},
			cat:     romm.RomFileUpdate,
			want:    filepath.Join(romDir, "load"),
		},
		{
			name:    "skipped category lands directly in the base folder",
			mapping: &AddonPlatformMapping{Categories: map[string]string{"dlc": ""}},
			cat:     romm.RomFileDLC,
			want:    romDir,
		},
		{
			name:    "base override plus custom subfolder",
			mapping: &AddonPlatformMapping{BaseDir: "/mnt/sw", Categories: map[string]string{"update": "updates"}},
			cat:     romm.RomFileUpdate,
			want:    filepath.Join("/mnt/sw", "updates"),
		},
		{
			name:    "leading ~ in base is expanded",
			mapping: &AddonPlatformMapping{BaseDir: "~/Games/switch"},
			cat:     romm.RomFileUpdate,
			want:    filepath.Join(expandHomeDir("~/Games/switch"), "update"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{}
			if tt.mapping != nil {
				c.AddonDirectoryMappings = map[string]AddonPlatformMapping{"switch": *tt.mapping}
			}
			if got := c.AddonDestination(sw, tt.cat, romDir); got != tt.want {
				t.Errorf("AddonDestination(%s) = %q, want %q", tt.cat, got, tt.want)
			}
		})
	}
}
