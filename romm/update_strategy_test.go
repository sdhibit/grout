package romm

import (
	"testing"
	"time"
)

func TestParseUpdateVersion(t *testing.T) {
	cases := []struct {
		name string
		want uint64
		ok   bool
	}{
		{"Game [0100000000010800][v131072].nsp", 131072, true},
		{"Game (v65536).nsp", 65536, true},
		{"Game [V196608].xci", 196608, true}, // case-insensitive
		{"Zelda-update.nsp", 0, false},       // no token
		{"Version 5 Deluxe.nsp", 0, false},   // bare "v" words are ignored
		// "v" omitted: the bare bracketed number is the version, but the 16-digit
		// title ID in the same name must not be mistaken for it.
		{"Game [0100000000010800][131072].nsp", 131072, true},
		{"Game [131072].nsp", 131072, true},
		// Only a title ID, no version token → nothing usable.
		{"Game [0100000000010800].nsp", 0, false},
		// A hex title ID (contains letters) never matches the decimal token regex,
		// so the plain version is picked cleanly.
		{"Game [01007EF00011E000][262144].nsp", 262144, true},
		// Human-readable versions must be ignored: dotted ones don't match at all,
		// and no-dot ones aren't multiples of the Switch version granularity.
		{"Game [0100000000010800][v131072][1.2.0].nsp", 131072, true}, // v wins over "[1.2.0]"
		{"Game [131072][2.0.0].nsp", 131072, true},                    // dotted ignored, real version kept
		{"Game [1.20].nsp", 0, false},                                 // dotted human version only
		{"Game [2.0.0].nsp", 0, false},                                // dotted human version only
		{"Game [0100000000010800][2].nsp", 0, false},                  // no-dot human "2": not a version ID
	}
	for _, c := range cases {
		got, ok := parseUpdateVersion(c.name)
		if got != c.want || ok != c.ok {
			t.Errorf("parseUpdateVersion(%q) = (%d, %v), want (%d, %v)", c.name, got, ok, c.want, c.ok)
		}
	}
}

func TestLatestUpdateFileByVersionToken(t *testing.T) {
	rom := Rom{Files: []RomFile{
		{ID: 1, FileName: "base.nsp"},
		{ID: 2, FileName: "Game [v65536].nsp", Category: RomFileUpdate},
		{ID: 3, FileName: "Game [v196608].nsp", Category: RomFileUpdate}, // newest
		{ID: 4, FileName: "Game [v131072].nsp", Category: RomFileUpdate},
		{ID: 5, FileName: "dlc.nsp", Category: RomFileDLC},
	}}
	latest, ok := rom.LatestUpdateFile()
	if !ok || latest.ID != 3 {
		t.Fatalf("LatestUpdateFile = %+v (ok=%v), want ID 3", latest, ok)
	}
}

func TestLatestUpdateFileFallsBackToTimestamp(t *testing.T) {
	old := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	rom := Rom{Files: []RomFile{
		{ID: 1, FileName: "upd-a.nsp", Category: RomFileUpdate, LastModified: old},
		{ID: 2, FileName: "upd-b.nsp", Category: RomFileUpdate, LastModified: recent},
	}}
	latest, ok := rom.LatestUpdateFile()
	if !ok || latest.ID != 2 {
		t.Fatalf("LatestUpdateFile (timestamp fallback) = %+v (ok=%v), want ID 2", latest, ok)
	}
}

func TestVersionedUpdateOutranksUnversioned(t *testing.T) {
	// A file with a parsed version token is authoritative over a timestamp-only
	// one, even if the unversioned file was modified more recently.
	recent := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	rom := Rom{Files: []RomFile{
		{ID: 1, FileName: "Game [v65536].nsp", Category: RomFileUpdate},
		{ID: 2, FileName: "re-added-old.nsp", Category: RomFileUpdate, LastModified: recent},
	}}
	latest, ok := rom.LatestUpdateFile()
	if !ok || latest.ID != 1 {
		t.Fatalf("LatestUpdateFile = %+v (ok=%v), want the versioned ID 1", latest, ok)
	}
}

func TestUpdatesCumulativeByPlatform(t *testing.T) {
	if !(Rom{PlatformFSSlug: "switch"}).UpdatesCumulative() {
		t.Error("switch should be a cumulative-update platform")
	}
	if (Rom{PlatformFSSlug: "ps4"}).UpdatesCumulative() {
		t.Error("ps4 is not yet configured as cumulative; should default to independent")
	}
	if (Rom{}).UpdatesCumulative() {
		t.Error("unknown/empty platform must default to independent")
	}
}

func TestNoUpdateFiles(t *testing.T) {
	rom := Rom{Files: []RomFile{{ID: 1, FileName: "base.nsp"}, {ID: 2, FileName: "dlc.nsp", Category: RomFileDLC}}}
	if _, ok := rom.LatestUpdateFile(); ok {
		t.Error("expected no latest update when the ROM has no update files")
	}
}
