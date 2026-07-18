package esde

import "sync"

// Variant discovery pulls actual install locations from the files EmuDeck and
// RetroDECK maintain, and from ES-DE's own es_settings.xml, so relocated
// installs (SD card, custom root) resolve without manual configuration.
//
// Discovery results are cached; Configure resets the caches because the
// selected variant (and with it the appdata dir es_settings.xml is read from)
// may have changed.

type cached[T any] struct {
	mu    sync.Mutex
	value T
	valid bool
}

func (c *cached[T]) get(load func() T) T {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid {
		c.value = load()
		c.valid = true
	}
	return c.value
}

func (c *cached[T]) reset() {
	c.mu.Lock()
	c.valid = false
	c.mu.Unlock()
}

func resetDiscoveryCaches() {
	retroDeckCache.reset()
	emuDeckPathCache.reset()
	esSettingsCache.reset()
}

func discoveredBasePath() string {
	switch CurrentVariant() {
	case VariantRetroDeck:
		return retroDeckPaths().RDHomePath
	case VariantEmuDeck:
		return emuDeckEmulationPath()
	}
	return ""
}

func discoveredRomsPath() string {
	if CurrentVariant() == VariantRetroDeck {
		if p := retroDeckPaths().RomsPath; p != "" {
			return p
		}
	}
	return esSettingsValue("ROMDirectory")
}

func discoveredBiosPath() string {
	if CurrentVariant() == VariantRetroDeck {
		return retroDeckPaths().BiosPath
	}
	return ""
}

func discoveredSavesPath() string {
	if CurrentVariant() == VariantRetroDeck {
		return retroDeckPaths().SavesPath
	}
	return ""
}

// discoveredAppDataPath exists for symmetry; there is currently no discovery
// source for the appdata dir beyond the variant defaults (ESDE_APPDATA_DIR is
// handled as an env override in GetAppDataDir).
func discoveredAppDataPath() string {
	return ""
}

func discoveredMediaPath() string {
	if CurrentVariant() == VariantRetroDeck {
		if p := retroDeckPaths().DownloadedMediaPath; p != "" {
			return p
		}
	}
	return esSettingsValue("MediaDirectory")
}
