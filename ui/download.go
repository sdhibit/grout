package ui

import (
	"crypto/tls"
	"errors"
	"fmt"
	"grout/cfw"
	"grout/cfw/esde"
	"grout/cfw/muos"
	"grout/internal"
	"grout/internal/artutil"
	"grout/internal/fileutil"
	"grout/internal/gamelist"
	"grout/internal/imageutil"
	"grout/romm"
	_ "image/gif"
	_ "image/jpeg"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/i18n"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"go.uber.org/atomic"
)

type DownloadInput struct {
	Config           internal.Config
	Host             romm.Host
	Platform         romm.Platform
	SelectedGames    []romm.Rom
	AllGames         []romm.Rom
	SearchFilter     string
	SelectedFileID   int
	SelectedAddonIDs []int // add-on RomFile IDs to fetch alongside the base game (categorized multi-part ROMs)
	// IncludeAllAddons makes a bulk download grab each game's base plus every one
	// of its updates/DLC without an add-on picker. It mirrors how bulk already
	// fetches base/simple/multi-disc games: everything selected is (re)downloaded,
	// with no skip-if-already-on-disk check.
	IncludeAllAddons bool
}

type DownloadOutput struct {
	DownloadedGames []romm.Rom
	Platform        romm.Platform
	AllGames        []romm.Rom
	SearchFilter    string
}

type DownloadScreen struct{}

type artDownload struct {
	URL      string
	Location string
	GameName string
	IsImage  bool
}

func NewDownloadScreen() *DownloadScreen {
	return &DownloadScreen{}
}

func (s *DownloadScreen) Execute(config internal.Config, host romm.Host, platform romm.Platform, selectedGames []romm.Rom, allGames []romm.Rom, searchFilter string, selectedFileID int, selectedAddonIDs []int, includeAllAddons bool) DownloadOutput {
	result, err := s.draw(DownloadInput{
		Config:           config,
		Host:             host,
		Platform:         platform,
		SelectedGames:    selectedGames,
		AllGames:         allGames,
		SelectedFileID:   selectedFileID,
		SelectedAddonIDs: selectedAddonIDs,
		IncludeAllAddons: includeAllAddons,
		SearchFilter:     searchFilter,
	})

	if err != nil {
		gaba.GetLogger().Error("Download failed", "error", err)
		return DownloadOutput{
			AllGames:     allGames,
			Platform:     platform,
			SearchFilter: searchFilter,
		}
	}

	if len(result.DownloadedGames) > 0 {
		gaba.GetLogger().Debug("Successfully downloaded games", "count", len(result.DownloadedGames))
	}

	return result
}

