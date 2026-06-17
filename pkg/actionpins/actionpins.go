// Package actionpins provides action pin resolution for GitHub Actions,
// mapping repository references to their pinned commit SHAs.
// It is intentionally free of dependencies on pkg/workflow so it can be
// imported by any package without introducing import cycles.
package actionpins

import (
	"cmp"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/semverutil"
)

var actionPinsLog = logger.New("actionpins:actionpins")

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

// ContainerPin represents a pinned container image reference.
type ContainerPin struct {
	Image       string `json:"image"`
	Digest      string `json:"digest"`
	PinnedImage string `json:"pinned_image"`
}

// ActionPinsData represents the structure of the embedded JSON file.
type ActionPinsData struct {
	Entries    map[string]ActionPin    `json:"entries"`
	Containers map[string]ContainerPin `json:"containers,omitempty"`
}

// SHAResolver resolves a GitHub Action's commit SHA for a given version tag.
type SHAResolver interface {
	ResolveSHA(ctx context.Context, repo, version string) (string, error)
}

// ResolutionErrorType classifies unresolved action-ref pinning outcomes for auditing.
type ResolutionErrorType string

const (
	// ResolutionErrorTypeDynamicResolutionFailed indicates dynamic tag/ref -> SHA resolution failed.
	ResolutionErrorTypeDynamicResolutionFailed ResolutionErrorType = "dynamic_resolution_failed"
	// ResolutionErrorTypePinNotFound indicates no usable hardcoded pin was found for the ref.
	ResolutionErrorTypePinNotFound ResolutionErrorType = "pin_not_found"
)

// ResolutionFailure captures an unresolved action-ref pinning event.
type ResolutionFailure struct {
	Repo      string
	Ref       string
	ErrorType ResolutionErrorType
}

// PinContext provides the runtime context needed for action pin resolution.
// Callers construct one from their own state (e.g. WorkflowData fields).
// The Warnings map is mutated in place to deduplicate warning output.
type PinContext struct {
	// Ctx is the context to propagate into dynamic SHA resolution calls.
	// When nil, context.Background() is used as a fallback.
	Ctx context.Context
	// Resolver resolves SHAs dynamically via GitHub CLI. May be nil.
	Resolver SHAResolver
	// StrictMode controls how resolution failures are handled.
	StrictMode bool
	// EnforcePinned requires unresolved refs to fail unless AllowActionRefs is true.
	EnforcePinned bool
	// AllowActionRefs lowers unresolved pinning failures to warnings.
	// When false, unresolved action refs return an error.
	AllowActionRefs bool
	// Warnings is a shared map for deduplicating warning messages.
	// Keys are cache keys in the form "repo@version".
	Warnings map[string]bool
	// RecordResolutionFailure receives unresolved pinning failures for auditing.
	RecordResolutionFailure func(f ResolutionFailure)
	// SkipHardcodedFallback skips the entire hardcoded-pin lookup (both exact/strict
	// and non-strict matches) when dynamic resolution fails. Set this when GH_HOST is
	// configured to a non-github.com host: the dynamic resolver will query the wrong
	// host and fail, so silently falling back to bundled pins would produce unverified
	// SHA pins and mask the real misconfiguration.
	SkipHardcodedFallback bool
}

var (
	cachedActionPins       []ActionPin
	cachedActionPinsByRepo map[string][]ActionPin
	cachedContainerPins    map[string]ContainerPin
	actionPinsOnce         sync.Once
)

