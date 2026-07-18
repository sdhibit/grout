package esde

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// retroDeckPathsData mirrors the "paths" object of RetroDECK's retrodeck.json.
// RetroDECK lets users move each of these, so they must be read rather than
// assumed (requested by the RetroDECK maintainer in rommapp/grout#236).
type retroDeckPathsData struct {
	RDHomePath          string `json:"rd_home_path"`
	RomsPath            string `json:"roms_path"`
	SavesPath           string `json:"saves_path"`
	BiosPath            string `json:"bios_path"`
	DownloadedMediaPath string `json:"downloaded_media_path"`
}

type retroDeckConfig struct {
	Paths retroDeckPathsData `json:"paths"`
}

func parseRetroDeckConfig(data []byte) retroDeckPathsData {
	var cfg retroDeckConfig
	if err := json.Unmarshal(data, &cfg); err == nil && cfg.Paths != (retroDeckPathsData{}) {
		return cfg.Paths
	}
	// Tolerate a flat layout without the "paths" wrapper.
	var flat retroDeckPathsData
	_ = json.Unmarshal(data, &flat)
	return flat
}

// FindRetroDeckConfig returns the first retrodeck.json found, or "".
func FindRetroDeckConfig() string {
	candidates := []string{os.Getenv("RETRODECK_CFG")}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, "retrodeck", "retrodeck.json"))
	}
	candidates = append(candidates,
		filepath.Join(homeDir(), ".config", "retrodeck", "retrodeck.json"),
		filepath.Join(homeDir(), ".var", "app", "net.retrodeck.retrodeck", "config", "retrodeck", "retrodeck.json"),
	)
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

var retroDeckCache cached[retroDeckPathsData]

func retroDeckPaths() retroDeckPathsData {
	return retroDeckCache.get(func() retroDeckPathsData {
		configPath := FindRetroDeckConfig()
		if configPath == "" {
			return retroDeckPathsData{}
		}
		data, err := os.ReadFile(configPath)
		if err != nil {
			return retroDeckPathsData{}
		}
		return parseRetroDeckConfig(data)
	})
}