func (s *DownloadScreen) draw(input DownloadInput) (DownloadOutput, error) {
	logger := gaba.GetLogger()

	output := DownloadOutput{
		Platform:     input.Platform,
		AllGames:     input.AllGames,
		SearchFilter: input.SearchFilter,
	}

	downloads, artDownloads, gamelistEntries, ignoreDirs := s.buildDownloads(input.Config, input.Host, input.Platform, input.SelectedGames, input.SelectedFileID, input.SelectedAddonIDs, input.IncludeAllAddons)

	// Nothing to fetch — e.g. every file was already downloaded and the user
	// unchecked them all in the add-on picker. Skip the download manager.
	if len(downloads) == 0 {
		logger.Debug("No files to download; nothing selected")
		return output, nil
	}

	headers := make(map[string]string)
	headers["Authorization"] = input.Host.AuthHeader()

	slices.SortFunc(downloads, func(a, b gaba.Download) int {
		return strings.Compare(strings.ToLower(a.DisplayName), strings.ToLower(b.DisplayName))
	})

	logger.Debug("Starting ROM download", "downloads", downloads)

	res, err := gaba.DownloadManager(downloads, headers, gaba.DownloadManagerOptions{
		AutoContinueOnComplete: input.Config.DownloadArt,
		SkipSSLVerification:    input.Host.InsecureSkipVerify,
	})
	if err != nil {
		logger.Error("Error downloading", "error", err)

		// Clean up any partial downloads when cancelled
		if errors.Is(err, gaba.ErrCancelled) {
			for _, d := range downloads {
				fileutil.DeleteFile(d.Location)
			}
		}

		return output, err
	}

	logger.Debug("Download results", "completed", len(res.Completed), "failed", len(res.Failed))

	if len(res.Failed) > 0 {
		for _, f := range res.Failed {
			logger.Warn("Download failed", "name", f.Download.DisplayName, "url", f.Download.URL, "error", f.Error)
		}

		for _, g := range downloads {
			failedMatch := slices.ContainsFunc(res.Failed, func(de gaba.DownloadError) bool {
				return de.Download.DisplayName == g.DisplayName
			})
			if failedMatch {
				fileutil.DeleteFile(g.Location)
			}
		}
	}

	if len(res.Completed) == 0 {
		return output, nil
	}

	// Tell ES-DE to skip the add-on folders (update/, dlc/, …) so their files
	// don't each show up as a separate game entry.
	for _, dir := range ignoreDirs {
		if err := esde.MarkDirectoryIgnored(dir); err != nil {
			logger.Warn("Failed to mark add-on folder ignored for ES-DE", "dir", dir, "error", err)
		}
	}

	for _, g := range input.SelectedGames {
		if !g.HasMultipleFiles {
			continue
		}

		completed := slices.ContainsFunc(res.Completed, func(d gaba.Download) bool {
			return d.DisplayName == g.Name
		})
		if !completed {
			continue
		}

		gamePlatform := input.Platform
		if input.Platform.ID == 0 && g.PlatformID != 0 {
			gamePlatform = romm.Platform{
				ID:     g.PlatformID,
				FSSlug: g.PlatformFSSlug,
				Name:   g.PlatformDisplayName,
			}
		}

		tmpZipPath := filepath.Join(fileutil.TempDir(), fmt.Sprintf("grout_multirom_%d.zip", g.ID))
		romDirectory := input.Config.GetPlatformRomDirectory(gamePlatform)
		extractDir := filepath.Join(romDirectory, g.FsNameNoExt)

		progress := &atomic.Float64{}
		_, err := gaba.ProcessMessage(
			i18n.Localize(&goi18n.Message{ID: "download_extracting", Other: "Extracting {{.Name}}..."}, map[string]interface{}{"Name": g.DisplayName}),
			gaba.ProcessMessageOptions{
				ShowThemeBackground: true,
				ShowProgressBar:     true,
				Progress:            progress,
			},
			func() (interface{}, error) {
				logger.Debug("Extracting multi-file ROM", "game", g.DisplayName, "dest", extractDir)

				if err := fileutil.Unzip(tmpZipPath, extractDir, progress); err != nil {
					logger.Error("Failed to extract multi-file ROM", "game", g.DisplayName, "error", err)
					os.Remove(tmpZipPath)
					return nil, err
				}

				// esdeGamePath, when set, is the exact path ES-DE's organizer chose
				// for the game entry (a rewritten .m3u, or a promoted primary disc for
				// emulators like PCSX2 that can't load playlists).
				esdeGamePath := ""
				if cfw.GetCFW() == cfw.MuOS {
					if err := muos.OrganizeMultiFileRom(extractDir, romDirectory, g.FsNameNoExt); err != nil {
						logger.Error("Failed to organize multi-file ROM for muOS", "game", g.FsNameNoExt, "error", err)
						os.Remove(tmpZipPath)
						os.RemoveAll(extractDir)
						return nil, err
					}
				} else if cfw.GetCFW() == cfw.ESDE {
					// ES-DE scans every file in the system directory, so turn the
					// extracted folder into a single "directory interpreted as a file"
					// — one game entry that launches the right inner file (a playlist,
					// or the primary disc for PS2/PCSX2, which can't use .m3u).
					p, err := esde.OrganizeMultiFileRom(extractDir, romDirectory, g.FsNameNoExt, esde.SupportsM3U(gamePlatform.FSSlug))
					if err != nil {
						logger.Error("Failed to organize multi-file ROM for ES-DE", "game", g.FsNameNoExt, "error", err)
						os.Remove(tmpZipPath)
						os.RemoveAll(extractDir)
						return nil, err
					}
					esdeGamePath = p
				}

				if err := os.Remove(tmpZipPath); err != nil {
					logger.Warn("Failed to remove temp zip file", "path", tmpZipPath, "error", err)
				}

				// Update the gamelist entry to point to the extracted file
				// instead of the (now deleted) temporary zip.
				newGamePath := esdeGamePath
				if newGamePath == "" {
					newGamePath = resolveExtractedGamePath(romDirectory, extractDir, g.FsNameNoExt)
				}
				for i, entry := range gamelistEntries {
					if entry.Game.ID == g.ID {
						gamelistEntries[i].GamePath = newGamePath
						break
					}
				}

				return nil, nil
			},
		)

		if err != nil {
			continue
		}
	}

	if input.Config.UnzipDownloads {
		for _, g := range input.SelectedGames {
			if g.HasMultipleFiles {
				continue
			}

			completed := slices.ContainsFunc(res.Completed, func(d gaba.Download) bool {
				return d.DisplayName == g.Name
			})
			if !completed {
				continue
			}

			gamePlatform := input.Platform
			if input.Platform.ID == 0 && g.PlatformID != 0 {
				gamePlatform = romm.Platform{
					ID:     g.PlatformID,
					FSSlug: g.PlatformFSSlug,
					Name:   g.PlatformDisplayName,
				}
			}

			if len(g.Files) > 0 {
				ext := strings.ToLower(filepath.Ext(g.Files[0].FileName))
				if ext == ".zip" || ext == ".7z" {
					romDirectory := input.Config.GetPlatformRomDirectory(gamePlatform)
					archivePath := filepath.Join(romDirectory, g.Files[0].FileName)

					progress := &atomic.Float64{}
					_, err := gaba.ProcessMessage(
						i18n.Localize(&goi18n.Message{ID: "download_extracting", Other: "Extracting {{.Name}}..."}, map[string]interface{}{"Name": g.Name}),
						gaba.ProcessMessageOptions{
							ShowThemeBackground: true,
							ShowProgressBar:     true,
							Progress:            progress,
						},
						func() (interface{}, error) {
							logger.Debug("Extracting single-file ROM", "game", g.Name, "file", archivePath)

							var archiveFiles []string
							var extractErr error
							if ext == ".7z" {
								archiveFiles, extractErr = fileutil.SevenZipFileNames(archivePath)
								if extractErr == nil {
									extractErr = fileutil.Un7zip(archivePath, romDirectory, progress)
								}
							} else {
								archiveFiles, extractErr = fileutil.ZipFileNames(archivePath)
								if extractErr == nil {
									extractErr = fileutil.Unzip(archivePath, romDirectory, progress)
								}
							}

							if extractErr != nil {
								logger.Error("Failed to extract single-file ROM", "game", g.Name, "error", extractErr)
								return nil, extractErr
							}

							if err := os.Remove(archivePath); err != nil {
								logger.Warn("Failed to remove archive file after extraction", "path", archivePath, "error", err)
							}

							if len(archiveFiles) > 0 {
								gamePath := archiveFiles[0]
								if len(archiveFiles) > 1 {
									for _, f := range archiveFiles {
										if strings.ToLower(filepath.Ext(f)) == ".m3u" {
											gamePath = f
											break
										}
									}
								}
								for i, entry := range gamelistEntries {
									if entry.Game.ID == g.ID {
										gamelistEntries[i].GamePath = filepath.Join(romDirectory, gamePath)
										break
									}
								}
							}

							return nil, nil
						},
					)

					if err != nil {
						logger.Warn("Failed to extract ROM, keeping archive file", "game", g.Name)
						continue
					}
				}
			}
		}
	}

	downloadedGames := make([]romm.Rom, 0, len(res.Completed))
	for _, g := range input.SelectedGames {
		if slices.ContainsFunc(res.Completed, func(d gaba.Download) bool {
			return d.DisplayName == g.Name
		}) {
			downloadedGames = append(downloadedGames, g)
		}
	}

	logger.Debug("Download complete", "successful", len(downloadedGames), "attempted", len(input.SelectedGames))

	if len(artDownloads) > 0 && len(downloadedGames) > 0 {
		progress := &atomic.Float64{}
		_, err := gaba.ProcessMessage(
			i18n.Localize(&goi18n.Message{ID: "download_artwork", Other: "Downloading artwork..."}, nil),
			gaba.ProcessMessageOptions{
				ShowThemeBackground: true,
				ShowProgressBar:     true,
				Progress:            progress,
			},
			func() (interface{}, error) {
				s.downloadArt(artDownloads, downloadedGames, headers, progress, input.Host.InsecureSkipVerify)
				return nil, nil
			},
		)

		if err != nil {
			logger.Warn("Art download process encountered an error", "error", err)
		}
	}

	cfw.FillGamesMetadata(gamelistEntries)

	output.DownloadedGames = downloadedGames
	return output, nil
}

