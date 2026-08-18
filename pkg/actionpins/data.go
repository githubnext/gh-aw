package actionpins

import (
	"cmp"
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/semverutil"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var actionPinsLog = logger.New("actionpins:actionpins")

//go:embed data/action_pins.json
var actionPinsJSON []byte

var (
	cachedActionPins       []ActionPin
	cachedActionPinsByRepo map[string][]ActionPin
	cachedContainerPins    map[string]ContainerPin
	actionPinsOnce         sync.Once
)

func getActionPins() []ActionPin {
	actionPinsOnce.Do(func() {
		actionPinsLog.Print("Unmarshaling action pins from embedded JSON (first call, will be cached)")

		data := loadActionPinsData(actionPinsJSON)

		pins := slices.Collect(maps.Values(data.Entries))

		slices.SortFunc(pins, func(a, b ActionPin) int {
			if a.Version != b.Version {
				return cmp.Compare(b.Version, a.Version) // descending by version
			}
			return cmp.Compare(a.Repo, b.Repo)
		})

		actionPinsLog.Printf("Successfully unmarshaled and sorted %d action pins from JSON", len(pins))
		cachedActionPins = pins

		cachedActionPinsByRepo = buildByRepoIndex(pins)
		actionPinsLog.Printf("Built per-repo action pin index for %d repos", len(cachedActionPinsByRepo))

		cachedContainerPins = data.Containers
		if cachedContainerPins == nil {
			cachedContainerPins = make(map[string]ContainerPin)
		}
		actionPinsLog.Printf("Loaded %d container pins from JSON", len(cachedContainerPins))
	})

	return cachedActionPins
}

// loadActionPinsData unmarshals embedded action pin data.
// Panics if the embedded JSON is invalid or any entry has an empty SHA, because
// those conditions indicate corrupted release data that would produce invalid workflow YAML.
func loadActionPinsData(raw []byte) ActionPinsData {
	var data ActionPinsData
	if err := json.Unmarshal(raw, &data); err != nil {
		actionPinsLog.Printf("Failed to unmarshal action pins JSON: %v", err)
		panic(fmt.Sprintf("failed to load action pins: %v", err))
	}

	if n := countPinKeyMismatches(data.Entries); n > 0 {
		actionPinsLog.Printf("Found %d key/version mismatches in action_pins.json", n)
	}

	if emptyKeys := collectEntriesWithEmptySHA(data.Entries); len(emptyKeys) > 0 {
		panic(fmt.Sprintf("action_pins.json has %d entries with empty SHA %v — these would produce invalid workflow YAML (e.g. 'owner/repo@ # version'); remove or fix these entries before releasing", len(emptyKeys), emptyKeys))
	}

	return data
}

// countPinKeyMismatches returns the number of entries where the key version does not
// match pin.Version, logging each mismatch for diagnostics.
func countPinKeyMismatches(entries map[string]ActionPin) int {
	count := 0
	for key, pin := range entries {
		if idx := strings.LastIndex(key, "@"); idx != -1 {
			keyVersion := key[idx+1:]
			if keyVersion != pin.Version {
				count++
				shortSHA := pin.SHA[:min(len(pin.SHA), 8)]
				actionPinsLog.Printf("WARNING: Key/version mismatch in action_pins.json: key=%s has version=%s but pin.Version=%s (sha=%s)",
					key, keyVersion, pin.Version, shortSHA)
			}
		}
	}
	return count
}

// collectEntriesWithEmptySHA returns the sorted keys of entries whose SHA field is empty,
// logging each offending entry for diagnostics.
func collectEntriesWithEmptySHA(entries map[string]ActionPin) []string {
	keys := sliceutil.FilterMapKeys(entries, func(_ string, pin ActionPin) bool {
		return pin.SHA == ""
	})
	for _, key := range keys {
		pin := entries[key]
		actionPinsLog.Printf("ERROR: Empty SHA in action_pins.json: key=%s repo=%s version=%s", key, pin.Repo, pin.Version)
	}
	slices.Sort(keys)
	return keys
}

// buildByRepoIndex groups pins by repository and sorts each group by version descending.
func buildByRepoIndex(pins []ActionPin) map[string][]ActionPin {
	byRepo := make(map[string][]ActionPin, len(pins))
	for _, pin := range pins {
		byRepo[pin.Repo] = append(byRepo[pin.Repo], pin)
	}
	for _, repoPins := range byRepo {
		slices.SortFunc(repoPins, func(a, b ActionPin) int {
			v1 := strings.TrimPrefix(a.Version, "v")
			v2 := strings.TrimPrefix(b.Version, "v")
			return semverutil.Compare(v2, v1) // descending by semver
		})
	}
	return byRepo
}

// GetActionPinsByRepo returns the sorted (version-descending) list of action pins
// for the given repository. Returns nil if the repo has no pins.
func GetActionPinsByRepo(repo string) []ActionPin {
	getActionPins()
	return cachedActionPinsByRepo[repo]
}

// GetLatestActionPinByRepo returns the latest ActionPin for a given repository, if any.
func GetLatestActionPinByRepo(repo string) (ActionPin, bool) {
	pins := GetActionPinsByRepo(repo)
	if len(pins) == 0 {
		return ActionPin{}, false
	}
	return pins[0], true
}

// GetContainerPin returns a pinned container image by its original image reference.
func GetContainerPin(image string) (ContainerPin, bool) {
	getActionPins()
	pin, ok := cachedContainerPins[image]
	return pin, ok
}
