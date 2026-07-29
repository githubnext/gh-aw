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
	CachedAt   time.Time `json:"cached_at"`   // When this cache entry was written
}

// loadForecastAICCache returns the cached AIC for a run when a valid, version-matching
// forecast cache file exists. The second return value reports whether a usable cache
// entry was found. Stale entries (version mismatch, mismatched run ID, or non-positive
// AIC) are treated as misses so the caller recomputes.
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
	if c.CLIVersion != GetVersion() || c.RunID != runID || c.AIC <= 0 {
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
	c := forecastAICCache{
		CLIVersion: GetVersion(),
		RunID:      runID,
		AIC:        aic,
		CachedAt:   time.Now().UTC(),
	}
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
	forecastRunLog.Printf("Wrote forecast AIC cache for run %d: aic=%.3f (path=%s)", runID, aic, path)
}