// fileContentURL builds the RomM endpoint for downloading a single named file
// from a ROM by its file ID.
func fileContentURL(host romm.Host, romID int, fileName string, fileID int) string {
	u, _ := url.JoinPath(host.URL(), "/api/roms/", strconv.Itoa(romID), "content", fileName)
	return u + "?" + url.Values{"file_ids": {strconv.Itoa(fileID)}}.Encode()
}

func (s *DownloadScreen) buildDownloads(config internal.Config, host romm.Host, platform romm.Platform, games []romm.Rom, selectedFileID int, selectedAddonIDs []int, includeAllAddons bool) ([]gaba.Download, []artDownload, []gamelist.RomGameEntry, []string) {
	downloads := make([]gaba.Download, 0, len(games))
	artDownloads := make([]artDownload, 0, len(games))
	gamesSummaries := make([]gamelist.RomGameEntry, 0, len(games))

	// ignoreDirs collects add-on subfolders (update/, dlc/, …) that live inside an
	// ES-DE ROM directory. They get a noload.txt marker after download so ES-DE
	// skips them instead of listing each add-on file as its own game.
	ignoreDirSet := make(map[string]bool)
	markIgnored := cfw.GetCFW() == cfw.ESDE

	// A nil selection means no add-on picker ran (bulk or non-categorized
	// download): base only. A non-nil (even empty) selection is an explicit
	// picker result, so keep the nil-ness intact to pass that distinction on to
	// planRomDownloads.
	var addonIDSet map[int]bool
	if selectedAddonIDs != nil {
		addonIDSet = make(map[int]bool, len(selectedAddonIDs))
		for _, id := range selectedAddonIDs {
			addonIDSet[id] = true
		}
	}

	for _, g := range games {
		gamelistRomEntry := gamelist.RomGameEntry{
			Game:     &g,
			Platform: &platform,
		}
		gamePlatform := platform
		if platform.ID == 0 && g.PlatformID != 0 {
			gamePlatform = romm.Platform{
				ID:     g.PlatformID,
				FSSlug: g.PlatformFSSlug,
				Name:   g.PlatformDisplayName,
			}
		}

		romDirectory := config.GetPlatformRomDirectory(gamePlatform)
		gamelistRomEntry.RomDirectory = romDirectory
		downloadLocation := ""

		sourceURL := ""

		// queuePrimaryDownload is whether the game's primary file (the base) is
		// actually fetched. It stays true for every path except a categorized game
		// whose base is already on disk and left unselected in the add-on picker —
		// there we keep the base only as the gamelist/artwork anchor.
		queuePrimaryDownload := true

		// extraDownloads holds add-on files (updates, DLC, …) for a categorized
		// multi-part ROM. The base file is handled via downloadLocation/sourceURL
		// below so it keeps the game's artwork and gamelist entry.
		var extraDownloads []gaba.Download
		if g.HasAddons() {
			// A bulk download (no add-on picker) grabs the base plus every one of
			// this game's updates/DLC. Everything is (re)fetched, matching how bulk
			// already treats base/simple/multi-disc games.
			gameAddonIDs := addonIDSet
			if includeAllAddons {
				gameAddonIDs = autoAddonSelection(g)
			}
			// A configured per-platform, per-category add-on directory (e.g. a
			// Switch emulator's watched update/DLC folders) overrides the default
			// category subfolder placement.
			plan := planRomDownloads(g, gameAddonIDs, romDirectory, func(cat romm.RomFileCategory) string {
				return config.AddonDestination(gamePlatform, cat, romDirectory)
			})
			var base *plannedDownload
			for i := range plan {
				if plan[i].IsBase {
					if base == nil {
						base = &plan[i]
					}
					continue
				}
				extraDownloads = append(extraDownloads, gaba.Download{
					URL:         fileContentURL(host, g.ID, plan[i].FileName, plan[i].FileID),
					Location:    plan[i].Location,
					DisplayName: fmt.Sprintf("%s – %s", g.Name, plan[i].FileName),
					Timeout:     config.DownloadTimeout.Duration(),
				})
				// Hide the add-on's folder from ES-DE when it's a subfolder of the
				// ROM directory (not the base folder itself, and not a custom path
				// routed elsewhere — those either must stay visible or aren't scanned).
				if markIgnored {
					dir := filepath.Dir(plan[i].Location)
					if rel, err := filepath.Rel(romDirectory, dir); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
						ignoreDirSet[dir] = true
					}
				}
			}
			if base == nil {
				gaba.GetLogger().Warn("Categorized ROM has no base file; skipping", "game", g.Name, "id", g.ID)
				continue
			}
			downloadLocation = base.Location
			sourceURL = fileContentURL(host, g.ID, base.FileName, base.FileID)
			if !base.Download {
				// Base is already downloaded (or the user unchecked it): keep it as
				// the gamelist/artwork anchor but don't re-fetch it. If there are no
				// add-ons to fetch either, there's nothing to do for this game.
				queuePrimaryDownload = false
				if len(extraDownloads) == 0 {
					continue
				}
			}
		} else if g.HasMultipleFiles {
			tmpDir := fileutil.TempDir()
			downloadLocation = filepath.Join(tmpDir, fmt.Sprintf("grout_multirom_%d.zip", g.ID))
			sourceURL, _ = url.JoinPath(host.URL(), "/api/roms/", strconv.Itoa(g.ID), "content", g.FsName)
		} else {
			// Skip games with no file metadata to avoid an out-of-range panic.
			// This can happen when the cached row was written without a
			// `files` array (e.g. after an incremental cache update).
			if len(g.Files) == 0 {
				gaba.GetLogger().Warn("Skipping ROM with no file metadata; refresh the library to repopulate it",
					"game", g.Name, "id", g.ID, "fs_name", g.FsName)
				continue
			}
			// Find the file to download - use selected file if specified, otherwise first file
			fileToDownload := g.Files[0]
			if selectedFileID > 0 {
				for _, f := range g.Files {
					if f.ID == selectedFileID {
						fileToDownload = f
						break
					}
				}
			}
			downloadLocation = filepath.Join(romDirectory, fileToDownload.FileName)
			sourceURL, _ = url.JoinPath(host.URL(), "/api/roms/", strconv.Itoa(g.ID), "content", fileToDownload.FileName)
			sourceURL += "?" + url.Values{"file_ids": {strconv.Itoa(fileToDownload.ID)}}.Encode()
		}

		gamelistRomEntry.GamePath = downloadLocation

		if queuePrimaryDownload {
			downloads = append(downloads, gaba.Download{
				URL:         sourceURL,
				Location:    downloadLocation,
				DisplayName: g.Name,
				Timeout:     config.DownloadTimeout.Duration(),
			})
		}
		// Queue the selected add-on files (they share the base game's artwork
		// and gamelist entry, so no extra art/metadata handling is needed).
		downloads = append(downloads, extraDownloads...)

		if config.DownloadArt && (g.PathCoverLarge != "" || g.PathCoverSmall != "" || g.URLCover != "") {
			// Prepare download for cover art
			artDir := config.GetArtDirectory(gamePlatform)
			var artFileName string
			if cfw.GetCFW() == cfw.MinUI && len(g.Files) > 0 {
				artFileName = g.Files[0].FileName + ".png"
			} else {
				artFileName = g.FsNameNoExt + ".png"
			}
			artLocation := filepath.Join(artDir, artFileName)
			coverURL := g.GetArtworkURL(config.ArtKind, host)
			gamelistRomEntry.ArtLocation.ImagePath = artLocation

			artDownloads = append(artDownloads, artDownload{
				URL:      coverURL,
				Location: artLocation,
				GameName: g.Name,
				IsImage:  true,
			})

			// Prepare download for additional art types if enabled
			artPreviewDir := config.GetArtPreviewDirectory(gamePlatform)
			if config.DownloadArtScreenshotPreview && artPreviewDir != "" {
				screenshotPreviewLocation := filepath.Join(artPreviewDir, artFileName)
				if screenshotURL := g.GetScreenshotURL(host); screenshotURL != "" {
					artDownloads = append(artDownloads, artDownload{
						URL:      screenshotURL,
						Location: screenshotPreviewLocation,
						GameName: g.Name,
						IsImage:  true,
					})
				}
			}

			artSplashDir := config.GetArtSplashDirectory(gamePlatform)
			if (config.DownloadSplashArt != artutil.ArtKindNone || config.AdditionalDownloads.Thumbnail != artutil.ArtKindNone) && artSplashDir != "" {
				artSplashFileName := g.FsNameNoExt
				isESBased := cfw.GetCFW().IsBasedOnEmulationStation()
				if cfw.GetCFW().UsesArtSuffixes() {
					artSplashFileName += "-thumb.png"
				} else {
					artSplashFileName += ".png"
				}
				splashArtLocation := filepath.Join(artSplashDir, artSplashFileName)
				kind := config.DownloadSplashArt
				if config.AdditionalDownloads.Thumbnail != artutil.ArtKindNone {
					kind = config.AdditionalDownloads.Thumbnail
				}
				if splashURL := g.GetSplashArtURL(kind, host); splashURL != "" {
					if isESBased {
						gamelistRomEntry.ArtLocation.ThumbnailPath = splashArtLocation
					}
					artDownloads = append(artDownloads, artDownload{
						URL:      splashURL,
						Location: splashArtLocation,
						GameName: g.Name,
						IsImage:  true,
					})
				}
			}

			artMarqueeDir := config.GetArtMarqueeDirectory(gamePlatform)
			if config.AdditionalDownloads.Marquee != artutil.ArtKindNone && artMarqueeDir != "" {
				marqueeArtFileName := g.FsNameNoExt
				// Batocera-style ES needs a -marquee suffix to avoid conflicts with cover art
				if cfw.GetCFW().UsesArtSuffixes() {
					marqueeArtFileName += "-marquee.png"
				} else {
					marqueeArtFileName += ".png"
				}
				marqueeArtLocation := filepath.Join(artMarqueeDir, marqueeArtFileName)
				marqueeURL := ""
				switch config.AdditionalDownloads.Marquee {
				case artutil.ArtKindMarquee:
					marqueeURL = g.GetMarqueeURL(host)
				case artutil.ArtKindLogo:
					marqueeURL = g.GetLogoURL(host)
				}
				if marqueeURL != "" {
					gamelistRomEntry.ArtLocation.MarqueePath = marqueeArtLocation
					artDownloads = append(artDownloads, artDownload{
						URL:      marqueeURL,
						Location: marqueeArtLocation,
						GameName: g.Name,
						IsImage:  true,
					})
				}
			}

			artVideoDir := config.GetArtVideoDirectory(gamePlatform)
			if config.AdditionalDownloads.Video && artVideoDir != "" {
				videoLocation := filepath.Join(artVideoDir, g.FsNameNoExt+".mp4")
				if videoURL := g.GetVideoURL(host); videoURL != "" {
					gamelistRomEntry.ArtLocation.VideoPath = videoLocation
					artDownloads = append(artDownloads, artDownload{
						URL:      videoURL,
						Location: videoLocation,
						GameName: g.Name,
						IsImage:  false,
					})
				}
			}

			artBezelDir := config.GetArtBezelDirectory(gamePlatform)
			if config.AdditionalDownloads.Bezel && artBezelDir != "" {
				bezelArtLocation := filepath.Join(artBezelDir, artFileName)
				if bezelURL := g.GetBezelURL(host); bezelURL != "" {
					gamelistRomEntry.ArtLocation.BezelPath = bezelArtLocation
					artDownloads = append(artDownloads, artDownload{
						URL:      bezelURL,
						Location: bezelArtLocation,
						GameName: g.Name,
						IsImage:  true,
					})
				}
			}

			manualDir := config.GetManualDirectory(gamePlatform)
			if config.AdditionalDownloads.Manual && manualDir != "" {
				manualLocation := filepath.Join(manualDir, g.FsNameNoExt+".pdf")
				if manualURL := g.GetManualURL(host); manualURL != "" {
					gamelistRomEntry.ArtLocation.ManualPath = manualLocation
					artDownloads = append(artDownloads, artDownload{
						URL:      manualURL,
						Location: manualLocation,
						GameName: g.Name,
						IsImage:  false,
					})
				}
			}

			boxbackDir := config.GetBoxbackDirectory(gamePlatform)
			if config.AdditionalDownloads.BoxBack && boxbackDir != "" {
				boxbackArtFileName := g.FsNameNoExt
				// Batocera-style ES needs a -boxback suffix to avoid conflicts with cover art
				if cfw.GetCFW().UsesArtSuffixes() {
					boxbackArtFileName += "-boxback.png"
				} else {
					boxbackArtFileName += ".png"
				}
				boxbackArtLocation := filepath.Join(boxbackDir, boxbackArtFileName)
				if boxbackURL := g.GetBoxbackURL(host); boxbackURL != "" {
					gamelistRomEntry.ArtLocation.BoxBackPath = boxbackArtLocation
					artDownloads = append(artDownloads, artDownload{
						URL:      boxbackURL,
						Location: boxbackArtLocation,
						GameName: g.Name,
						IsImage:  true,
					})
				}
			}

			fanartDir := config.GetFanartDirectory(gamePlatform)
			if config.AdditionalDownloads.Fanart && fanartDir != "" {
				fanartFileName := g.FsNameNoExt
				// Batocera-style ES needs a -fanart suffix to avoid conflicts with cover art
				if cfw.GetCFW().UsesArtSuffixes() {
					fanartFileName += "-fanart.png"
				} else {
					fanartFileName += ".png"
				}
				fanartLocation := filepath.Join(fanartDir, fanartFileName)
				if fanartURL := g.GetFanartURL(host); fanartURL != "" {
					gamelistRomEntry.ArtLocation.FanartPath = fanartLocation
					artDownloads = append(artDownloads, artDownload{
						URL:      fanartURL,
						Location: fanartLocation,
						GameName: g.Name,
						IsImage:  true,
					})
				}
			}

		}
		gamesSummaries = append(gamesSummaries, gamelistRomEntry)
	}

	ignoreDirs := make([]string, 0, len(ignoreDirSet))
	for dir := range ignoreDirSet {
		ignoreDirs = append(ignoreDirs, dir)
	}

	return downloads, artDownloads, gamesSummaries, ignoreDirs
}

