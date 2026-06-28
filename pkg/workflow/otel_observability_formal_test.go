package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const determinismTestIterations = 10
const authPlaceholder = "******"

func TestFormal_EndpointFormNormalization(t *testing.T) {
	t.Run("string object and array normalize to ordered entries", func(t *testing.T) {
		stringForm := map[string]any{
			"observability": map[string]any{
				"otlp": map[string]any{
					"endpoint": "https://string.example.com:4317",
				},
			},
		}
		assert.Equal(t,
			[]otlpEndpointEntry{{URL: "https://string.example.com:4317"}},
			collectAllOTLPEndpoints(stringForm),
		)

		objectForm := map[string]any{
			"observability": map[string]any{
				"otlp": map[string]any{
					"endpoint": map[string]any{"url": "https://object.example.com:4317"},
				},
			},
		}
		assert.Equal(t,
			[]otlpEndpointEntry{{URL: "https://object.example.com:4317"}},
			collectAllOTLPEndpoints(objectForm),
		)

		arrayForm := map[string]any{
			"observability": map[string]any{
				"otlp": map[string]any{
					"endpoint": []any{
						map[string]any{"url": "https://first.example.com:4317"},
						map[string]any{"url": "https://second.example.com:4317"},
					},
				},
			},
		}
		assert.Equal(t,
			[]otlpEndpointEntry{
				{URL: "https://first.example.com:4317"},
				{URL: "https://second.example.com:4317"},
			},
			collectAllOTLPEndpoints(arrayForm),
		)
	})

	t.Run("empty and absent normalize to empty", func(t *testing.T) {
		assert.Empty(t, collectAllOTLPEndpoints(nil))
		assert.Empty(t, collectAllOTLPEndpoints(map[string]any{}))
		assert.Empty(t, collectAllOTLPEndpoints(map[string]any{"observability": map[string]any{}}))
	})
}

func TestFormal_HeaderMapDeterminism(t *testing.T) {
	headers := map[string]any{"z": "3", "a": "1", "m": "2"}
	want := "a=1,m=2,z=3"

	for range determinismTestIterations {
		assert.Equal(t, want, normalizeOTLPHeadersForEndpoint(headers, "https://example.com:4317"))
	}
}

func TestFormal_SentryAuthHeaderRewrite(t *testing.T) {
	normalizedSentryHeaders := normalizeOTLPHeadersForEndpoint("Authorization="+authPlaceholder, "https://o0.ingest.sentry.io/api/0/envelope/")
	normalizedNonSentryHeaders := normalizeOTLPHeadersForEndpoint("Authorization="+authPlaceholder, "https://otlp.example.com:4317")
	normalizedSentryMixedHeaders := normalizeOTLPHeadersForEndpoint(
		map[string]any{"Authorization": authPlaceholder, "X-Tenant": "acme"},
		"https://o0.ingest.sentry.io/api/0/envelope/",
	)

	assert.Equal(t, "x-sentry-auth="+authPlaceholder, normalizedSentryHeaders)
	assert.Equal(t, "Authorization="+authPlaceholder, normalizedNonSentryHeaders)
	assert.Equal(t, "x-sentry-auth="+authPlaceholder+",X-Tenant=acme", normalizedSentryMixedHeaders)
}

func TestFormal_IfMissingPolicyValidation(t *testing.T) {
	assert.Equal(t, "error", normalizeOTLPIfMissingMode("error"))
	assert.Equal(t, "warn", normalizeOTLPIfMissingMode("WARN"))
	assert.Equal(t, "ignore", normalizeOTLPIfMissingMode(" Ignore "))
}

func TestFormal_ServiceNameFormation(t *testing.T) {
	assert.Equal(t, "gh-aw", otelServiceName(nil))
	assert.Equal(t, "gh-aw.repo-triage-weekly", otelServiceName(&WorkflowData{WorkflowID: "repo-triage-weekly", Name: "Sample Name"}))
	assert.Equal(t, "gh-aw.repo-triage-weekly", otelServiceName(&WorkflowData{WorkflowID: "Repo Triage/Weekly", Name: "Sample Name"}))
	assert.Equal(t, "gh-aw.workflow-name", otelServiceName(&WorkflowData{Name: "Workflow Name"}))
}

func TestFormal_StaticDomainExtraction(t *testing.T) {
	assert.Equal(t, "traces.example.com", extractOTLPEndpointDomain("https://traces.example.com:4317"))
	assert.Empty(t, extractOTLPEndpointDomain(""))
	assert.Empty(t, extractOTLPEndpointDomain("${{ secrets.OTLP_ENDPOINT }}"))
}

func TestFormal_ExpressionProducesNoAllowlistEntry(t *testing.T) {
	assert.Empty(t, extractOTLPEndpointDomain("${{ secrets.OTLP_ENDPOINT }}"))
}

