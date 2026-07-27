package ui

import (
	"errors"
	"path/filepath"

	"grout/internal/fileutil"
	"grout/romm"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	gabaconst "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

type FileVersionScreen struct{}

func NewFileVersionScreen() *FileVersionScreen {
	return &FileVersionScreen{}
}

// FileVersionResult reports the outcome of the version picker. Confirmed is
// false when the user backed out (the download should be cancelled).
type FileVersionResult struct {
	Confirmed      bool
	SelectedFileID int
}

// Draw shows a single-select list of a game's selectable versions — the base
// game plus any standalone alternate builds (hacks, mods, translations, demos,
// prototypes) — for the user to pick one to download. Supplemental add-ons
// (updates/DLC) are deliberately excluded; those are chosen in the add-on step,
// never as a whole-game version here. Files already present in romDir are marked.
// This replaces the old in-detail dropdown, which rendered unreadably over the
// ROM description.
func (s *FileVersionScreen) Draw(game romm.Rom, romDir string) (FileVersionResult, error) {
	versionFiles := game.VersionFiles()
	items := make([]gaba.MenuItem, 0, len(versionFiles))
	for _, f := range versionFiles {
		text := f.FileName
		if fileutil.FileExists(filepath.Join(romDir, f.FileName)) {
			text = gabaconst.Download + " " + text
		}
		items = append(items, gaba.MenuItem{Text: text, Metadata: f.ID})
	}
	// Nothing (or only one) to choose: proceed without a picker.
	if len(items) == 0 {
		return FileVersionResult{Confirmed: true}, nil
	}

	options := gaba.DefaultListOptions(
		i18n.Localize(&goi18n.Message{ID: "file_version_title", Other: "Select Version to Download"}, nil),
		items,
	)
	options.UseSmallTitle = true
	options.FooterHelpItems = []gaba.FooterHelpItem{
		FooterBack(),
		FooterDownload(),
	}

	res, err := gaba.List(options)
	if err != nil {
		if errors.Is(err, gaba.ErrCancelled) {
			return FileVersionResult{Confirmed: false}, nil
		}
		return FileVersionResult{}, err
	}
	if res.Action != gaba.ListActionSelected || len(res.Selected) == 0 {
		return FileVersionResult{Confirmed: false}, nil
	}
	idx := res.Selected[0]
	if idx < 0 || idx >= len(items) {
		return FileVersionResult{Confirmed: false}, nil
	}
	id, _ := items[idx].Metadata.(int)
	return FileVersionResult{Confirmed: true, SelectedFileID: id}, nil
}
