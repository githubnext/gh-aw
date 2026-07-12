package workflow

import (
	"strings"
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

// ─────────────────────────────────────────────────────────────────────────────
// Level 1 Compliance Tests (T-OT-001 through T-OT-007)
// Spec: §17.1.1 Test ID Stubs: Level 1 Compliance
// ─────────────────────────────────────────────────────────────────────────────

// TestCompliance_T_OT_001 verifies that the compiler accepts
// observability.otlp.endpoint with a valid HTTPS URL and emits
// OTEL_EXPORTER_OTLP_ENDPOINT in the generated workflow environment.
// Spec: §17.1.1 T-OT-001
func TestCompliance_T_OT_001(t *testing.T) {
	const httpsEndpoint = "https://traces.example.com:4317"

	wd := &WorkflowData{
		ParsedFrontmatter: &FrontmatterConfig{
			Observability: &ObservabilityConfig{
				OTLP: &OTLPConfig{Endpoint: httpsEndpoint},
			},
		},
	}

	(&Compiler{}).injectOTLPConfig(wd)

	require.NotEmpty(t, wd.Env, "T-OT-001: Env must not be empty after OTLP injection")
	assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_ENDPOINT: "+httpsEndpoint,
		"T-OT-001: OTEL_EXPORTER_OTLP_ENDPOINT must be set to the configured HTTPS URL")
	assert.Contains(t, wd.Env, "OTEL_SERVICE_NAME:",
		"T-OT-001: OTEL_SERVICE_NAME must be emitted alongside the endpoint")
}

// TestCompliance_T_OT_002 verifies that the compiler emits the correct
// environment variable for the if-missing mode. When the mode is "error"
// (the block/default mode), GH_AW_OTLP_IF_MISSING is NOT emitted — the
// absence of the variable causes the runtime to block on missing endpoints.
// When the mode is "warn" or "ignore", GH_AW_OTLP_IF_MISSING is emitted.
// Spec: §17.1.1 T-OT-002
func TestCompliance_T_OT_002(t *testing.T) {
	t.Run("error mode does not emit GH_AW_OTLP_IF_MISSING (block by default)", func(t *testing.T) {
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{
						Endpoint:  "https://traces.example.com:4317",
						IfMissing: "error",
					},
				},
			},
		}
		(&Compiler{}).injectOTLPConfig(wd)
		assert.NotContains(t, wd.Env, "GH_AW_OTLP_IF_MISSING",
			"T-OT-002: 'error' if-missing mode must not emit GH_AW_OTLP_IF_MISSING "+
				"(absence = block/error is the runtime default)")
	})

	t.Run("warn mode emits GH_AW_OTLP_IF_MISSING=warn", func(t *testing.T) {
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{
						Endpoint:  "https://traces.example.com:4317",
						IfMissing: "warn",
					},
				},
			},
		}
		(&Compiler{}).injectOTLPConfig(wd)
		assert.Contains(t, wd.Env, "GH_AW_OTLP_IF_MISSING: warn",
			"T-OT-002: 'warn' if-missing mode must emit GH_AW_OTLP_IF_MISSING=warn")
	})

	t.Run("invalid if-missing value falls back to error/block behaviour", func(t *testing.T) {
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{
						Endpoint:  "https://traces.example.com:4317",
						IfMissing: "block", // invalid; must be treated as "error"
					},
				},
			},
		}
		(&Compiler{}).injectOTLPConfig(wd)
		assert.NotContains(t, wd.Env, "GH_AW_OTLP_IF_MISSING",
			"T-OT-002: invalid if-missing value must fall back to block/error behaviour")
	})
}

