// Package actionpins provides action pin resolution for GitHub Actions,
// mapping repository references to their pinned commit SHAs.
// It is intentionally free of dependencies on pkg/workflow so it can be
// imported by any package without introducing import cycles.
package actionpins

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/semverutil"
)

var log = logger.New("workflow:action_pins")

//go:embed data/action_pins.json
var actionPinsJSON []byte

// ActionYAMLInput holds an input definition parsed from a GitHub Action's action.yml.
type ActionYAMLInput struct {
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty"    json:"required,omitempty"`
	Default     string `yaml:"default,omitempty"     json:"default,omitempty"`
}

// ActionPin represents a pinned GitHub Action with its commit SHA.
type ActionPin struct {
	Repo    string                      `json:"repo"`
	Version string                      `json:"version"`
	SHA     string                      `json:"sha"`
	Inputs  map[string]*ActionYAMLInput `json:"inputs,omitempty"`
}

// ActionPinsData represents the structure of the embedded JSON file.
type ActionPinsData struct {
	Entries map[string]ActionPin `json:"entries"`
}

// SHAResolver resolves a GitHub Action's commit SHA for a given version tag.
type SHAResolver interface {
	ResolveSHA(repo, version string) (string, error)
}

// PinContext provides the runtime context needed for action pin resolution.
// Callers construct one from their own state (e.g. WorkflowData fields).
// The Warnings map is mutated in place to deduplicate warning output.
type PinContext struct {
	// Resolver resolves SHAs dynamically via GitHub CLI. May be nil.
	Resolver SHAResolver
	// StrictMode controls how resolution failures are handled.
	StrictMode bool
	// Warnings is a shared map for deduplicating warning messages.
	// Keys are cache keys in the form "repo@version".
	Warnings map[string]bool
}

var (
	cachedActionPins       []ActionPin
	cachedActionPinsByRepo map[string][]ActionPin
	actionPinsOnce         sync.Once
)

// GetActionPins returns all loaded action pins sorted by version descending.
func GetActionPins() []ActionPin {
	return getActionPins()
}

func getActionPins() []ActionPin {
	actionPinsOnce.Do(func() {
		log.Print("Unmarshaling action pins from embedded JSON (first call, will be cached)")

		var data ActionPinsData
		if err := json.Unmarshal(actionPinsJSON, &data); err != nil {
			log.Printf("Failed to unmarshal action pins JSON: %v", err)
			panic(fmt.Sprintf("failed to load action pins: %v", err))
		}

		mismatchCount := 0
		for key, pin := range data.Entries {
			if idx := strings.LastIndex(key, "@"); idx != -1 {
				keyVersion := key[idx+1:]
				if keyVersion != pin.Version {
					mismatchCount++
					shortSHA := pin.SHA
					if len(pin.SHA) > 8 {
						shortSHA = pin.SHA[:8]
					}
					log.Printf("WARNING: Key/version mismatch in action_pins.json: key=%s has version=%s but pin.Version=%s (sha=%s)",
						key, keyVersion, pin.Version, shortSHA)
				}
			}
		}
		if mismatchCount > 0 {
			log.Printf("Found %d key/version mismatches in action_pins.json", mismatchCount)
		}

		pins := make([]ActionPin, 0, len(data.Entries))
		for _, pin := range data.Entries {
			pins = append(pins, pin)
		}

		sort.Slice(pins, func(i, j int) bool {
			if pins[i].Version != pins[j].Version {
				return pins[i].Version > pins[j].Version
			}
			return pins[i].Repo < pins[j].Repo
		})

		log.Printf("Successfully unmarshaled and sorted %d action pins from JSON", len(pins))
		cachedActionPins = pins

		byRepo := make(map[string][]ActionPin, len(pins))
		for _, pin := range pins {
			byRepo[pin.Repo] = append(byRepo[pin.Repo], pin)
		}
		for repo, repoPins := range byRepo {
			sort.Slice(repoPins, func(i, j int) bool {
				v1 := strings.TrimPrefix(repoPins[i].Version, "v")
				v2 := strings.TrimPrefix(repoPins[j].Version, "v")
				return semverutil.Compare(v1, v2) > 0
			})
			byRepo[repo] = repoPins
		}
		cachedActionPinsByRepo = byRepo
		log.Printf("Built per-repo action pin index for %d repos", len(byRepo))
	})

	return cachedActionPins
}