// resolveExtractedGamePath returns the best path for a multi-file ROM after extraction.
func resolveExtractedGamePath(romDirectory, extractDir, fsNameNoExt string) string {
	logger := gaba.GetLogger()

	m3uPath := filepath.Join(romDirectory, fsNameNoExt+".m3u")
	if _, err := os.Stat(m3uPath); err == nil {
		logger.Debug("Multi-file ROM gamelist path resolved to .m3u in rom directory", "path", m3uPath)
		return m3uPath
	}
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		logger.Debug("Multi-file ROM gamelist path falling back to extract directory (unreadable)", "path", extractDir, "error", err)
		return extractDir
	}
	var firstFile string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(entry.Name())) == ".m3u" {
			p := filepath.Join(extractDir, entry.Name())
			logger.Debug("Multi-file ROM gamelist path resolved to .m3u in extract directory", "path", p)
			return p
		}
		if firstFile == "" {
			firstFile = filepath.Join(extractDir, entry.Name())
		}
	}
	if firstFile != "" {
		logger.Debug("Multi-file ROM gamelist path resolved to first file (no .m3u found)", "path", firstFile)
		return firstFile
	}
	logger.Debug("Multi-file ROM gamelist path falling back to extract directory (no files found)", "path", extractDir)
	return extractDir
}