func TestFormal_TopLevelHeadersApplyToStringFormOnly(t *testing.T) {
	stringForm := map[string]any{
		"observability": map[string]any{
			"otlp": map[string]any{
				"endpoint": "https://string.example.com:4317",
				"headers":  "Authorization=" + authPlaceholder,
			},
		},
	}
	entries := collectAllOTLPEndpoints(stringForm)
	require.Len(t, entries, 1)
	assert.Equal(t, "Authorization="+authPlaceholder, entries[0].Headers)

	objectForm := map[string]any{
		"observability": map[string]any{
			"otlp": map[string]any{
				"endpoint": map[string]any{
					"url":     "https://object.example.com:4317",
					"headers": "X-Per-Entry=v",
				},
				"headers": "Authorization=" + authPlaceholder,
			},
		},
	}
	objectEntries := collectAllOTLPEndpoints(objectForm)
	require.Len(t, objectEntries, 1)
	assert.Equal(t, "X-Per-Entry=v", objectEntries[0].Headers)
}

func TestFormal_FanOutPreservesDeclarationOrder(t *testing.T) {
	frontmatter := map[string]any{
		"observability": map[string]any{
			"otlp": map[string]any{
				"endpoint": []any{
					map[string]any{"url": "https://one.example.com:4317"},
					map[string]any{"url": "https://two.example.com:4317"},
					map[string]any{"url": "https://three.example.com:4317"},
				},
			},
		},
	}

	entries := collectAllOTLPEndpoints(frontmatter)
	require.Len(t, entries, 3)
	assert.Equal(t, "https://one.example.com:4317", entries[0].URL)
	assert.Equal(t, "https://two.example.com:4317", entries[1].URL)
	assert.Equal(t, "https://three.example.com:4317", entries[2].URL)
}

func TestFormal_MirrorPathConstant(t *testing.T) {
	assert.Equal(t, "/tmp/gh-aw/otel.jsonl", constants.TmpGhAwDirSlash+constants.OtelJsonlFilename)
}

func TestFormal_EmptyURLEntriesDiscarded(t *testing.T) {
	frontmatter := map[string]any{
		"observability": map[string]any{
			"otlp": map[string]any{
				"endpoint": []any{
					map[string]any{"url": ""},
					map[string]any{"url": "https://valid.example.com:4317"},
				},
			},
		},
	}

	assert.Equal(t, []otlpEndpointEntry{{URL: "https://valid.example.com:4317"}}, collectAllOTLPEndpoints(frontmatter))
}

func TestFormal_StringHeaderFormPreservedForNonSentry(t *testing.T) {
	assert.Equal(t,
		"Authorization="+authPlaceholder,
		normalizeOTLPHeadersForEndpoint("Authorization="+authPlaceholder, "https://otlp.example.com:4317"),
	)
}

func TestFormal_NilAndEmptyHeadersYieldEmptyString(t *testing.T) {
	assert.Empty(t, normalizeOTLPHeadersForEndpoint(nil, "https://example.com:4317"))
	assert.Empty(t, normalizeOTLPHeadersForEndpoint("", "https://example.com:4317"))
	assert.Empty(t, normalizeOTLPHeadersForEndpoint(map[string]any{}, "https://example.com:4317"))
}

func TestFormal_InvalidIfMissingFallsBackToDefault(t *testing.T) {
	for _, mode := range []string{"fail", "silent", "skip", "abort"} {
		assert.Empty(t, normalizeOTLPIfMissingMode(mode))
	}

	workflowData := &WorkflowData{
		RawFrontmatter: map[string]any{
			"observability": map[string]any{
				"otlp": map[string]any{
					"endpoint":   "https://traces.example.com:4317",
					"if-missing": "fail",
				},
			},
		},
		ParsedFrontmatter: &FrontmatterConfig{
			Observability: &ObservabilityConfig{
				OTLP: &OTLPConfig{
					Endpoint:  "https://traces.example.com:4317",
					IfMissing: "fail",
				},
			},
		},
	}
	(&Compiler{}).injectOTLPConfig(workflowData)
	assert.NotContains(t, workflowData.Env, "GH_AW_OTLP_IF_MISSING")

	validWorkflowData := &WorkflowData{
		RawFrontmatter: map[string]any{
			"observability": map[string]any{
				"otlp": map[string]any{
					"endpoint":   "https://traces.example.com:4317",
					"if-missing": "warn",
				},
			},
		},
		ParsedFrontmatter: &FrontmatterConfig{
			Observability: &ObservabilityConfig{
				OTLP: &OTLPConfig{
					Endpoint:  "https://traces.example.com:4317",
					IfMissing: "warn",
				},
			},
		},
	}
	(&Compiler{}).injectOTLPConfig(validWorkflowData)
	assert.Contains(t, validWorkflowData.Env, "GH_AW_OTLP_IF_MISSING")
	assert.Contains(t, validWorkflowData.Env, "warn")
}

func TestFormal_AbsentObservabilityProducesNoEndpoints(t *testing.T) {
	assert.Empty(t, collectAllOTLPEndpoints(nil))
	assert.Empty(t, collectAllOTLPEndpoints(map[string]any{}))
	assert.Empty(t, collectAllOTLPEndpoints(map[string]any{"observability": nil}))
}
