package esde

import (
	"os"
	"path/filepath"
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

func isDir(t *testing.T, path string) bool {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return fi.IsDir()
}

func TestSupportsM3U(t *testing.T) {
	if !SupportsM3U("psx") {
		t.Error("psx (DuckStation/RetroArch) should support .m3u")
	}
	if SupportsM3U("ps2") {
		t.Error("ps2 (PCSX2) should NOT support .m3u")
	}
}

// m3u-capable system: the directory is interpreted as the <game>.m3u file, with
// every disc kept inside it.
func TestInterpretsDirectoryAsM3U(t *testing.T) {
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

	wantDir := filepath.Join(romDir, game+".m3u")
	if gamePath != wantDir {
		t.Errorf("returned game path = %q, want %q", gamePath, wantDir)
	}
	if !isDir(t, wantDir) {
		t.Error("interpreted entry should be a directory")
	}
	// The matching inner file (what ES-DE launches) and both discs live inside.
	for _, inner := range []string{game + ".m3u", "Final Fantasy VII (Disc 1).chd", "Final Fantasy VII (Disc 2).chd"} {
		if _, err := os.Stat(filepath.Join(wantDir, inner)); err != nil {
			t.Errorf("expected %q inside interpreted dir: %v", inner, err)
		}
	}
	if _, err := os.Stat(filepath.Join(romDir, game)); !os.IsNotExist(err) {
		t.Errorf("original extract dir should be gone, stat err = %v", err)
	}
}

// A playlist not already named after the game is renamed so the directory name
// stays clean.
func TestInterpretsDirectoryAsM3URenamesPlaylist(t *testing.T) {
	game := "Chrono Cross"
	romDir := mkExtract(t, game, map[string]string{
		"playlist.m3u":              "Chrono Cross (Disc 1).chd\nChrono Cross (Disc 2).chd\n",
		"Chrono Cross (Disc 1).chd": "d1",
		"Chrono Cross (Disc 2).chd": "d2",
	})

	gamePath, err := OrganizeMultiFileRom(filepath.Join(romDir, game), romDir, game, true)
	if err != nil {
		t.Fatalf("organize failed: %v", err)
	}

	wantDir := filepath.Join(romDir, game+".m3u")
	if gamePath != wantDir {
		t.Errorf("returned game path = %q, want %q", gamePath, wantDir)
	}
	if _, err := os.Stat(filepath.Join(wantDir, game+".m3u")); err != nil {
		t.Errorf("playlist should have been renamed to %q: %v", game+".m3u", err)
	}
	if _, err := os.Stat(filepath.Join(wantDir, "playlist.m3u")); !os.IsNotExist(err) {
		t.Error("original playlist name should be gone")
	}
}

// PS2/PCSX2 can't load an .m3u, so the directory is interpreted as the primary
// disc (disc 1 per the playlist), which ES-DE launches; the other discs stay
// inside for in-emulator swapping.
func TestInterpretsDirectoryAsPrimaryDisc(t *testing.T) {
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

	wantDir := filepath.Join(romDir, "Final Fantasy X (Disc 1).chd")
	if gamePath != wantDir {
		t.Errorf("returned game path = %q, want %q", gamePath, wantDir)
	}
	if !isDir(t, wantDir) {
		t.Error("interpreted entry should be a directory")
	}
	// The launch file (matching the dir name) and the other disc are both inside.
	for _, inner := range []string{"Final Fantasy X (Disc 1).chd", "Final Fantasy X (Disc 2).chd"} {
		if _, err := os.Stat(filepath.Join(wantDir, inner)); err != nil {
			t.Errorf("expected %q inside interpreted dir: %v", inner, err)
		}
	}
	if _, err := os.Stat(filepath.Join(romDir, game)); !os.IsNotExist(err) {
		t.Errorf("original extract dir should be gone, stat err = %v", err)
	}
}

func TestNoLaunchableFileHidesDir(t *testing.T) {
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