func (s *DownloadScreen) downloadArt(artDownloads []artDownload, downloadedGames []romm.Rom, headers map[string]string, progress *atomic.Float64, insecureSkipVerify bool) {
	logger := gaba.GetLogger()

	downloadedGameNames := make(map[string]bool)
	for _, g := range downloadedGames {
		downloadedGameNames[g.Name] = true
	}

	totalArt := 0
	for _, art := range artDownloads {
		if downloadedGameNames[art.GameName] {
			totalArt++
		}
	}

	successCount := 0
	failCount := 0
	processedCount := 0

	for _, art := range artDownloads {
		if !downloadedGameNames[art.GameName] {
			continue
		}

		artDir := filepath.Dir(art.Location)
		if err := os.MkdirAll(artDir, 0755); err != nil {
			logger.Warn("Failed to create art directory", "dir", artDir, "game", art.GameName, "error", err)
			failCount++
			processedCount++
			if totalArt > 0 {
				progress.Store(float64(processedCount) / float64(totalArt))
			}
			continue
		}

		req, err := http.NewRequest("GET", art.URL, nil)
		if err != nil {
			logger.Warn("Failed to create art request", "game", art.GameName, "error", err)
			failCount++
			processedCount++
			if totalArt > 0 {
				progress.Store(float64(processedCount) / float64(totalArt))
			}
			continue
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		client := &http.Client{Timeout: romm.DefaultClientTimeout}
		if insecureSkipVerify {
			client.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
		}
		resp, err := client.Do(req)
		if err != nil {
			logger.Warn("Failed to download art", "game", art.GameName, "url", art.URL, "error", err)
			failCount++
			processedCount++
			if totalArt > 0 {
				progress.Store(float64(processedCount) / float64(totalArt))
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			logger.Warn("Art download failed with bad status", "game", art.GameName, "url", art.URL, "status", resp.Status)
			failCount++
			processedCount++
			if totalArt > 0 {
				progress.Store(float64(processedCount) / float64(totalArt))
			}
			continue
		}

		outFile, err := os.Create(art.Location)
		if err != nil {
			resp.Body.Close()
			logger.Warn("Failed to create art file", "game", art.GameName, "location", art.Location, "error", err)
			failCount++
			processedCount++
			if totalArt > 0 {
				progress.Store(float64(processedCount) / float64(totalArt))
			}
			continue
		}

		_, err = io.Copy(outFile, resp.Body)
		resp.Body.Close()
		outFile.Close()

		if err != nil {
			logger.Warn("Failed to write art file", "game", art.GameName, "location", art.Location, "error", err)
			os.Remove(art.Location)
			failCount++
			processedCount++
			if totalArt > 0 {
				progress.Store(float64(processedCount) / float64(totalArt))
			}
			continue
		}

		if art.IsImage {
			if err := imageutil.ProcessArtImage(art.Location); err != nil {
				logger.Warn("Failed to process art image", "game", art.GameName, "location", art.Location, "error", err, "url", art.URL)
				os.Remove(art.Location)
				failCount++
				processedCount++
				if totalArt > 0 {
					progress.Store(float64(processedCount) / float64(totalArt))
				}
				continue
			}
		}

		successCount++

		processedCount++
		if totalArt > 0 {
			progress.Store(float64(processedCount) / float64(totalArt))
		}
	}
}