func getActionPins() []ActionPin {
	actionPinsOnce.Do(func() {
		actionPinsLog.Print("Unmarshaling action pins from embedded JSON (first call, will be cached)")

		var data ActionPinsData
		if err := json.Unmarshal(actionPinsJSON, &data); err != nil {
			actionPinsLog.Printf("Failed to unmarshal action pins JSON: %v", err)
			panic(fmt.Sprintf("failed to load action pins: %v", err))
		}

		if n := countPinKeyMismatches(data.Entries); n > 0 {
			actionPinsLog.Printf("Found %d key/version mismatches in action_pins.json", n)
		}

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

// getLatestActionPinReference returns the pinned reference for the latest version of the repo.
// Returns an empty string if no pin is found.
func getLatestActionPinReference(repo string) string {
	pins := GetActionPinsByRepo(repo)
	if len(pins) == 0 {
		return ""
	}
	return FormatPinnedActionReference(repo, pins[0].SHA, pins[0].Version)
}

// FormatPinnedActionReference formats a pinned action reference with repo, SHA, and version comment.
// Example: "actions/checkout@abc123 # v4.1.0"
func FormatPinnedActionReference(repo, sha, version string) string {
	return repo + "@" + sha + " # " + version
}

func formatPinnedActionWithResolution(repo, sha, sourceVersion, resolvedVersion string) string {
	if sourceVersion == resolvedVersion || resolvedVersion == "" {
		return FormatPinnedActionReference(repo, sha, sourceVersion)
	}

	return FormatPinnedActionReference(repo, sha, resolvedVersion+" (source "+sourceVersion+")")
}

// FormatCacheKey generates a cache key for action resolution.
// Example: "actions/checkout@v4"
func FormatCacheKey(repo, version string) string {
	return repo + "@" + version
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

// findVersionBySHA returns the version string for a given SHA in the embedded pins
// for the specified repo. Returns "" if no matching pin is found.
func findVersionBySHA(repo, sha string) string {
	for _, pin := range GetActionPinsByRepo(repo) {
		if pin.SHA == sha {
			return pin.Version
		}
	}
	return ""
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

// initWarnings ensures ctx.Warnings is initialized, avoiding nil map writes.
func initWarnings(ctx *PinContext) {
	if ctx.Warnings == nil {
		ctx.Warnings = make(map[string]bool)
	}
}

// recordPinResolutionFailure silently records an unresolved action-ref pinning event
// to the audit callback (ctx.RecordResolutionFailure), if one is configured.
// If ctx is nil or ctx.RecordResolutionFailure is nil, the function returns early without recording.
func recordPinResolutionFailure(ctx *PinContext, actionRepo, version string, errorType ResolutionErrorType) {
	if ctx == nil || ctx.RecordResolutionFailure == nil {
		return
	}
	ctx.RecordResolutionFailure(ResolutionFailure{
		Repo:      actionRepo,
		Ref:       version,
		ErrorType: errorType,
	})
}

// ResolveActionPin returns the pinned action reference for a given action@version.
// It consults ctx.Resolver first, then falls back to embedded pins.
// If ctx is nil, only embedded pins are consulted.
func ResolveActionPin(actionRepo, version string, ctx *PinContext) (string, error) {
	if ctx == nil {
		ctx = &PinContext{}
	}
	actionPinsLog.Printf("Resolving action pin: repo=%s, version=%s, strict_mode=%t", actionRepo, version, ctx.StrictMode)

	isAlreadySHA := gitutil.IsValidFullSHA(version)
	if pinnedRef, ok := resolveActionPinDynamically(actionRepo, version, isAlreadySHA, ctx); ok {
		return pinnedRef, nil
	}

	if pinnedRef, ok := resolveActionPinFromHardcodedPins(actionRepo, version, isAlreadySHA, ctx); ok {
		return pinnedRef, nil
	}

	if isAlreadySHA {
		actionPinsLog.Printf("SHA %s not found in hardcoded pins, returning as-is", version)
		return FormatPinnedActionReference(actionRepo, version, version), nil
	}

	initWarnings(ctx)
	cacheKey := FormatCacheKey(actionRepo, version)
	errorType := ResolutionErrorTypePinNotFound
	if ctx.Resolver != nil {
		errorType = ResolutionErrorTypeDynamicResolutionFailed
	}
	recordPinResolutionFailure(ctx, actionRepo, version, errorType)
	if ctx.EnforcePinned && !ctx.AllowActionRefs {
		if ctx.Resolver != nil {
			return "", fmt.Errorf("unable to pin action %s@%s: resolution failed", actionRepo, version)
		}
		return "", fmt.Errorf("unable to pin action %s@%s", actionRepo, version)
	}

	if !ctx.Warnings[cacheKey] {
		warningMsg := fmt.Sprintf("Unable to pin action %s@%s", actionRepo, version)
		if ctx.Resolver != nil {
			warningMsg += ": resolution failed"
		}
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMsg))
		ctx.Warnings[cacheKey] = true
	}
	return "", nil
}

func resolveActionPinDynamically(actionRepo, version string, isAlreadySHA bool, ctx *PinContext) (string, bool) {
	if ctx.Resolver == nil || isAlreadySHA {
		logDynamicResolutionSkipped(ctx.Resolver != nil, isAlreadySHA)
		return "", false
	}

	actionPinsLog.Printf("Attempting dynamic resolution for %s@%s", actionRepo, version)
	sha, err := ctx.Resolver.ResolveSHA(cmp.Or(ctx.Ctx, context.Background()), actionRepo, version)
	if err == nil && sha != "" {
		actionPinsLog.Printf("Dynamic resolution succeeded: %s@%s → %s", actionRepo, version, sha)
		resolvedVersion := findVersionBySHA(actionRepo, sha)
		result := formatPinnedActionWithResolution(actionRepo, sha, version, resolvedVersion)
		actionPinsLog.Printf("Returning pinned reference: %s", result)
		return result, true
	}

	actionPinsLog.Printf("Dynamic resolution failed for %s@%s: %v", actionRepo, version, err)
	return "", false
}

func logDynamicResolutionSkipped(hasResolver, isAlreadySHA bool) {
	if isAlreadySHA {
		actionPinsLog.Printf("Version is already a SHA, skipping dynamic resolution")
		return
	}
	if !hasResolver {
		actionPinsLog.Printf("No action resolver available, skipping dynamic resolution")
	}
}

func resolveActionPinFromHardcodedPins(actionRepo, version string, isAlreadySHA bool, ctx *PinContext) (string, bool) {
	// When the caller is targeting a non-github.com host (e.g. GHES/GHEC), the
	// dynamic resolver already failed because it queried the wrong host.  Silently
	// falling back to bundled pins in that case produces unverified SHA pins and
	// masks the real problem, so skip this fallback entirely.
	if ctx.SkipHardcodedFallback {
		actionPinsLog.Printf("SkipHardcodedFallback set, skipping hardcoded pin lookup for %s@%s", actionRepo, version)
		return "", false
	}

	actionPinsLog.Printf("Falling back to hardcoded pins for %s@%s", actionRepo, version)

	matchingPins := GetActionPinsByRepo(actionRepo)
	if len(matchingPins) == 0 {
		actionPinsLog.Printf("No hardcoded pins found for %s", actionRepo)
		return "", false
	}

	actionPinsLog.Printf("Found %d hardcoded pin(s) for %s", len(matchingPins), actionRepo)
	if pinnedRef, ok := resolveExactHardcodedPin(actionRepo, version, isAlreadySHA, matchingPins); ok {
		return pinnedRef, true
	}
	if isAlreadySHA || ctx.StrictMode {
		return "", false
	}
	return resolveNonStrictHardcodedPin(actionRepo, version, matchingPins, ctx), true
}

func resolveExactHardcodedPin(actionRepo, version string, isAlreadySHA bool, matchingPins []ActionPin) (string, bool) {
	for _, pin := range matchingPins {
		if pin.Version == version {
			actionPinsLog.Printf("Exact version match: requested=%s, found=%s", version, pin.Version)
			return FormatPinnedActionReference(actionRepo, pin.SHA, pin.Version), true
		}
	}
	if !isAlreadySHA {
		return "", false
	}
	for _, pin := range matchingPins {
		if pin.SHA == version {
			actionPinsLog.Printf("Exact SHA match: requested=%s, found version=%s", version, pin.Version)
			return FormatPinnedActionReference(actionRepo, pin.SHA, pin.Version), true
		}
	}
	return "", false
}

func resolveNonStrictHardcodedPin(actionRepo, version string, matchingPins []ActionPin, ctx *PinContext) string {
	selectedPin, foundCompatible := findCompatiblePin(matchingPins, version)
	if foundCompatible {
		actionPinsLog.Printf("No exact match for version %s, using highest semver-compatible version: %s", version, selectedPin.Version)
	} else {
		selectedPin = matchingPins[0]
		actionPinsLog.Printf("No exact match for version %s, no semver-compatible versions found, using highest available: %s", version, selectedPin.Version)
	}

	initWarnings(ctx)
	cacheKey := FormatCacheKey(actionRepo, version)
	if !ctx.Warnings[cacheKey] {
		warningMsg := fmt.Sprintf("Unable to resolve %s@%s dynamically, using hardcoded pin for %s@%s",
			actionRepo, version, actionRepo, selectedPin.Version)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMsg))
		ctx.Warnings[cacheKey] = true
	}

	actionPinsLog.Printf("Using version in non-strict mode: %s@%s (requested) → %s@%s (used)",
		actionRepo, version, actionRepo, selectedPin.Version)
	return formatPinnedActionWithResolution(actionRepo, selectedPin.SHA, version, selectedPin.Version)
}

// ResolveLatestActionPin returns the pinned action reference for a given repository,
// preferring the user's cache (via ctx.Resolver) over the embedded action_pins.json.
// If ctx is nil, only embedded pins are consulted.
func ResolveLatestActionPin(repo string, ctx *PinContext) string {
	if ctx == nil {
		return getLatestActionPinReference(repo)
	}

	pins := GetActionPinsByRepo(repo)
	if len(pins) == 0 {
		return getLatestActionPinReference(repo)
	}

	latestVersion := pins[0].Version
	pinnedRef, err := ResolveActionPin(repo, latestVersion, ctx)
	if err != nil || pinnedRef == "" {
		return getLatestActionPinReference(repo)
	}
	return pinnedRef
}
