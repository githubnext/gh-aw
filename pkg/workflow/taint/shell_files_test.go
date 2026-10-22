package taint

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileCoverageRequiresSecretFlow(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  string
		gaps int
	}{
		{name: "untainted dynamic input", run: `cat "$UNKNOWN"`},
		{name: "untainted dynamic input redirection", run: `cat < "$UNKNOWN"`},
		{name: "untainted dynamic output", run: `echo public > "$UNKNOWN"`},
		{name: "untainted file does not introduce a source", run: `echo public > known; cat "$UNKNOWN"`},
		{name: "unrelated malformed pattern", run: `echo public > known; jq '[' known`},
		{name: "secret-bearing dynamic output", run: `printf '%s' "$TOKEN" > "$UNKNOWN"`, gaps: 1},
		{name: "secret-bearing dynamic tee", run: `printf '%s' "$TOKEN" | tee "$UNKNOWN" >/dev/null`, gaps: 1},
		{name: "possible secret file input", run: `printf '%s' "$TOKEN" > known; cat "$UNKNOWN" >/dev/null`, gaps: 1},
		{name: "possible secret redirected input", run: `printf '%s' "$TOKEN" > known; cat < "$UNKNOWN" >/dev/null`, gaps: 1},
		{name: "possible secret pattern input", run: `printf '%s' "$TOKEN" > known; cat '[' >/dev/null`, gaps: 1},
		{name: "secret-bearing input filename", run: `cat < "$TOKEN" >/dev/null`, gaps: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := checkSecretScript(t, tt.run)
			require.Len(t, result.Gaps, tt.gaps, "%+v", result.Gaps)
			require.Empty(t, result.Findings)
		})
	}
}

func TestFileInputPreservesExpansionDisclosures(t *testing.T) {
	result := checkSecretScript(t, `printf '%s' "$TOKEN" > known; cat < "$(printf '%s' "$TOKEN" >&2)" >/dev/null`)
	require.NotEmpty(t, result.Findings)
	require.NotEmpty(t, result.Gaps)
}
