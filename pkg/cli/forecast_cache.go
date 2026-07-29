package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/github/gh-aw/pkg/constants"
)

// forecastAICCacheFileName is a forecast-specific cache file written to each run
// folder alongside the downloaded usage artifact. It stores only the computed AI
// Credits (AIC) value so that repeated forecast runs can reload the metric without
// re-scanning the run directory or re-parsing the usage artifact.
//
// A dedicated file is used (rather than the shared run_summary.json) because
// run_summary.json acts as a "fully processed" marker for `gh aw logs`/`audit`;
// writing a partial summary there would make those commands serve incomplete data.
const forecastAICCacheFileName = "forecast_aic.json"

// forecastAICCache is the on-disk payload cached per run for forecasting.
type forecastAICCache struct {
	CLIVersion string    `json:"cli_version"` // CLI version used to compute the AIC (cache invalidation key)
	RunID      int64     `json:"run_id"`      // Workflow run database ID
	AIC        float64   `json:"ai_credits"`  // Total AI Credits consumed by the run
	NoData     bool      `json:"no_data"`     // True when the run has no usage artifact / no AIC data (negative cache)
	CachedAt   time.Time `json:"cached_at"`   // When this cache entry was written
}

// loadForecastAICCache returns the cached AIC for a run when a valid, version-matching
// forecast cache file exists. The second return value reports whether a usable cache
// entry was found. A negative-cache entry (NoData) is a hit that returns 0, letting the
// caller skip the network for runs that are known to have no AIC data. Stale entries
// (version mismatch, mismatched run ID, or a positive-AIC entry that is somehow <= 0)
// are treated as misses so the caller recomputes.
func loadForecastAICCache(dir string, runID int64) (float64, bool) {
	path := filepath.Join(dir, forecastAICCacheFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	var c forecastAICCache
	if err := json.Unmarshal(data, &c); err != nil {
		return 0, false
	}
	if c.CLIVersion != GetVersion() || c.RunID != runID {
		return 0, false
	}
	// Negative cache: the run is known to have no AIC data. Report a hit so the caller
	// short-circuits and avoids re-downloading the (absent) usage artifact every run.
	if c.NoData {
		return 0, true
	}
	if c.AIC <= 0 {
		return 0, false
	}
	return c.AIC, true
}

// saveForecastAICCache writes the computed AIC for a run to the forecast cache file.
// It is best-effort: any error (including a non-positive AIC) is silently ignored so a
// cache-write failure never fails the forecast. The run directory is expected to already
// exist (it holds the downloaded usage artifact), but is created defensively.
func saveForecastAICCache(dir string, runID int64, aic float64) {
	if aic <= 0 {
		return
	}
	writeForecastAICCache(dir, runID, forecastAICCache{
		CLIVersion: GetVersion(),
		RunID:      runID,
		AIC:        aic,
		CachedAt:   time.Now().UTC(),
	})
}

// saveForecastNoDataCache writes a negative-cache marker for a run that has no usage
// artifact or no AIC data. This lets subsequent forecast runs skip the (repeated,
// definitively empty) network lookups for completed runs that will never yield AIC
// data. Only call this for permanent no-data conditions — never for transient errors
// such as context cancellation or download failures.
func saveForecastNoDataCache(dir string, runID int64) {
	writeForecastAICCache(dir, runID, forecastAICCache{
		CLIVersion: GetVersion(),
		RunID:      runID,
		NoData:     true,
		CachedAt:   time.Now().UTC(),
	})
}

// writeForecastAICCache serialises c to the run's forecast cache file. Best-effort:
// all errors are logged and swallowed so a cache-write failure never fails the forecast.
func writeForecastAICCache(dir string, runID int64, c forecastAICCache) {
	data, err := json.MarshalIndent(&c, "", "  ")
	if err != nil {
		forecastRunLog.Printf("Failed to marshal forecast AIC cache for run %d: %v", runID, err)
		return
	}
	if err := os.MkdirAll(dir, constants.DirPermPublic); err != nil {
		forecastRunLog.Printf("Failed to create dir for forecast AIC cache for run %d: %v", runID, err)
		return
	}
	path := filepath.Join(dir, forecastAICCacheFileName)
	if err := os.WriteFile(path, data, constants.FilePermPublic); err != nil {
		forecastRunLog.Printf("Failed to write forecast AIC cache for run %d: %v", runID, err)
		return
	}
	forecastRunLog.Printf("Wrote forecast AIC cache for run %d: aic=%.3f, no_data=%t (path=%s)", runID, c.AIC, c.NoData, path)
}
