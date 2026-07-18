package cfw

import (
	"grout/cfw/arkos"
	"grout/cfw/batocera"
	"grout/cfw/esde"
	"grout/cfw/knulli"
	"grout/cfw/muos"
	"grout/cfw/rocknix"
	"grout/internal/emulationstation"
	"grout/internal/gamelist"
	"path/filepath"
	"strings"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
)

func scheduleESRestart() {
	err := emulationstation.ScheduleESRestart()
	if err != nil {
		gaba.GetLogger().Debug("Unable to schedule ES restart", "error", err)
	}
}

func AddGroutToGamelist(c CFW) {
	switch c {
	case Knulli:
		gamelist.AddGroutEntry(knulli.GetGroutGamelist(), "./Grout/Grout.sh")
	case ROCKNIX:
		gamelist.AddGroutEntry(rocknix.GetGroutGamelist(), "./Grout.sh")
	case ArkOS:
		gamelist.AddGroutEntry(arkos.GetGroutGamelist(), "./Grout.sh")
	case Batocera:
		gamelist.AddGroutEntry(batocera.GetGroutGamelist(), "./Grout/Grout.sh")
	case ESDE:
		// ES-DE runs grout as a child process and has no restart hook like
		// batocera-es-swissknife, so no ES restart is scheduled; ES-DE picks up
		// gamelist changes on its next rescan or restart.
		gamelist.AddGroutEntry(esde.GetGroutGamelist(), "./Grout.sh")
		return
	default:
		return
	}
	scheduleESRestart()
}

// esdeGamelistOptions writes gamelists to ES-DE's appdata layout and records
// game paths relative to the system's ROM folder, as ES-DE expects.
func esdeGamelistOptions() gamelist.Options {
	return gamelist.Options{
		GamelistPath: func(romDir string, filename gamelist.FileName) string {
			return esde.GetGamelistPath(romDir, string(filename))
		},
		GamePath: func(entry gamelist.RomGameEntry) string {
			rel, err := filepath.Rel(entry.RomDirectory, entry.GamePath)
			if err != nil || strings.HasPrefix(rel, "..") {
				return entry.GamePath
			}
			return "./" + rel
		},
	}
}

func FillGamesMetadata(entries []gamelist.RomGameEntry) {
	logger := gaba.GetLogger()
	switch GetCFW() {
	case Knulli, ROCKNIX, ArkOS, Batocera:
		if err := gamelist.AddRomGamesToGamelist(entries, gamelist.GameListFileName); err != nil {
			logger.Warn("Failed to add games to ES gamelist.xml", "error", err)
		}
		scheduleESRestart()
	case ESDE:
		if err := gamelist.AddRomGamesToGamelist(entries, gamelist.GameListFileName, esdeGamelistOptions()); err != nil {
			logger.Warn("Failed to add games to ES-DE gamelist.xml", "error", err)
		}
	case Spruce, Allium, Onion, Koriki:
		if err := gamelist.AddRomGamesToGamelist(entries, gamelist.MiyooGameListFileName); err != nil {
			logger.Warn("Failed to add games to miyoogamelist.xml", "error", err)
		}
	case MuOS:
		for _, entry := range entries {
			muos.AddGameDescription(entry)
		}
	default:
		return
	}
}
