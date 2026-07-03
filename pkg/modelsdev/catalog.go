package modelsdev

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/syncutil"
)

const (
	fetchTimeout = 5 * time.Second
	maxBodyBytes = 4 * 1024 * 1024 // 4 MiB safety cap
)

// catalogURL is a variable so tests can override it with a local HTTP server.
var catalogURL = "https://models.dev/catalog.json"

var log = logger.New("modelsdev:catalog")

// rawCatalog mirrors the top-level models.dev catalog JSON structure.
type rawCatalog struct {
	Providers map[string]rawProvider `json:"providers"`
}

type rawProvider struct {
	Models map[string]rawModel `json:"models"`
}

type rawModel struct {
	// Cost values are per-million-token numbers (or pre-normalized strings) in the catalog.
	Cost map[string]json.RawMessage `json:"cost"`
}

// pricingCache maps normalizedProvider → normalizedModel → per-token pricing.
type pricingCache = map[string]map[string]map[string]float64

var (
	catalogCache syncutil.OnceLoader[pricingCache]

	// httpClientFactory is overridable for tests.
	httpClientFactory = func() *http.Client {
		return &http.Client{Timeout: fetchTimeout}
	}
)

// FindPricing looks up per-token pricing for the given provider/model from the downloaded
// models.dev catalog. Returns (nil, false) when the catalog is unavailable or the model
// is not found.
func FindPricing(ctx context.Context, provider, model string) (map[string]float64, bool) {
	catalog := ensureCatalog(ctx)
	if len(catalog) == 0 {
		return nil, false
	}

	normalizedProvider := normalizeProvider(provider)
	trimmedModel := strings.TrimSpace(model)
	if trimmedModel == "" {
		return nil, false
	}
	normalizedModel := strings.ToLower(trimmedModel)
	comparableModel := normalizeComparableModelID(normalizedModel)

	// Provider-scoped exact match.
	if normalizedProvider != "" {
		if providerModels, ok := catalog[normalizedProvider]; ok {
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
	for _, providerModels := range catalog {
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

// ensureCatalog downloads and normalizes the models.dev pricing catalog at most once per
// process. Network failures are logged and result in an empty (non-nil) cache so
// subsequent calls are instant no-ops.
func ensureCatalog(ctx context.Context) pricingCache {
	downloaded, _ := catalogCache.Get(func() (pricingCache, error) {
		downloaded, err := downloadAndParseCatalog(ctx)
		if err != nil {
			log.Printf("models.dev catalog download failed (pricing fallback unavailable): %v", err)
			return pricingCache{}, nil
		} else {
			total := 0
			for _, models := range downloaded {
				total += len(models)
			}
			log.Printf("Downloaded models.dev catalog: %d providers, %d total models", len(downloaded), total)
		}
		return downloaded, nil
	})
	return downloaded
}

func downloadAndParseCatalog(ctx context.Context) (pricingCache, error) {
	reqCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, catalogURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := httpClientFactory().Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", catalogURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP %d from %s", resp.StatusCode, catalogURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return parseCatalog(body)
}

// parseCatalog parses the raw models.dev catalog JSON and normalizes pricing to per-token
// float64 values. Numeric catalog values are in USD per-million tokens and are divided by
// 1,000,000; string values are treated as already per-token.
func parseCatalog(data []byte) (pricingCache, error) {
	var raw rawCatalog
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing models.dev catalog JSON: %w", err)
	}

	parsed := make(pricingCache)
	for providerName, provider := range raw.Providers {
		normalizedProvider := normalizeProvider(providerName)
		if normalizedProvider == "" {
			continue
		}
		if parsed[normalizedProvider] == nil {
			parsed[normalizedProvider] = make(map[string]map[string]float64)
		}
		for modelName, model := range provider.Models {
			trimmedModel := strings.TrimSpace(modelName)
			if trimmedModel == "" {
				continue
			}
			normalizedModel := strings.ToLower(trimmedModel)
			pricing := parseCostMap(model.Cost)
			if len(pricing) > 0 {
				parsed[normalizedProvider][normalizedModel] = pricing
			}
		}
	}
	return parsed, nil
}

// parseCostMap converts a raw cost map from models.dev (per-million numbers or
// already-normalized per-token strings) into per-token float64 values.
func parseCostMap(raw map[string]json.RawMessage) map[string]float64 {
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

func normalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "github", "copilot", "github_models":
		return "github-copilot"
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

func normalizeComparableModelID(value string) string {
	return strings.NewReplacer(".", "-", "_", "-").Replace(strings.ToLower(strings.TrimSpace(value)))
}
