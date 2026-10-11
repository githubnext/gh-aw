package cli

import (
	"encoding/json"
	"testing"

	"github.com/github/gh-aw/pkg/scanfindings"
	"github.com/stretchr/testify/require"
)

func TestZizmorFindingsToSharedPreservesNativeSemantics(t *testing.T) {
	var findings []zizmorFinding
	require.NoError(t, json.Unmarshal([]byte(`[
		{
			"ident":"template-injection","desc":"untrusted input","url":"https://example.com/rule",
			"determinations":{"severity":"High"},
			"locations":[
				{"symbolic":{"key":{"Local":{"given_path":"first.yml"}}},
				 "concrete":{"location":{"start_point":{"row":2,"column":4}}}},
				{"symbolic":{"key":{"Local":{"given_path":"second.yml"}}},
				 "concrete":{"location":{"start_point":{"row":8,"column":9}}}}
			]
		},
		{"ident":"no-location","determinations":{"severity":"Critical"}},
		{"ident":"informational","desc":"notice","determinations":{"severity":"note"},
		 "locations":[{"concrete":{"location":{"start_point":{"row":0,"column":0}}}}]}
	]`), &findings))

	lines := []string{"one", "two", "three", "four", "five"}
	require.Equal(t, []scanfindings.Finding{
		{
			RuleID: "template-injection", Severity: scanfindings.SeverityHigh,
			Message: "[High] template-injection: untrusted input (https://example.com/rule)",
			File:    "display.yml", Line: 3, Column: 5, Context: lines,
		},
		{
			RuleID: "informational", Severity: scanfindings.SeverityInfo,
			Message: "[note] informational: notice",
			File:    "display.yml", Line: 1, Column: 1, Context: []string{"one"},
		},
	}, zizmorFindingsToShared("display.yml", findings, lines))
}

func TestPoutineFindingsToSharedPreservesNativeSemantics(t *testing.T) {
	var output poutineOutput
	require.NoError(t, json.Unmarshal([]byte(`{
		"findings":[
			{"rule_id":"known","purl":"pkg:github/example",
			 "meta":{"path":"native.yml","line":3,"details":"extra detail"}},
			{"rule_id":"missing","meta":{}},
			{"rule_id":"note","meta":{"path":"note.yml","line":-1}}
		],
		"rules":{
			"known":{"id":"known","title":"Rule title","description":"unused","level":"error"},
			"note":{"id":"note","level":"note"}
		}
	}`), &output))

	lines := []string{"one", "two", "three", "four", "five"}
	require.Equal(t, []scanfindings.Finding{
		{
			RuleID: "known", Severity: scanfindings.SeverityHigh,
			Message: "[error] known: Rule title - extra detail",
			File:    "native.yml", Line: 3, Column: 1, Context: lines,
		},
		{
			RuleID: "missing", Severity: scanfindings.SeverityMedium,
			Message: "[warning] missing: missing",
			File:    "fallback.yml", Line: 1, Column: 1, Context: []string{"one"},
		},
		{
			RuleID: "note", Severity: scanfindings.SeverityInfo,
			Message: "[note] note: note",
			File:    "note.yml", Line: -1, Column: 1,
		},
	}, poutineFindingsToShared(output.Findings, output.Rules, "fallback.yml", lines))
}

func TestGrypeFindingsToSharedPreservesNativeSemantics(t *testing.T) {
	var output grypeOutput
	require.NoError(t, json.Unmarshal([]byte(`{"matches":[
		{
			"vulnerability":{"id":"CVE-1","dataSource":"https://example.com/CVE-1",
				"severity":"Critical","fix":{"versions":["1.2","1.3"],"state":"fixed"}},
			"artifact":{"name":"lib","version":"1.0","type":"deb"}
		},
		{"vulnerability":{"id":"CVE-2"},"artifact":{"name":"other","version":"2.0"}},
		{"vulnerability":{"id":"CVE-3","severity":"Negligible"},
		 "artifact":{"name":"minor","version":"3.0"}}
	]}`), &output))

	require.Equal(t, []scanfindings.Finding{
		{
			RuleID: "CVE-1", Severity: scanfindings.SeverityCritical,
			Message: "[Critical] CVE-1: lib@1.0 (fix: 1.2, 1.3) (https://example.com/CVE-1)",
			File:    "image:tag", Line: 1, Column: 1,
		},
		{
			RuleID: "CVE-2", Severity: scanfindings.SeverityUnknown,
			Message: "[Unknown] CVE-2: other@2.0",
			File:    "image:tag", Line: 1, Column: 1,
		},
		{
			RuleID: "CVE-3", Severity: scanfindings.SeverityLow,
			Message: "[Negligible] CVE-3: minor@3.0",
			File:    "image:tag", Line: 1, Column: 1,
		},
	}, grypeFindingsToShared("image:tag", output.Matches))
}

func TestRunnerGuardFindingsToSharedPreservesNativeSemantics(t *testing.T) {
	var output runnerGuardOutput
	require.NoError(t, json.Unmarshal([]byte(`{"score":50,"grade":"D","findings":[
		{"rule_id":"RGS-001","name":"Rule name","severity":"HIGH",
		 "description":"extra detail","remediation":"unused","file":"native.yml",
		 "job_id":"agent","line":3},
		{"rule_id":"RGS-002","name":"Unnamed severity","file":"default.yml","line":0},
		{"rule_id":"RGS-003","severity":"low","file":"negative.yml","line":-1}
	]}`), &output))

	lines := []string{"one", "two", "three", "four", "five"}
	require.Equal(t, []scanfindings.Finding{
		{
			RuleID: "RGS-001", Severity: scanfindings.SeverityHigh,
			Message: "[HIGH] RGS-001: Rule name - extra detail",
			File:    "native.yml", Line: 3, Column: 1, Context: lines,
		},
		{
			RuleID: "RGS-002", Severity: scanfindings.SeverityUnknown,
			Message: "RGS-002: Unnamed severity",
			File:    "default.yml", Line: 1, Column: 1, Context: []string{"one"},
		},
		{
			RuleID: "RGS-003", Severity: scanfindings.SeverityLow,
			Message: "[low] RGS-003",
			File:    "negative.yml", Line: -1, Column: 1,
		},
	}, runnerGuardFindingsToShared(output.Findings, lines))
}

func TestScannerFindingsToSharedEmptyInput(t *testing.T) {
	require.Empty(t, zizmorFindingsToShared("file.yml", nil, nil))
	require.Empty(t, poutineFindingsToShared(nil, nil, "file.yml", nil))
	require.Empty(t, grypeFindingsToShared("image:tag", nil))
	require.Empty(t, runnerGuardFindingsToShared(nil, nil))
}
