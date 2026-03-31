//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePortSpec(t *testing.T) {
	tests := []struct {
		name          string
		spec          any
		expectedPorts []int
		warnContains  string
	}{
		{
			name:          "explicit host:container mapping",
			spec:          "5432:5432",
			expectedPorts: []int{5432},
		},
		{
			name:          "container port only (dynamic host)",
			spec:          "5432",
			expectedPorts: []int{5432},
		},
		{
			name:          "remapped host port",
			spec:          "49152:5432",
			expectedPorts: []int{5432},
		},
		{
			name:          "explicit TCP protocol",
			spec:          "5432/tcp",
			expectedPorts: []int{5432},
		},
		{
			name:          "UDP port skipped",
			spec:          "5432/udp",
			expectedPorts: nil,
			warnContains:  "UDP",
		},
		{
			name:          "integer port spec",
			spec:          5432,
			expectedPorts: []int{5432},
		},
		{
			name:          "float64 port spec (YAML parsing)",
			spec:          float64(5432),
			expectedPorts: []int{5432},
		},
		{
			name:          "port range",
			spec:          "6000-6002:6000-6002",
			expectedPorts: []int{6000, 6001, 6002},
		},
		{
			name:          "port range container only",
			spec:          "6000-6002",
			expectedPorts: []int{6000, 6001, 6002},
		},
		{
			name:          "host:container with TCP suffix",
			spec:          "5432:5432/tcp",
			expectedPorts: []int{5432},
		},
		{
			name:          "invalid port number",
			spec:          "abc",
			expectedPorts: nil,
			warnContains:  "invalid port number",
		},
		{
			name:          "invalid port range (end < start)",
			spec:          "6010-6000",
			expectedPorts: nil,
			warnContains:  "end < start",
		},
		{
			name:          "port range exceeding cap",
			spec:          "1000-2000",
			expectedPorts: nil,
			warnContains:  "exceeding cap",
		},
		{
			name:          "unsupported type",
			spec:          true,
			expectedPorts: nil,
			warnContains:  "unsupported port spec type",
		},
		{
			name:          "empty string",
			spec:          "",
			expectedPorts: nil,
		},
		{
			name:          "port zero is out of range",
			spec:          "0",
			expectedPorts: nil,
			warnContains:  "outside valid range",
		},
		{
			name:          "port above 65535",
			spec:          "70000",
			expectedPorts: nil,
			warnContains:  "outside valid range",
		},
		{
			name:          "integer port zero is out of range",
			spec:          0,
			expectedPorts: nil,
			warnContains:  "outside valid range",
		},
		{
			name:          "unknown protocol skipped",
			spec:          "5432/sctp",
			expectedPorts: nil,
			warnContains:  "unsupported protocol",
		},
		{
			name:          "float64 non-integer rejected",
			spec:          float64(5432.5),
			expectedPorts: nil,
			warnContains:  "not an integer",
		},
		{
			name:          "port range with out-of-range values",
			spec:          "65530-65540",
			expectedPorts: nil,
			warnContains:  "outside valid range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ports, warnings := parsePortSpec(tt.spec)
			assert.Equal(t, tt.expectedPorts, ports)

			if tt.warnContains != "" {
				require.NotEmpty(t, warnings, "expected a warning containing %q", tt.warnContains)
				found := false
				for _, w := range warnings {
					if strings.Contains(w, tt.warnContains) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected warning containing %q, got %v", tt.warnContains, warnings)
			}
		})
	}
}

