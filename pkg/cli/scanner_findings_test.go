package cli

import (
	"encoding/json"
	"testing"

	"github.com/github/gh-aw/pkg/scanfindings"
	"github.com/stretchr/testify/require"
)

const zizmorNativeFindingsFixture = `[
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
	]`

const poutineNativeFindingsFixture = `{
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
	}`

const grypeNativeFindingsFixture = `{"matches":[
		{
			"vulnerability":{"id":"CVE-1","dataSource":"https://example.com/CVE-1",
				"severity":"Critical","fix":{"versions":["1.2","1.3"],"state":"fixed"}},
			"artifact":{"name":"lib","version":"1.0","type":"deb"}
		},
		{"vulnerability":{"id":"CVE-2"},"artifact":{"name":"other","version":"2.0"}},
		{"vulnerability":{"id":"CVE-3","severity":"Negligible"},
		 "artifact":{"name":"minor","version":"3.0"}}
	]}`

const runnerGuardNativeFindingsFixture = `{"score":50,"grade":"D","findings":[
		{"rule_id":"RGS-001","name":"Rule name","severity":"HIGH",
		 "description":"extra detail","remediation":"unused","file":"native.yml",
		 "job_id":"agent","line":3},
		{"rule_id":"RGS-002","name":"Unnamed severity","file":"default.yml","line":0},
		{"rule_id":"RGS-003","severity":"low","file":"negative.yml","line":-1}
	]}`