// GetActionPinsByRepo returns the sorted (version-descending) list of action pins
// for the given repository. Returns nil if the repo has no pins.
func GetActionPinsByRepo(repo string) []ActionPin {
	getActionPins()
	return cachedActionPinsByRepo[repo]
}

// GetActionPinByRepo returns the latest ActionPin for a given repository, if any.
func GetActionPinByRepo(repo string) (ActionPin, bool) {
	pins := GetActionPinsByRepo(repo)
	if len(pins) == 0 {
		return ActionPin{}, false
	}
	return pins[0], true
}

// getActionPin returns the pinned reference for the latest version of the repo.
// Returns an empty string if no pin is found.
func getActionPin(repo string) string {
	pins := GetActionPinsByRepo(repo)
	if len(pins) == 0 {
		return ""
	}
	return FormatReference(repo, pins[0].SHA, pins[0].Version)
}

// FormatReference formats an action reference with repo, SHA, and version comment.
// Example: "actions/checkout@abc123 # v4.1.0"
func FormatReference(repo, sha, version string) string {
	return fmt.Sprintf("%s@%s # %s", repo, sha, version)
}

// FormatCacheKey generates a cache key for action resolution.
// Example: "actions/checkout@v4"
func FormatCacheKey(repo, version string) string {
	return fmt.Sprintf("%s@%s", repo, version)
}

// ExtractRepo extracts the action repository from a uses string.
// Examples: "actions/checkout@v5" -> "actions/checkout"
func ExtractRepo(uses string) string {
	before, _, ok := strings.Cut(uses, "@")
	if !ok {
		return uses
	}
	return before
}

// ExtractVersion extracts the version from a uses string.
// Examples: "actions/checkout@v5" -> "v5", "actions/checkout" -> ""
func ExtractVersion(uses string) string {
	_, after, ok := strings.Cut(uses, "@")
	if !ok {
		return ""
	}
	return after
}

// isValidFullSHA checks if a string is a valid 40-character hexadecimal SHA.
func isValidFullSHA(s string) bool {
	return gitutil.IsValidFullSHA(s)
}

// findCompatiblePin returns the first pin whose version is semver-compatible with
// the requested version, or ActionPin{}, false if no compatible pin is found.
func findCompatiblePin(pins []ActionPin, version string) (ActionPin, bool) {
	for _, pin := range pins {
		if semverutil.IsCompatible(pin.Version, version) {
			return pin, true
		}
	}
	return ActionPin{}, false
}