func TestExtractServicePortExpressions(t *testing.T) {
	tests := []struct {
		name             string
		servicesYAML     string
		expectedResult   string
		expectedWarnings []string
	}{
		{
			name:           "empty services YAML",
			servicesYAML:   "",
			expectedResult: "",
		},
		{
			name: "single service with single port",
			servicesYAML: `services:
  postgres:
    image: postgres:15
    ports:
      - 5432:5432
`,
			expectedResult: "${{ job.services['postgres'].ports['5432'] }}",
		},
		{
			name: "multiple services with ports",
			servicesYAML: `services:
  postgres:
    image: postgres:15
    ports:
      - 5432:5432
  redis:
    image: redis:7
    ports:
      - 6379:6379
`,
			expectedResult: "${{ job.services['postgres'].ports['5432'] }},${{ job.services['redis'].ports['6379'] }}",
		},
		{
			name: "service with multiple ports",
			servicesYAML: `services:
  mydb:
    image: mydb:latest
    ports:
      - 5432:5432
      - 8080:8080
`,
			expectedResult: "${{ job.services['mydb'].ports['5432'] }},${{ job.services['mydb'].ports['8080'] }}",
		},
		{
			name: "service without ports emits warning",
			servicesYAML: `services:
  postgres:
    image: postgres:15
`,
			expectedResult:   "",
			expectedWarnings: []string{"service \"postgres\" has no ports mapping"},
		},
		{
			name: "mixed services: some with ports, some without",
			servicesYAML: `services:
  postgres:
    image: postgres:15
    ports:
      - 5432:5432
  redis:
    image: redis:7
`,
			expectedResult:   "${{ job.services['postgres'].ports['5432'] }}",
			expectedWarnings: []string{"service \"redis\" has no ports mapping"},
		},
		{
			name: "UDP port skipped with warning",
			servicesYAML: `services:
  myservice:
    image: myservice:latest
    ports:
      - 5432:5432/udp
`,
			expectedResult:   "",
			expectedWarnings: []string{"UDP"},
		},
		{
			name: "port range expansion",
			servicesYAML: `services:
  myservice:
    image: myservice:latest
    ports:
      - 6000-6002:6000-6002
`,
			expectedResult: "${{ job.services['myservice'].ports['6000'] }},${{ job.services['myservice'].ports['6001'] }},${{ job.services['myservice'].ports['6002'] }}",
		},
		{
			name: "dynamic host port (container port only)",
			servicesYAML: `services:
  postgres:
    image: postgres:15
    ports:
      - 5432
`,
			expectedResult: "${{ job.services['postgres'].ports['5432'] }}",
		},
		{
			name: "remapped host port uses container port in expression",
			servicesYAML: `services:
  postgres:
    image: postgres:15
    ports:
      - 49152:5432
`,
			expectedResult: "${{ job.services['postgres'].ports['5432'] }}",
		},
		{
			name:           "invalid YAML returns empty",
			servicesYAML:   "not: valid: yaml: [",
			expectedResult: "",
		},
		{
			name: "integer port values",
			servicesYAML: `services:
  postgres:
    image: postgres:15
    ports:
      - 5432
`,
			expectedResult: "${{ job.services['postgres'].ports['5432'] }}",
		},
		{
			name: "hyphenated service ID",
			servicesYAML: `services:
  my-postgres:
    image: postgres:15
    ports:
      - 5432:5432
`,
			expectedResult: "${{ job.services['my-postgres'].ports['5432'] }}",
		},
		{
			name: "invalid ports format (not a list) emits warning",
			servicesYAML: `services:
  postgres:
    image: postgres:15
    ports: 5432
`,
			expectedResult:   "",
			expectedWarnings: []string{"invalid ports mapping"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, warnings := ExtractServicePortExpressions(tt.servicesYAML)
			assert.Equal(t, tt.expectedResult, result)

			if tt.expectedWarnings != nil {
				for _, expectedWarning := range tt.expectedWarnings {
					found := false
					for _, w := range warnings {
						if strings.Contains(w, expectedWarning) {
							found = true
							break
						}
					}
					assert.True(t, found, "expected warning containing %q, got %v", expectedWarning, warnings)
				}
			}
		})
	}
}

func TestExtractServicePortExpressions_DeterministicOrder(t *testing.T) {
	// Run multiple times to verify deterministic ordering
	servicesYAML := `services:
  zeta:
    image: zeta:latest
    ports:
      - 1111:1111
  alpha:
    image: alpha:latest
    ports:
      - 2222:2222
  middle:
    image: middle:latest
    ports:
      - 3333:3333
`
	expected := "${{ job.services['alpha'].ports['2222'] }},${{ job.services['middle'].ports['3333'] }},${{ job.services['zeta'].ports['1111'] }}"

	for i := range 10 {
		result, _ := ExtractServicePortExpressions(servicesYAML)
		assert.Equal(t, expected, result, "iteration %d: order should be deterministic (alphabetical by service ID)", i)
	}
}