func TestScannerFindingsToShared(t *testing.T) {
	lines := []string{"one", "two", "three", "four", "five"}
	tests := []struct {
		name    string
		fixture string
		convert func(*testing.T, string, []string) []scanfindings.Finding
		want    []scanfindings.Finding
		edges   []scannerFindingTestCase
	}{
		{
			name: "zizmor", fixture: zizmorNativeFindingsFixture,
			convert: func(t *testing.T, fixture string, fileLines []string) []scanfindings.Finding {
				var findings []zizmorFinding
				require.NoError(t, json.Unmarshal([]byte(fixture), &findings), "zizmor fixture JSON should parse")
				return zizmorFindingsToShared("display.yml", findings, fileLines)
			},
			want: []scanfindings.Finding{
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
			},
			edges: []scannerFindingTestCase{
				{name: "nil input", fixture: `null`, want: []scanfindings.Finding{}},
				{name: "non-nil empty input", fixture: `[]`, want: []scanfindings.Finding{}},
				{
					name:    "missing null and empty locations are skipped",
					fixture: `[{},{"ident":"null","locations":null},{"ident":"empty","locations":[]}]`,
					want:    []scanfindings.Finding{},
				},
				{
					name: "partial primary location is not replaced by secondary",
					fixture: `[{"ident":"partial","locations":[{},
						{"concrete":{"location":{"start_point":{"row":2,"column":4}}}}]}]`,
					want: []scanfindings.Finding{{
						RuleID: "partial", Severity: scanfindings.SeverityUnknown, Message: "partial",
						File: "display.yml", Line: 1, Column: 1, Context: []string{"one"},
					}},
				},
				{
					name: "invalid coordinates and unknown severity",
					fixture: `[{"ident":"negative","determinations":{"severity":"future"},
						"locations":[{"concrete":{"location":{"start_point":{"row":-2,"column":-3}}}}]},
						{"ident":"past-end","locations":[{"concrete":{"location":{"start_point":{"row":5}}}}]}]`,
					want: []scanfindings.Finding{
						{RuleID: "negative", Severity: scanfindings.SeverityUnknown, Message: "[future] negative", File: "display.yml", Line: -1, Column: -2},
						{RuleID: "past-end", Severity: scanfindings.SeverityUnknown, Message: "past-end", File: "display.yml", Line: 6, Column: 1},
					},
				},
				{
					name:     "valid location without source",
					fixture:  `[{"locations":[{"concrete":{"location":{"start_point":{"row":2,"column":4}}}}]}]`,
					noSource: true,
					want:     []scanfindings.Finding{{Severity: scanfindings.SeverityUnknown, File: "display.yml", Line: 3, Column: 5}},
				},
				{
					name: "input order and duplicates survive skipped findings",
					fixture: `[{"ident":"z","locations":[{}]},{"ident":"skipped"},
						{"ident":"a","determinations":{"severity":"Critical"},"locations":[{}]},
						{"ident":"z","locations":[{}]}]`,
					want: []scanfindings.Finding{
						{RuleID: "z", Severity: scanfindings.SeverityUnknown, Message: "z", File: "display.yml", Line: 1, Column: 1, Context: []string{"one"}},
						{RuleID: "a", Severity: scanfindings.SeverityCritical, Message: "[Critical] a", File: "display.yml", Line: 1, Column: 1, Context: []string{"one"}},
						{RuleID: "z", Severity: scanfindings.SeverityUnknown, Message: "z", File: "display.yml", Line: 1, Column: 1, Context: []string{"one"}},
					},
				},
			},
		},
		{
			name: "poutine", fixture: poutineNativeFindingsFixture,
			convert: func(t *testing.T, fixture string, fileLines []string) []scanfindings.Finding {
				var output poutineOutput
				require.NoError(t, json.Unmarshal([]byte(fixture), &output), "poutine fixture JSON should parse")
				return poutineFindingsToShared(output.Findings, output.Rules, "fallback.yml", fileLines)
			},
			want: []scanfindings.Finding{
				{RuleID: "known", Severity: scanfindings.SeverityHigh, Message: "[error] known: Rule title - extra detail", File: "native.yml", Line: 3, Column: 0, Context: lines},
				{RuleID: "missing", Severity: scanfindings.SeverityMedium, Message: "[warning] missing: missing", File: "fallback.yml", Line: 1, Column: 0, Context: []string{"one"}},
				{RuleID: "note", Severity: scanfindings.SeverityInfo, Message: "[note] note: note", File: "note.yml", Line: -1, Column: 0},
			},
			edges: []scannerFindingTestCase{
				{name: "nil input", fixture: `{"findings":null}`, want: []scanfindings.Finding{}},
				{name: "non-nil empty input", fixture: `{"findings":[]}`, want: []scanfindings.Finding{}},
				{
					name:    "partial finding and null rules use defaults",
					fixture: `{"findings":[{}],"rules":null}`,
					want:    []scanfindings.Finding{{Severity: scanfindings.SeverityMedium, Message: "[warning]", File: "fallback.yml", Line: 1, Column: 0, Context: []string{"one"}}},
				},
				{
					name: "invalid lines and unknown rule severity",
					fixture: `{"findings":[{"rule_id":"negative","meta":{"path":"first.yml","line":-2}},
						{"rule_id":"future","meta":{"path":"second.yml","line":6}}],
						"rules":{"negative":{},"future":{"level":"future"}}}`,
					want: []scanfindings.Finding{
						{RuleID: "negative", Severity: scanfindings.SeverityMedium, Message: "[warning] negative: negative", File: "first.yml", Line: -2, Column: 0},
						{RuleID: "future", Severity: scanfindings.SeverityUnknown, Message: "[future] future: future", File: "second.yml", Line: 6, Column: 0},
					},
				},
				{
					name:    "valid location without source",
					fixture: `{"findings":[{"meta":{"line":3}}]}`, noSource: true,
					want: []scanfindings.Finding{{Severity: scanfindings.SeverityMedium, Message: "[warning]", File: "fallback.yml", Line: 3, Column: 0}},
				},
				{
					name: "input order paths and duplicates are preserved",
					fixture: `{"findings":[{"rule_id":"z","meta":{"path":"first.yml"}},
						{"rule_id":"a","meta":{"path":"second.yml"}},{"rule_id":"z","meta":{"path":"first.yml"}}],
						"rules":{"a":{"level":"error"}}}`,
					want: []scanfindings.Finding{
						{RuleID: "z", Severity: scanfindings.SeverityMedium, Message: "[warning] z: z", File: "first.yml", Line: 1, Column: 0, Context: []string{"one"}},
						{RuleID: "a", Severity: scanfindings.SeverityHigh, Message: "[error] a: a", File: "second.yml", Line: 1, Column: 0, Context: []string{"one"}},
						{RuleID: "z", Severity: scanfindings.SeverityMedium, Message: "[warning] z: z", File: "first.yml", Line: 1, Column: 0, Context: []string{"one"}},
					},
				},
			},
		},
		{
			name: "grype", fixture: grypeNativeFindingsFixture,
			convert: func(t *testing.T, fixture string, _ []string) []scanfindings.Finding {
				var output grypeOutput
				require.NoError(t, json.Unmarshal([]byte(fixture), &output), "grype fixture JSON should parse")
				return grypeFindingsToShared("image:tag", output.Matches)
			},
			want: []scanfindings.Finding{
				{RuleID: "CVE-1", Severity: scanfindings.SeverityCritical, Message: "[Critical] CVE-1: lib@1.0 (fix: 1.2, 1.3) (https://example.com/CVE-1)", File: "image:tag", Line: 1, Column: 1},
				{RuleID: "CVE-2", Severity: scanfindings.SeverityUnknown, Message: "[Unknown] CVE-2: other@2.0", File: "image:tag", Line: 1, Column: 1},
				{RuleID: "CVE-3", Severity: scanfindings.SeverityLow, Message: "[Negligible] CVE-3: minor@3.0", File: "image:tag", Line: 1, Column: 1},
			},
			edges: []scannerFindingTestCase{
				{name: "nil input", fixture: `{"matches":null}`, want: []scanfindings.Finding{}},
				{name: "non-nil empty input", fixture: `{"matches":[]}`, want: []scanfindings.Finding{}},
				{
					name:    "partial match uses image location and unknown severity",
					fixture: `{"matches":[{}]}`,
					want:    []scanfindings.Finding{{Severity: scanfindings.SeverityUnknown, Message: "[Unknown] : @", File: "image:tag", Line: 1, Column: 1}},
				},
				{
					name:    "unknown severity and empty fix versions",
					fixture: `{"matches":[{"vulnerability":{"id":"CVE-4","severity":"future","fix":{"state":"fixed","versions":[]}}}]}`,
					want:    []scanfindings.Finding{{RuleID: "CVE-4", Severity: scanfindings.SeverityUnknown, Message: "[future] CVE-4: @", File: "image:tag", Line: 1, Column: 1}},
				},
				{
					name: "input order artifacts and duplicates are preserved",
					fixture: `{"matches":[{"vulnerability":{"id":"CVE-z"},"artifact":{"name":"first","version":"1"}},
						{"vulnerability":{"id":"CVE-a","severity":"Critical"},"artifact":{"name":"second","version":"2"}},
						{"vulnerability":{"id":"CVE-z"},"artifact":{"name":"first","version":"1"}}]}`,
					want: []scanfindings.Finding{
						{RuleID: "CVE-z", Severity: scanfindings.SeverityUnknown, Message: "[Unknown] CVE-z: first@1", File: "image:tag", Line: 1, Column: 1},
						{RuleID: "CVE-a", Severity: scanfindings.SeverityCritical, Message: "[Critical] CVE-a: second@2", File: "image:tag", Line: 1, Column: 1},
						{RuleID: "CVE-z", Severity: scanfindings.SeverityUnknown, Message: "[Unknown] CVE-z: first@1", File: "image:tag", Line: 1, Column: 1},
					},
				},
			},
		},
		{
			name: "runner-guard", fixture: runnerGuardNativeFindingsFixture,
			convert: func(t *testing.T, fixture string, fileLines []string) []scanfindings.Finding {
				var output runnerGuardOutput
				require.NoError(t, json.Unmarshal([]byte(fixture), &output), "runner-guard fixture JSON should parse")
				return runnerGuardFindingsToShared(output.Findings, fileLines)
			},
			want: []scanfindings.Finding{
				{RuleID: "RGS-001", Severity: scanfindings.SeverityHigh, Message: "[HIGH] RGS-001: Rule name - extra detail", File: "native.yml", Line: 3, Column: 0, Context: lines},
				{RuleID: "RGS-002", Severity: scanfindings.SeverityUnknown, Message: "RGS-002: Unnamed severity", File: "default.yml", Line: 1, Column: 0, Context: []string{"one"}},
				{RuleID: "RGS-003", Severity: scanfindings.SeverityLow, Message: "[low] RGS-003", File: "negative.yml", Line: -1, Column: 0},
			},
			edges: []scannerFindingTestCase{
				{name: "nil input", fixture: `{"findings":null}`, want: []scanfindings.Finding{}},
				{name: "non-nil empty input", fixture: `{"findings":[]}`, want: []scanfindings.Finding{}},
				{
					name:    "partial finding does not invent a file",
					fixture: `{"findings":[{}]}`,
					want:    []scanfindings.Finding{{Severity: scanfindings.SeverityUnknown, Line: 1, Column: 0, Context: []string{"one"}}},
				},
				{
					name: "invalid lines and unknown severity",
					fixture: `{"findings":[{"rule_id":"negative","file":"first.yml","line":-2},
						{"rule_id":"future","file":"second.yml","line":6,"severity":"future"}]}`,
					want: []scanfindings.Finding{
						{RuleID: "negative", Severity: scanfindings.SeverityUnknown, Message: "negative", File: "first.yml", Line: -2, Column: 0},
						{RuleID: "future", Severity: scanfindings.SeverityUnknown, Message: "[future] future", File: "second.yml", Line: 6, Column: 0},
					},
				},
				{
					name:    "valid location without source",
					fixture: `{"findings":[{"file":"native.yml","line":3}]}`, noSource: true,
					want: []scanfindings.Finding{{Severity: scanfindings.SeverityUnknown, File: "native.yml", Line: 3, Column: 0}},
				},
				{
					name: "input order paths and duplicates are preserved",
					fixture: `{"findings":[{"rule_id":"z","file":"first.yml"},
						{"rule_id":"a","file":"second.yml","severity":"HIGH"},{"rule_id":"z","file":"first.yml"}]}`,
					want: []scanfindings.Finding{
						{RuleID: "z", Severity: scanfindings.SeverityUnknown, Message: "z", File: "first.yml", Line: 1, Column: 0, Context: []string{"one"}},
						{RuleID: "a", Severity: scanfindings.SeverityHigh, Message: "[HIGH] a", File: "second.yml", Line: 1, Column: 0, Context: []string{"one"}},
						{RuleID: "z", Severity: scanfindings.SeverityUnknown, Message: "z", File: "first.yml", Line: 1, Column: 0, Context: []string{"one"}},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cases := append([]scannerFindingTestCase{{name: "native semantics", fixture: tt.fixture, want: tt.want}}, tt.edges...)
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					fileLines := lines
					if tc.noSource {
						fileLines = nil
					}
					got := tt.convert(t, tc.fixture, fileLines)
					require.NotNil(t, got, "converter should return a non-nil slice even when no findings remain")
					require.Len(t, got, len(tc.want), "converter should preserve finding count except for locationless zizmor findings")
					require.Equal(t, tc.want, got, "converter should preserve native defaults, messages, locations, order and duplicates")
				})
			}
		})
	}
}

type scannerFindingTestCase struct {
	name     string
	fixture  string
	noSource bool
	want     []scanfindings.Finding
}