// ResolveActionPin returns the pinned action reference for a given action@version.
// It consults ctx.Resolver first, then falls back to embedded pins.
// If ctx is nil, only embedded pins are consulted.
func ResolveActionPin(actionRepo, version string, ctx *PinContext) (string, error) {
	if ctx == nil {
		ctx = &PinContext{}
	}
	log.Printf("Resolving action pin: repo=%s, version=%s, strict_mode=%t", actionRepo, version, ctx.StrictMode)

	isAlreadySHA := isValidFullSHA(version)

	if ctx.Resolver != nil && !isAlreadySHA {
		log.Printf("Attempting dynamic resolution for %s@%s", actionRepo, version)
		sha, err := ctx.Resolver.ResolveSHA(actionRepo, version)
		if err == nil && sha != "" {
			log.Printf("Dynamic resolution succeeded: %s@%s → %s", actionRepo, version, sha)
			result := FormatReference(actionRepo, sha, version)
			log.Printf("Returning pinned reference: %s", result)
			return result, nil
		}
		log.Printf("Dynamic resolution failed for %s@%s: %v", actionRepo, version, err)
	} else {
		if isAlreadySHA {
			log.Printf("Version is already a SHA, skipping dynamic resolution")
		} else {
			log.Printf("No action resolver available, skipping dynamic resolution")
		}
	}

	log.Printf("Falling back to hardcoded pins for %s@%s", actionRepo, version)
	matchingPins := GetActionPinsByRepo(actionRepo)

	if len(matchingPins) == 0 {
		log.Printf("No hardcoded pins found for %s", actionRepo)
	} else {
		log.Printf("Found %d hardcoded pin(s) for %s", len(matchingPins), actionRepo)

		for _, pin := range matchingPins {
			if pin.Version == version {
				log.Printf("Exact version match: requested=%s, found=%s", version, pin.Version)
				return FormatReference(actionRepo, pin.SHA, pin.Version), nil
			}
		}

		if isAlreadySHA {
			for _, pin := range matchingPins {
				if pin.SHA == version {
					log.Printf("Exact SHA match: requested=%s, found version=%s", version, pin.Version)
					return FormatReference(actionRepo, pin.SHA, pin.Version), nil
				}
			}
			log.Printf("SHA %s not found in hardcoded pins, returning as-is", version)
			return FormatReference(actionRepo, version, version), nil
		}

		if !ctx.StrictMode && len(matchingPins) > 0 {
			selectedPin, foundCompatible := findCompatiblePin(matchingPins, version)
			if foundCompatible {
				log.Printf("No exact match for version %s, using highest semver-compatible version: %s", version, selectedPin.Version)
			} else {
				selectedPin = matchingPins[0]
				log.Printf("No exact match for version %s, no semver-compatible versions found, using highest available: %s", version, selectedPin.Version)
			}

			if !isAlreadySHA {
				if ctx.Warnings == nil {
					ctx.Warnings = make(map[string]bool)
				}
				cacheKey := FormatCacheKey(actionRepo, version)
				if !ctx.Warnings[cacheKey] {
					warningMsg := fmt.Sprintf("Unable to resolve %s@%s dynamically, using hardcoded pin for %s@%s",
						actionRepo, version, actionRepo, selectedPin.Version)
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMsg))
					ctx.Warnings[cacheKey] = true
				}
			}
			log.Printf("Using version in non-strict mode: %s@%s (requested) → %s@%s (used)",
				actionRepo, version, actionRepo, selectedPin.Version)
			return FormatReference(actionRepo, selectedPin.SHA, version), nil
		}
	}

	if isAlreadySHA {
		log.Printf("SHA %s not found in hardcoded pins, returning as-is", version)
		return FormatReference(actionRepo, version, version), nil
	}

	if ctx.Warnings == nil {
		ctx.Warnings = make(map[string]bool)
	}
	cacheKey := FormatCacheKey(actionRepo, version)
	if !ctx.Warnings[cacheKey] {
		warningMsg := fmt.Sprintf("Unable to pin action %s@%s", actionRepo, version)
		if ctx.Resolver != nil {
			warningMsg = fmt.Sprintf("Unable to pin action %s@%s: resolution failed", actionRepo, version)
		}
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMsg))
		ctx.Warnings[cacheKey] = true
	}
	return "", nil
}

// GetCachedActionPin returns the pinned action reference for a given repository,
// preferring the user's cache (via ctx.Resolver) over the embedded action_pins.json.
// If ctx is nil, only embedded pins are consulted.
func GetCachedActionPin(repo string, ctx *PinContext) string {
	if ctx == nil {
		return getActionPin(repo)
	}

	pins := GetActionPinsByRepo(repo)
	if len(pins) == 0 {
		return getActionPin(repo)
	}

	latestVersion := pins[0].Version
	pinnedRef, err := ResolveActionPin(repo, latestVersion, ctx)
	if err != nil || pinnedRef == "" {
		return getActionPin(repo)
	}
	return pinnedRef
}