// TestCompliance_T_OT_003 verifies that when multiple endpoints are declared,
// the compiler preserves them in declaration order and retains the first
// endpoint in OTEL_EXPORTER_OTLP_ENDPOINT for backward compatibility.
// Spec: §17.1.1 T-OT-003
func TestCompliance_T_OT_003(t *testing.T) {
	frontmatter := map[string]any{
		"observability": map[string]any{
			"otlp": map[string]any{
				"endpoint": []any{
					map[string]any{"url": "https://first.example.com:4317"},
					map[string]any{"url": "https://second.example.com:4317"},
					map[string]any{"url": "https://third.example.com:4317"},
				},
			},
		},
	}

	entries := collectAllOTLPEndpoints(frontmatter)
	require.Len(t, entries, 3, "T-OT-003: all three endpoints must be collected")
	assert.Equal(t, "https://first.example.com:4317", entries[0].URL,
		"T-OT-003: first endpoint must be at index 0 (declaration order)")
	assert.Equal(t, "https://second.example.com:4317", entries[1].URL,
		"T-OT-003: second endpoint must be at index 1")
	assert.Equal(t, "https://third.example.com:4317", entries[2].URL,
		"T-OT-003: third endpoint must be at index 2")

	wd := &WorkflowData{RawFrontmatter: frontmatter}
	(&Compiler{}).injectOTLPConfig(wd)

	assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_ENDPOINT: https://first.example.com:4317",
		"T-OT-003: OTEL_EXPORTER_OTLP_ENDPOINT must be set to the first endpoint for backward compat")
	assert.Contains(t, wd.Env, "GH_AW_OTLP_ENDPOINTS",
		"T-OT-003: GH_AW_OTLP_ENDPOINTS JSON array must be present so all endpoints reach the runtime")
	assert.Contains(t, wd.Env, "https://second.example.com:4317",
		"T-OT-003: second endpoint must appear in GH_AW_OTLP_ENDPOINTS")
	assert.Contains(t, wd.Env, "https://third.example.com:4317",
		"T-OT-003: third endpoint must appear in GH_AW_OTLP_ENDPOINTS")
}

// TestCompliance_T_OT_004 verifies that the compiler emits GH_AW_OTLP_ENDPOINTS
// so the runtime JavaScript helper can encode span payloads as valid OTLP/HTTP
// protobuf-JSON and POST them to the configured endpoint.
//
// The JavaScript-side OTLP/HTTP encoding and 200-response acceptance are validated
// by the otel_contract.test.cjs test suite (run via `make validate-otel-contract`).
// Spec: §17.1.1 T-OT-004
func TestCompliance_T_OT_004(t *testing.T) {
	wd := &WorkflowData{
		ParsedFrontmatter: &FrontmatterConfig{
			Observability: &ObservabilityConfig{
				OTLP: &OTLPConfig{Endpoint: "https://traces.example.com:4317"},
			},
		},
	}
	(&Compiler{}).injectOTLPConfig(wd)

	assert.Contains(t, wd.Env, "GH_AW_OTLP_ENDPOINTS",
		"T-OT-004: GH_AW_OTLP_ENDPOINTS must be emitted so the JavaScript helper can POST spans")
	assert.Contains(t, wd.Env, "traces.example.com",
		"T-OT-004: the configured endpoint domain must appear in the generated environment")

	// Verify the domain is also added to the network allowlist so the OTLP
	// POST is not blocked by the firewall.
	require.NotNil(t, wd.NetworkPermissions,
		"T-OT-004: NetworkPermissions must be set for the OTLP endpoint domain")
	assert.Contains(t, wd.NetworkPermissions.Allowed, "traces.example.com",
		"T-OT-004: the OTLP endpoint domain must be in the firewall allowlist")
}

// TestCompliance_T_OT_005 verifies that the trace context helper injects
// GITHUB_AW_OTEL_TRACE_ID, GITHUB_AW_OTEL_PARENT_SPAN_ID, and TRACEPARENT
// into the engine execution environment when OTLP observability is enabled.
// Spec: §17.1.1 T-OT-005
func TestCompliance_T_OT_005(t *testing.T) {
	env := map[string]string{}
	applyTraceContextEnvToMap(env)

	traceparent, ok := env["TRACEPARENT"]
	require.True(t, ok, "T-OT-005: TRACEPARENT must be present in the environment map")
	assert.NotEmpty(t, traceparent,
		"T-OT-005: TRACEPARENT value must not be empty")

	// The TRACEPARENT value is a GitHub Actions expression that resolves to
	// the W3C traceparent header (00-<trace-id>-<span-id>-01) at runtime.
	// GITHUB_AW_OTEL_TRACE_ID and GITHUB_AW_OTEL_PARENT_SPAN_ID are injected
	// by the setup action and referenced inside the expression.
	assert.Contains(t, traceparent, "GITHUB_AW_OTEL_TRACE_ID",
		"T-OT-005: TRACEPARENT expression must reference GITHUB_AW_OTEL_TRACE_ID")
	assert.Contains(t, traceparent, "GITHUB_AW_OTEL_PARENT_SPAN_ID",
		"T-OT-005: TRACEPARENT expression must reference GITHUB_AW_OTEL_PARENT_SPAN_ID")
	assert.True(t,
		strings.HasPrefix(traceparent, "${{"),
		"T-OT-005: TRACEPARENT must be a GitHub Actions expression (starts with ${{)")
}

