package romm

import (
	"regexp"
	"strconv"
	"time"
)

// UpdateStrategy describes how a platform's "update" add-ons relate to one
// another, which decides how many of them a download needs to fetch.
type UpdateStrategy int

const (
	// UpdatesIndependent keeps every update. This is the safe default for any
	// platform whose update semantics aren't known: without proof that a newer
	// update contains the older ones, we can't drop any.
	UpdatesIndependent UpdateStrategy = iota
	// UpdatesCumulative means each update is a full build that supersedes all
	// earlier ones, so only the newest is needed (e.g. Nintendo Switch title
	// updates). DLC is always independent regardless of this setting.
	UpdatesCumulative
)

// cumulativeUpdatePlatforms is the skeleton for per-system update handling. Only
// platforms proven to ship cumulative, self-contained updates belong here; every
// other platform falls through to UpdatesIndependent and keeps all updates.
//
// Today only Switch is implemented — its updates are full builds that include all
// prior updates. Other systems (PS4/PS5, Vita, Wii U, …) need their own update
// semantics confirmed before being added, and may eventually be driven by user
// config rather than this fixed table.
var cumulativeUpdatePlatforms = map[string]bool{
	"switch": true,
}

// UpdateStrategyForPlatform returns how the given platform's updates should be
// treated. Unknown platforms are UpdatesIndependent.
func UpdateStrategyForPlatform(fsSlug string) UpdateStrategy {
	if cumulativeUpdatePlatforms[fsSlug] {
		return UpdatesCumulative
	}
	return UpdatesIndependent
}

// UpdatesCumulative reports whether this ROM's platform ships cumulative updates,
// so only the latest update file is needed. When false, all updates are kept.
func (r Rom) UpdatesCumulative() bool {
	return UpdateStrategyForPlatform(r.PlatformFSSlug) == UpdatesCumulative
}

// updateVersionTokenRe matches a bracketed number in a filename, capturing an
// optional leading "v". Switch update names carry a monotonic version integer,
// usually "[v131072]" but sometimes with the "v" omitted ("[131072]").
var updateVersionTokenRe = regexp.MustCompile(`(?i)[\[(]\s*(v)?\s*(\d+)\s*[\])]`)

// switchTitleIDDigits is the length of a Switch title ID (16 hex chars). When a
// title ID happens to be all decimal digits (e.g. "0100000000010800") it looks
// like a bracketed number, so a bare 16-digit token is treated as a title ID, not
// a version. Title IDs containing hex letters never match the decimal token regex.
const switchTitleIDDigits = 16

// switchVersionGranularity is the step between consecutive Switch title-update
// versions (0x10000). Retail update version IDs are always multiples of it, which
// tells a real version apart from a human-readable one that lost its dots (e.g.
// "[1]", "[20]") when the "v" prefix is also absent.
const switchVersionGranularity = 65536

// parseUpdateVersion extracts the internal update version integer from a filename.
// A single name can carry several bracketed tokens — the 16-char title ID, the
// internal version ("[v131072]" or, with the "v" dropped, "[131072]"), and a
// human-readable version ("[1.20]", "[2.0.0]"). It resolves them as:
//   - an explicit "v"-prefixed number is unambiguous and always wins;
//   - otherwise a bare bracketed number is accepted only when it looks like a real
//     Switch version: a nonzero multiple of the version granularity that isn't a
//     16-digit title ID. Dotted human versions never match the digits-only regex.
//
// The bool is false when no usable version token is present (callers then fall
// back to file timestamps).
func parseUpdateVersion(fileName string) (uint64, bool) {
	matches := updateVersionTokenRe.FindAllStringSubmatch(fileName, -1)
	if matches == nil {
		return 0, false
	}
	// An explicit "v" prefix is unambiguous — use it wherever it appears.
	for i := len(matches) - 1; i >= 0; i-- {
		if matches[i][1] != "" {
			if n, err := strconv.ParseUint(matches[i][2], 10, 64); err == nil {
				return n, true
			}
		}
	}
	// No "v" anywhere: accept only a bare number shaped like a real Switch version.
	// Scan from the end since the version conventionally trails the title ID.
	for i := len(matches) - 1; i >= 0; i-- {
		digits := matches[i][2]
		if len(digits) == switchTitleIDDigits {
			continue // title ID, not a version
		}
		n, err := strconv.ParseUint(digits, 10, 64)
		if err != nil || n == 0 || n%switchVersionGranularity != 0 {
			continue // human-readable or otherwise not a Switch version ID
		}
		return n, true
	}
	return 0, false
}

// fileTimestamp is the best available "when did this file change" signal, used as
// a fallback for ordering updates whose filenames carry no version token.
func fileTimestamp(f RomFile) time.Time {
	if !f.LastModified.IsZero() {
		return f.LastModified
	}
	if !f.UpdatedAt.IsZero() {
		return f.UpdatedAt
	}
	return f.CreatedAt
}

// updateFileNewer reports whether update file a is newer than b. A parsed version
// token is authoritative and outranks an unversioned file; among versioned files
// the higher number wins; otherwise (and to break ties) the newer timestamp wins.
func updateFileNewer(a, b RomFile) bool {
	av, aok := parseUpdateVersion(a.FileName)
	bv, bok := parseUpdateVersion(b.FileName)
	if aok != bok {
		return aok
	}
	if aok && bok && av != bv {
		return av > bv
	}
	return fileTimestamp(a).After(fileTimestamp(b))
}

// LatestUpdateFile returns the newest update-category file for the ROM. The bool is
// false when the ROM has no update files. It always computes the latest regardless
// of platform strategy; callers decide whether to act on it (see UpdatesCumulative).
func (r Rom) LatestUpdateFile() (RomFile, bool) {
	var best RomFile
	found := false
	for _, f := range r.Files {
		if f.Category != RomFileUpdate {
			continue
		}
		if !found || updateFileNewer(f, best) {
			best, found = f, true
		}
	}
	return best, found
}
