package romm

import "testing"

// A Switch-style categorized multi-part game: base files in the root (no
// category), an update, and two DLC — the layout RomM produces from update/ and
// dlc/ folders.
func switchStyleRom() Rom {
	return Rom{
		ID: 1,
		Files: []RomFile{
			{ID: 10, FileName: "Zelda.nsp"},                              // base (no category)
			{ID: 11, FileName: "Zelda[game].nsp", Category: RomFileGame}, // explicit base
			{ID: 12, FileName: "Zelda-update.nsp", Category: RomFileUpdate},
			{ID: 13, FileName: "Zelda-dlc1.nsp", Category: RomFileDLC},
			{ID: 14, FileName: "Zelda-dlc2.nsp", Category: RomFileDLC},
			{ID: 15, FileName: "Zelda.txt", Category: RomFileManual}, // auxiliary, not an add-on
		},
	}
}

func TestRomFileCategoryClassification(t *testing.T) {
	if !RomFileCategory("").IsBase() || !RomFileGame.IsBase() {
		t.Error("empty and \"game\" categories must be base")
	}
	if RomFileUpdate.IsBase() || RomFileDLC.IsBase() {
		t.Error("update/dlc must not be base")
	}
	// Supplemental add-ons: content layered on top of the base game.
	for _, c := range []RomFileCategory{RomFileUpdate, RomFileDLC, RomFilePatch} {
		if !c.IsSupplementalAddon() {
			t.Errorf("%q should be a supplemental add-on", c)
		}
		if c.IsStandaloneVersion() {
			t.Errorf("%q should not be a standalone version", c)
		}
	}

	// Standalone versions: whole, independently playable ROMs. Hacks and prototypes
	// are versions, NOT add-ons — the distinction that keeps them out of the add-on
	// picker and in the File Version picker.
	for _, c := range []RomFileCategory{"", RomFileGame, RomFileHack, RomFileMod, RomFileTranslation, RomFileDemo, RomFilePrototype} {
		if !c.IsStandaloneVersion() {
			t.Errorf("%q should be a standalone version", c)
		}
		if c.IsSupplementalAddon() {
			t.Errorf("%q should not be a supplemental add-on", c)
		}
	}

	// Auxiliary files are neither versions nor add-ons.
	for _, c := range []RomFileCategory{RomFileManual, RomFileSoundtrack, RomFileScreenshot, RomFileCheat} {
		if c.IsSupplementalAddon() || c.IsStandaloneVersion() {
			t.Errorf("%q (auxiliary) should be neither add-on nor version", c)
		}
	}
}

func TestVersionFiles(t *testing.T) {
	// A game with only base + update + dlc exposes just its base files as versions.
	v := switchStyleRom().VersionFiles()
	if len(v) != 2 {
		t.Fatalf("expected 2 version files (the base files), got %d", len(v))
	}
	for _, f := range v {
		if !f.IsBase() {
			t.Errorf("file %d (%s) should not be a version", f.ID, f.Category)
		}
	}

	// A romhack is a standalone version alongside the base; an update is not.
	rom := Rom{Files: []RomFile{
		{ID: 1, FileName: "Sonic.md"},
		{ID: 2, FileName: "Sonic-hack.md", Category: RomFileHack},
		{ID: 3, FileName: "Sonic-upd.md", Category: RomFileUpdate},
	}}
	v = rom.VersionFiles()
	ids := map[int]bool{}
	for _, f := range v {
		ids[f.ID] = true
	}
	if len(v) != 2 || !ids[1] || !ids[2] || ids[3] {
		t.Errorf("version files = %v, want base(1)+hack(2) but not update(3)", ids)
	}
}

func TestBaseFiles(t *testing.T) {
	base := switchStyleRom().BaseFiles()
	if len(base) != 2 {
		t.Fatalf("expected 2 base files (uncategorized + explicit game), got %d", len(base))
	}
	for _, f := range base {
		if !f.IsBase() {
			t.Errorf("file %d (%s) is not base", f.ID, f.Category)
		}
	}
}

func TestHasAddons(t *testing.T) {
	if !switchStyleRom().HasAddons() {
		t.Error("switch-style rom should report add-ons")
	}
	plain := Rom{Files: []RomFile{{ID: 1, FileName: "Mario.sfc"}}}
	if plain.HasAddons() {
		t.Error("a plain single-file rom should have no add-ons")
	}
	// A rom whose only extra files are auxiliary (manual) has no game-content add-ons.
	auxOnly := Rom{Files: []RomFile{
		{ID: 1, FileName: "g.sfc"},
		{ID: 2, FileName: "g.pdf", Category: RomFileManual},
	}}
	if auxOnly.HasAddons() {
		t.Error("manual-only rom should not report game-content add-ons")
	}
}

func TestAddonGroupsOrderingAndContents(t *testing.T) {
	groups := switchStyleRom().AddonGroups()
	if len(groups) != 2 {
		t.Fatalf("expected 2 add-on groups (update, dlc), got %d", len(groups))
	}
	// Updates must come before DLC (stable display order).
	if groups[0].Category != RomFileUpdate {
		t.Errorf("first group = %q, want update", groups[0].Category)
	}
	if groups[1].Category != RomFileDLC {
		t.Errorf("second group = %q, want dlc", groups[1].Category)
	}
	if len(groups[0].Files) != 1 {
		t.Errorf("update group should have 1 file, got %d", len(groups[0].Files))
	}
	if len(groups[1].Files) != 2 {
		t.Errorf("dlc group should have 2 files, got %d", len(groups[1].Files))
	}
	// The manual (auxiliary) must never appear in add-on groups.
	for _, g := range groups {
		for _, f := range g.Files {
			if f.Category == RomFileManual {
				t.Error("manual leaked into add-on groups")
			}
		}
	}
}
