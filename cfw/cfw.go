package cfw

import (
	"log"
	"os"
	"strings"
)

type CFW string

const (
	NextUI   CFW = "NEXTUI"
	MuOS     CFW = "MUOS"
	Knulli   CFW = "KNULLI"
	Spruce   CFW = "SPRUCE"
	ROCKNIX  CFW = "ROCKNIX"
	Trimui   CFW = "TRIMUI"
	Allium   CFW = "ALLIUM"
	Onion    CFW = "ONION"
	Koriki   CFW = "KORIKI"
	ArkOS    CFW = "ARKOS"
	Batocera CFW = "BATOCERA"
	MinUI    CFW = "MINUI"
	ESDE     CFW = "ESDE"
)

func GetCFW() CFW {
	cfwEnv := strings.ToUpper(os.Getenv("CFW"))
	cfw := CFW(cfwEnv)

	switch cfw {
	case MuOS, NextUI, Knulli, Spruce, ROCKNIX, Trimui, Allium, Onion, Koriki, ArkOS, Batocera, MinUI, ESDE:
		return cfw
	default:
		log.SetOutput(os.Stderr)
		log.Fatalf("Unsupported CFW: '%s'. Valid options: NextUI, muOS, Knulli, Spruce, ROCKNIX, Trimui, Allium, Onion, Koriki, ArkOS, Batocera, MinUI, ESDE", cfwEnv)
		return ""
	}
}

func (c CFW) IsBasedOnEmulationStation() bool {
	switch c {
	case Knulli, ROCKNIX, ArkOS, Batocera, ESDE:
		return true
	default:
		return false
	}
}

// UsesArtSuffixes reports whether downloaded art files need -thumb/-marquee/
// -boxback/-fanart filename suffixes. Batocera-style EmulationStation keeps
// every media type in one images/ directory, so suffixes avoid collisions.
// ES-DE instead keeps one directory per media type and matches files purely by
// ROM basename, so suffixes would break media matching.
func (c CFW) UsesArtSuffixes() bool {
	return c.IsBasedOnEmulationStation() && c != ESDE
}
