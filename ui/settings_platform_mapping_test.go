package ui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"grout/cfw"
	"grout/romm"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
)

// romDirEntries creates the given folders under a temp dir and returns its
// os.DirEntry listing, for exercising folder-detection logic.
func romDirEntries(t *testing.T, folders ...string) []os.DirEntry {
	t.Helper()
	tmp := t.TempDir()
	for _, f := range folders {
		if err := os.MkdirAll(filepath.Join(tmp, f), 0755); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestPreferredExistingOptionIndex(t *testing.T) {
	s := &PlatformMappingScreen{}

	// Option layout mirrors buildPlatformOptions: Skip, then Create/existing
	// entries, then a trailing Custom keyboard entry.
	opts := func(values ...string) []gaba.Option {
		out := []gaba.Option{{Value: ""}} // Skip
		for _, v := range values {
			out = append(out, gaba.Option{Value: v})
		}
		out = append(out, gaba.Option{Type: gaba.OptionTypeKeyboard, Value: ""}) // Custom
		return out
	}

	cases := []struct {
		name    string
		folders []string // physically present ROM folders
		cfwDirs []string // platform's folder names, primary first
		options []gaba.Option
		wantIdx int
	}{
		{
			// EmuDeck created "megadrive" for Sega Genesis; primary "genesis" is absent.
			name:    "alias folder auto-selected",
			folders: []string{"megadrive", "snes"},
			cfwDirs: []string{"genesis", "megadrive"},
			options: opts("genesis", "megadrive"), // Create 'genesis', /megadrive
			wantIdx: 2,
		},
		{
			// Both exist: primary wins.
			name:    "primary preferred over alias",
			folders: []string{"genesis", "megadrive"},
			cfwDirs: []string{"genesis", "megadrive"},
			options: opts("genesis", "megadrive"),
			wantIdx: 1,
		},
		{
			// Shared secondary alias must not steal selection from the primary.
			name:    "primary wins over shared secondary",
			folders: []string{"arcade", "neogeo"},
			cfwDirs: []string{"arcade", "mame", "fbneo", "neogeo"},
			options: opts("arcade", "neogeo"),
			wantIdx: 1,
		},
		{
			// Nothing present -> no auto-select (Skip).
			name:    "no match falls through to skip",
			folders: []string{"snes"},
			cfwDirs: []string{"genesis", "megadrive"},
			options: opts("genesis", "megadrive"),
			wantIdx: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entries := romDirEntries(t, tc.folders...)
			got := s.preferredExistingOptionIndex(tc.options, entries, tc.cfwDirs, cfw.ESDE)
			if got != tc.wantIdx {
				t.Errorf("preferredExistingOptionIndex() = %d, want %d", got, tc.wantIdx)
			}
		})
	}
}

// The platform-mapping filters (Category/Family) are built from the distinct values
// present across platforms; when RomM doesn't populate a field, the list is empty and
// the filter row must be hidden instead of showing a useless "All"-only picker (#247).
func TestDistinctPlatformValues(t *testing.T) {
	platforms := []romm.Platform{
		{Category: "console", Family: "Nintendo"},
		{Category: "handheld", Family: "Nintendo"},
		{Category: "console", Family: "Sega"}, // duplicate category
		{Category: "", Family: ""},            // empty values are skipped
	}

	gotCategory := distinctPlatformValues(platforms, func(p romm.Platform) string { return p.Category })
	if want := []string{"console", "handheld"}; !slices.Equal(gotCategory, want) {
		t.Errorf("category values = %v, want %v (sorted, deduped, no empties)", gotCategory, want)
	}

	gotFamily := distinctPlatformValues(platforms, func(p romm.Platform) string { return p.Family })
	if want := []string{"Nintendo", "Sega"}; !slices.Equal(gotFamily, want) {
		t.Errorf("family values = %v, want %v", gotFamily, want)
	}
}

func TestDistinctPlatformValues_AllEmptyHidesFilter(t *testing.T) {
	// RomM returned no category/family metadata for any platform: the result is empty,
	// which is the signal to hide the filter entirely.
	platforms := []romm.Platform{
		{Category: "", Family: "", Generation: 3},
		{Category: "", Family: "", Generation: 4},
	}

	if got := distinctPlatformValues(platforms, func(p romm.Platform) string { return p.Category }); len(got) != 0 {
		t.Errorf("expected no category values (filter hidden), got %v", got)
	}
	if got := distinctPlatformValues(platforms, func(p romm.Platform) string { return p.Family }); len(got) != 0 {
		t.Errorf("expected no family values (filter hidden), got %v", got)
	}
}
