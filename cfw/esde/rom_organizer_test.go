package esde

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mkExtract creates <romDir>/<gameName>/ populated with the given files (each with
// dummy content) and returns the rom directory.
func mkExtract(t *testing.T, gameName string, files map[string]string) string {
	t.Helper()
	romDir := t.TempDir()
	extractDir := filepath.Join(romDir, gameName)
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(extractDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return romDir
}

func TestSupportsM3U(t *testing.T) {
	if !SupportsM3U("psx") {
		t.Error("psx (DuckStation/RetroArch) should support .m3u")
	}
	if SupportsM3U("ps2") {
		t.Error("ps2 (PCSX2) should NOT support .m3u")
	}
}

func TestOrganizeM3UHidesDiscsAndRewritesPlaylist(t *testing.T) {
	game := "Final Fantasy VII"
	romDir := mkExtract(t, game, map[string]string{
		game + ".m3u":                    "Final Fantasy VII (Disc 1).chd\nFinal Fantasy VII (Disc 2).chd\n",
		"Final Fantasy VII (Disc 1).chd": "d1",
		"Final Fantasy VII (Disc 2).chd": "d2",
	})

	gamePath, err := OrganizeMultiFileRom(filepath.Join(romDir, game), romDir, game, true)
	if err != nil {
		t.Fatalf("organize failed: %v", err)
	}

	rootM3U := filepath.Join(romDir, game+".m3u")
	if gamePath != rootM3U {
		t.Errorf("returned game path = %q, want %q", gamePath, rootM3U)
	}
	content, err := os.ReadFile(rootM3U)
	if err != nil {
		t.Fatalf("root .m3u missing: %v", err)
	}
	got := strings.TrimSpace(string(content))
	want := ".Final Fantasy VII/Final Fantasy VII (Disc 1).chd\n.Final Fantasy VII/Final Fantasy VII (Disc 2).chd"
	if got != want {
		t.Errorf("m3u content =\n%q\nwant\n%q", got, want)
	}

	hiddenDir := filepath.Join(romDir, "."+game)
	for _, disc := range []string{"Final Fantasy VII (Disc 1).chd", "Final Fantasy VII (Disc 2).chd"} {
		if _, err := os.Stat(filepath.Join(hiddenDir, disc)); err != nil {
			t.Errorf("disc %q not in hidden dir: %v", disc, err)
		}
	}
	if _, err := os.Stat(filepath.Join(romDir, game)); !os.IsNotExist(err) {
		t.Errorf("extract dir should be gone, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(hiddenDir, game+".m3u")); !os.IsNotExist(err) {
		t.Error("original in-dir .m3u should have been removed")
	}
}

// PS2/PCSX2 can't load an .m3u, so the primary disc (disc 1 per the playlist) is
// promoted to the system root as the single launchable entry and the rest hidden.
func TestOrganizeNoM3USupportPromotesPrimaryDisc(t *testing.T) {
	game := "Final Fantasy X"
	romDir := mkExtract(t, game, map[string]string{
		game + ".m3u":                  "Final Fantasy X (Disc 1).chd\nFinal Fantasy X (Disc 2).chd\n",
		"Final Fantasy X (Disc 1).chd": "d1",
		"Final Fantasy X (Disc 2).chd": "d2",
	})

	gamePath, err := OrganizeMultiFileRom(filepath.Join(romDir, game), romDir, game, false)
	if err != nil {
		t.Fatalf("organize failed: %v", err)
	}

	wantPrimary := filepath.Join(romDir, "Final Fantasy X (Disc 1).chd")
	if gamePath != wantPrimary {
		t.Errorf("returned game path = %q, want %q", gamePath, wantPrimary)
	}
	if _, err := os.Stat(wantPrimary); err != nil {
		t.Errorf("primary disc not promoted to root: %v", err)
	}

	// No playlist should remain in the system root (PCSX2 can't use it).
	if _, err := os.Stat(filepath.Join(romDir, game+".m3u")); !os.IsNotExist(err) {
		t.Error("no root .m3u expected for a non-m3u system")
	}

	// The second disc and the (unusable) playlist are hidden.
	hiddenDir := filepath.Join(romDir, "."+game)
	if _, err := os.Stat(filepath.Join(hiddenDir, "Final Fantasy X (Disc 2).chd")); err != nil {
		t.Errorf("disc 2 should be hidden: %v", err)
	}
	if _, err := os.Stat(filepath.Join(hiddenDir, game+".m3u")); err != nil {
		t.Errorf("unusable .m3u should be hidden: %v", err)
	}
	if _, err := os.Stat(filepath.Join(romDir, game)); !os.IsNotExist(err) {
		t.Errorf("extract dir should be gone, stat err = %v", err)
	}
}

func TestOrganizeNoDiscsHidesWholeDir(t *testing.T) {
	game := "Weird Game"
	romDir := mkExtract(t, game, map[string]string{}) // empty extract dir

	gamePath, err := OrganizeMultiFileRom(filepath.Join(romDir, game), romDir, game, true)
	if err != nil {
		t.Fatalf("organize failed: %v", err)
	}
	if gamePath != "" {
		t.Errorf("expected empty game path when nothing launchable, got %q", gamePath)
	}
	if _, err := os.Stat(filepath.Join(romDir, "."+game)); err != nil {
		t.Errorf("directory should have been hidden: %v", err)
	}
}
