package ui

import (
	"path/filepath"
	"testing"

	"grout/romm"
)

func switchRom() romm.Rom {
	return romm.Rom{
		ID: 1,
		Files: []romm.RomFile{
			{ID: 10, FileName: "Zelda.nsp"},                                   // base (uncategorized)
			{ID: 12, FileName: "Zelda-upd.nsp", Category: romm.RomFileUpdate}, // update
			{ID: 13, FileName: "Zelda-dlc1.nsp", Category: romm.RomFileDLC},   // dlc
			{ID: 14, FileName: "Zelda-dlc2.nsp", Category: romm.RomFileDLC},   // dlc
			{ID: 15, FileName: "Zelda.txt", Category: romm.RomFileManual},     // auxiliary
		},
	}
}

func find(plan []plannedDownload, id int) (plannedDownload, bool) {
	for _, p := range plan {
		if p.FileID == id {
			return p, true
		}
	}
	return plannedDownload{}, false
}

func TestPlanAddonsBySelectionKeepsBaseAsAnchor(t *testing.T) {
	romDir := "/roms/switch"
	// The base is already downloaded, so the user left it unchecked and picked
	// only the update and dlc1 (not dlc2).
	selected := map[int]bool{12: true, 13: true}

	plan := planRomDownloads(switchRom(), selected, romDir, nil)

	// Base is still present (it anchors the gamelist/artwork entry), at romDir
	// root, flagged IsBase — but not fetched, since it wasn't selected.
	base, ok := find(plan, 10)
	if !ok {
		t.Fatal("base file missing from plan")
	}
	if base.Location != filepath.Join(romDir, "Zelda.nsp") || !base.IsBase {
		t.Errorf("base placement wrong: %+v", base)
	}
	if base.Download {
		t.Error("unselected base should not be flagged for download")
	}

	// Selected add-ons land in their category subfolder by default and are fetched.
	upd, ok := find(plan, 12)
	if !ok || upd.Location != filepath.Join(romDir, "update", "Zelda-upd.nsp") || upd.IsBase || !upd.Download {
		t.Errorf("update placement wrong: %+v (ok=%v)", upd, ok)
	}
	dlc1, ok := find(plan, 13)
	if !ok || dlc1.Location != filepath.Join(romDir, "dlc", "Zelda-dlc1.nsp") || !dlc1.Download {
		t.Errorf("dlc1 placement wrong: %+v (ok=%v)", dlc1, ok)
	}

	// Unselected DLC and the auxiliary manual must NOT be in the plan.
	if _, ok := find(plan, 14); ok {
		t.Error("unselected dlc2 should not be downloaded")
	}
	if _, ok := find(plan, 15); ok {
		t.Error("auxiliary manual should never be in the download plan")
	}
}

func TestPlanBaseSelectedIsFetched(t *testing.T) {
	// User re-checked the base along with the update.
	selected := map[int]bool{10: true, 12: true}
	plan := planRomDownloads(switchRom(), selected, "/roms/switch", nil)

	base, ok := find(plan, 10)
	if !ok || !base.Download {
		t.Errorf("selected base should be flagged for download: %+v (ok=%v)", base, ok)
	}
}

func TestPlanEmptySelectionFetchesNothing(t *testing.T) {
	// A non-nil, empty selection means the picker ran and the user unchecked
	// everything: the base stays as an anchor but nothing is fetched.
	plan := planRomDownloads(switchRom(), map[int]bool{}, "/roms/switch", nil)
	if len(plan) != 1 || plan[0].FileID != 10 {
		t.Fatalf("expected base-only anchor plan, got %+v", plan)
	}
	if plan[0].Download {
		t.Error("empty selection should not fetch the base")
	}
}

func TestPlanNilSelectionDownloadsBaseOnly(t *testing.T) {
	// A nil selection means no picker ran (bulk / non-categorized): fetch the
	// base, no add-ons.
	plan := planRomDownloads(switchRom(), nil, "/roms/switch", nil)
	if len(plan) != 1 || plan[0].FileID != 10 || !plan[0].Download {
		t.Fatalf("expected base-only fetch plan, got %+v", plan)
	}
}

func TestAutoAddonSelectionPicksBaseAndAllSupplemental(t *testing.T) {
	// Bulk download's implicit "grab everything" choice: base + every update/DLC,
	// but never the auxiliary manual.
	selected := autoAddonSelection(switchRom())

	for _, id := range []int{10, 12, 13, 14} { // base, update, dlc1, dlc2
		if !selected[id] {
			t.Errorf("expected file %d to be selected", id)
		}
	}
	if selected[15] { // manual
		t.Error("auxiliary manual should not be selected")
	}

	// Feeding it to the planner fetches the base plus all three add-ons.
	plan := planRomDownloads(switchRom(), selected, "/roms/switch", nil)
	for _, id := range []int{10, 12, 13, 14} {
		p, ok := find(plan, id)
		if !ok || !p.Download {
			t.Errorf("file %d should be fetched in a bulk grab: %+v (ok=%v)", id, p, ok)
		}
	}
	if _, ok := find(plan, 15); ok {
		t.Error("auxiliary manual must never be in the download plan")
	}
}

func TestPlanAddonDestinationOverride(t *testing.T) {
	// Redirect updates and DLC to an emulator-specific folder (the Switch case).
	addonDest := func(c romm.RomFileCategory) string {
		if c == romm.RomFileUpdate || c == romm.RomFileDLC {
			return "/home/deck/.local/share/eden/load"
		}
		return ""
	}
	plan := planRomDownloads(switchRom(), map[int]bool{12: true, 13: true}, "/roms/switch", addonDest)

	upd, _ := find(plan, 12)
	if upd.Location != filepath.Join("/home/deck/.local/share/eden/load", "Zelda-upd.nsp") {
		t.Errorf("update override wrong: %s", upd.Location)
	}
	// Base is unaffected by the add-on override.
	base, _ := find(plan, 10)
	if base.Location != filepath.Join("/roms/switch", "Zelda.nsp") {
		t.Errorf("base should ignore addon override: %s", base.Location)
	}
}

func TestPlanPlainSingleFileRom(t *testing.T) {
	rom := romm.Rom{ID: 2, Files: []romm.RomFile{{ID: 1, FileName: "Mario.sfc"}}}
	plan := planRomDownloads(rom, nil, "/roms/snes", nil)
	if len(plan) != 1 || plan[0].Location != filepath.Join("/roms/snes", "Mario.sfc") || !plan[0].IsBase || !plan[0].Download {
		t.Fatalf("plain rom plan wrong: %+v", plan)
	}
}