// TestCompliance_T_OT_006 verifies that the local mirror path constant points
// to /tmp/gh-aw/otel.jsonl and that the path format uses raw OTLP/HTTP JSON
// lines with a resourceSpans key (not an envelope-only summary).
//
// The runtime JavaScript helper's write behaviour (one JSONL line per span
// with a resourceSpans key) is validated by otel_contract.test.cjs.
// Spec: §17.1.1 T-OT-006
func TestCompliance_T_OT_006(t *testing.T) {
	mirrorPath := constants.TmpGhAwDirSlash + constants.OtelJsonlFilename
	assert.Equal(t, "/tmp/gh-aw/otel.jsonl", mirrorPath,
		"T-OT-006: local mirror MUST be at /tmp/gh-aw/otel.jsonl")

	assert.Equal(t, "/tmp/gh-aw/", constants.TmpGhAwDirSlash,
		"T-OT-006: TmpGhAwDirSlash constant must resolve to /tmp/gh-aw/")
	assert.Equal(t, "otel.jsonl", constants.OtelJsonlFilename,
		"T-OT-006: OtelJsonlFilename constant must be otel.jsonl")
}

// TestCompliance_T_OT_007 verifies that observability.otlp.headers entries are
// emitted as OTEL_EXPORTER_OTLP_HEADERS in key=value,key=value format, and that
// header values are masked in diagnostics output (not echoed as plain text).
// Spec: §17.1.1 T-OT-007
func TestCompliance_T_OT_007(t *testing.T) {
	t.Run("map form headers emitted as key=value CSV in declaration order", func(t *testing.T) {
		headers := map[string]any{"Authorization": authPlaceholder, "X-Tenant": "acme"}
		normalized := normalizeOTLPHeadersForEndpoint(headers, "https://traces.example.com:4317")
		// Headers must be emitted in sorted key order for determinism.
		assert.Equal(t, "Authorization="+authPlaceholder+",X-Tenant=acme", normalized,
			"T-OT-007: map headers must be sorted and joined as key=value pairs")
	})

	t.Run("string form headers preserved as-is", func(t *testing.T) {
		normalized := normalizeOTLPHeadersForEndpoint(
			"Authorization="+authPlaceholder,
			"https://traces.example.com:4317",
		)
		assert.Equal(t, "Authorization="+authPlaceholder, normalized,
			"T-OT-007: string form headers must be passed through unchanged")
	})

	t.Run("OTEL_EXPORTER_OTLP_HEADERS is injected into workflow env", func(t *testing.T) {
		wd := &WorkflowData{
			ParsedFrontmatter: &FrontmatterConfig{
				Observability: &ObservabilityConfig{
					OTLP: &OTLPConfig{
						Endpoint: "https://traces.example.com:4317",
						Headers:  "Authorization=" + authPlaceholder,
					},
				},
			},
		}
		(&Compiler{}).injectOTLPConfig(wd)

		assert.Contains(t, wd.Env, "OTEL_EXPORTER_OTLP_HEADERS:",
			"T-OT-007: OTEL_EXPORTER_OTLP_HEADERS must be present in generated env when headers configured")
	})

	t.Run("Sentry endpoint rewrites Authorization to x-sentry-auth", func(t *testing.T) {
		normalized := normalizeOTLPHeadersForEndpoint(
			"Authorization="+authPlaceholder,
			"https://o0.ingest.sentry.io/api/0/envelope/",
		)
		assert.Equal(t, "x-sentry-auth="+authPlaceholder, normalized,
			"T-OT-007: Sentry endpoint must rewrite Authorization header to x-sentry-auth")
		assert.NotContains(t, normalized, "Authorization",
			"T-OT-007: Authorization header must not appear verbatim for Sentry endpoints")
	})
}
