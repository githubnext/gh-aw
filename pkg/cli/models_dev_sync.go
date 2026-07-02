package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/github/gh-aw/pkg/logger"
)

const (
	modelsDevFetchTimeout = 5 * time.Second
	modelsDevMaxBodyBytes = 4 * 1024 * 1024 // 4 MiB safety cap
)

// modelsDevCatalogURL is a variable so tests can override it with a local HTTP server.
var modelsDevCatalogURL = "https://models.dev/catalog.json"

var modelsDevLog = logger.New("cli:models_dev_sync")

// rawModelsDevCatalog mirrors the top-level models.dev catalog JSON structure.
type rawModelsDevCatalog struct {
	Providers map[string]rawModelsDevProvider `json:"providers"`
}

type rawModelsDevProvider struct {
	Models map[string]rawModelsDevModel `json:"models"`
}

type rawModelsDevModel struct {
	// Cost values are per-million-token numbers (or pre-normalized strings) in the catalog.
	Cost map[string]json.RawMessage `json:"cost"`
}

// modelsDevPricingCache maps normalizedProvider → normalizedModel → per-token pricing.
type modelsDevPricingCache = map[string]map[string]map[string]float64

var (
	modelsDevOnce  sync.Once
	modelsDevCache modelsDevPricingCache

	// modelsDevHTTPClientFactory is overridable for tests.
	modelsDevHTTPClientFactory = func() *http.Client {
		return &http.Client{Timeout: modelsDevFetchTimeout}
	}
)

// ensureModelsDevCatalog downloads and normalizes the models.dev pricing catalog at most
// once per process. Network failures are logged and result in an empty (non-nil) cache so
// subsequent calls are instant no-ops.
func ensureModelsDevCatalog(ctx context.Context) modelsDevPricingCache {
	modelsDevOnce.Do(func() {
		cache, err := downloadAndParseModelsDevCatalog(ctx)
		if err != nil {
			modelsDevLog.Printf("models.dev catalog download failed (pricing fallback unavailable): %v", err)
			cache = modelsDevPricingCache{}
		} else {
			total := 0
			for _, models := range cache {
				total += len(models)
			}
			modelsDevLog.Printf("Downloaded models.dev catalog: %d providers, %d total models", len(cache), total)
		}
		modelsDevCache = cache
	})
	return modelsDevCache
}

func downloadAndParseModelsDevCatalog(ctx context.Context) (modelsDevPricingCache, error) {
	reqCtx, cancel := context.WithTimeout(ctx, modelsDevFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, modelsDevCatalogURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := modelsDevHTTPClientFactory().Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", modelsDevCatalogURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP %d from %s", resp.StatusCode, modelsDevCatalogURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, modelsDevMaxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return parseModelsDevCatalog(body)
}

// parseModelsDevCatalog parses the raw models.dev catalog JSON and normalizes pricing
// to per-token float64 values. Numeric catalog values are in USD per-million tokens and
// are divided by 1,000,000; string values are treated as already per-token.
func parseModelsDevCatalog(data []byte) (modelsDevPricingCache, error) {
	var raw rawModelsDevCatalog
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing models.dev catalog JSON: %w", err)
	}

	cache := make(modelsDevPricingCache)
	for providerName, provider := range raw.Providers {
		normalizedProvider := normalizeCatalogProvider(providerName)
		if normalizedProvider == "" {
			continue
		}
		if cache[normalizedProvider] == nil {
			cache[normalizedProvider] = make(map[string]map[string]float64)
		}
		for modelName, model := range provider.Models {
			normalizedModel := strings.ToLower(strings.TrimSpace(modelName))
			if normalizedModel == "" {
				continue
			}
			pricing := parseDevCostMap(model.Cost)
			if len(pricing) > 0 {
				cache[normalizedProvider][normalizedModel] = pricing
			}
		}
	}
	return cache, nil
}

// parseDevCostMap converts a raw cost map from models.dev (per-million numbers or
// already-normalized per-token strings) into per-token float64 values.
func parseDevCostMap(raw map[string]json.RawMessage) map[string]float64 {
	if len(raw) == 0 {
		return nil
	}
	result := make(map[string]float64, len(raw))
	for key, val := range raw {
		if len(val) == 0 {
			continue
		}
		// Attempt numeric decode — models.dev stores prices per million tokens.
		var f float64
		if err := json.Unmarshal(val, &f); err == nil {
			result[key] = f / 1_000_000 // convert per-million → per-token
			continue
		}
		// Fall back to string decode (pre-normalized per-token string values).
		var s string
		if err := json.Unmarshal(val, &s); err == nil {
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
				result[key] = parsed
			}
		}
	}
	return result
}

// findPricingInModelsDev looks up per-token pricing for the given provider/model from the
// downloaded models.dev catalog. Returns (nil, false) when the catalog is unavailable or
// the model is not found. Uses the same normalization as findModelPricing.
func findPricingInModelsDev(ctx context.Context, provider, model string) (map[string]float64, bool) {
	cache := ensureModelsDevCatalog(ctx)
	if len(cache) == 0 {
		return nil, false
	}

	normalizedProvider := normalizeCatalogProvider(provider)
	normalizedModel := strings.ToLower(strings.TrimSpace(model))
	if normalizedModel == "" {
		return nil, false
	}
	comparableModel := normalizeComparableModelID(normalizedModel)

	// Provider-scoped exact match.
	if normalizedProvider != "" {
		if providerModels, ok := cache[normalizedProvider]; ok {
			if pricing, ok := providerModels[normalizedModel]; ok {
				return pricing, true
			}
			// Comparable (dot/underscore-normalized) model ID match.
			for mn, pricing := range providerModels {
				if normalizeComparableModelID(mn) == comparableModel {
					return pricing, true
				}
			}
		}
	}

	// Cross-provider fallback (when provider is unknown or empty).
	for _, providerModels := range cache {
		if pricing, ok := providerModels[normalizedModel]; ok {
			return pricing, true
		}
		for mn, pricing := range providerModels {
			if normalizeComparableModelID(mn) == comparableModel {
				return pricing, true
			}
		}
	}

	return nil, false
}
