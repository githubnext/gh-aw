package taint

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShellValueFreeConsumers(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  string
		gaps int
		logs int
	}{
		{name: "colon presence guard", run: `: "${TOKEN:?credential required}"`},
		{name: "true discards arguments", run: `true "$TOKEN"`},
		{name: "false discards arguments", run: `false "$TOKEN"`},
		{name: "empty predicate", run: `test -z "${TOKEN:-}"`},
		{name: "nonempty predicate", run: `[ -n "$TOKEN" ]`},
		{name: "quoted predicate operator", run: `test "-n" "$TOKEN"`},
		{name: "single operand", run: `test "$TOKEN"`},
		{name: "negated predicate", run: `[ ! -z "$TOKEN" ]`},
		{name: "string comparison", run: `[ "$TOKEN" != literal ]`},
		{name: "file predicate", run: `test -e "$TOKEN"`},
		{name: "pipeline ignored by no-op", run: `printf '%s' "$TOKEN" | true`},
		{name: "pipeline ignored by predicate", run: `printf '%s' "$TOKEN" | test -n literal`},
		{name: "environment no-op", run: `env COPY="$TOKEN" : "$TOKEN"`},
		{name: "environment predicate", run: `env COPY="$TOKEN" test -n "$TOKEN"`},
		{name: "safe alternate", run: `printf '%s' "${TOKEN:+present}" "${TOKEN+present}"`},
		{name: "alternate retains nested taint", run: `echo "${TOKEN:+$TOKEN}"`, logs: 1},
		{name: "default retains original value", run: `echo "${TOKEN:-public}"`, logs: 1},
		{name: "guard error leaks replacement", run: `: "${MISSING:?$TOKEN}"`, logs: 1},
		{name: "no-op error leaks replacement", run: `true "${MISSING?$TOKEN}"`, logs: 1},
		{name: "predicate error leaks replacement", run: `test -n "${MISSING:?$TOKEN}"`, logs: 1},
		{name: "substitution result discarded", run: `: "$(printf '%s' "$TOKEN")"`},
		{name: "substitution stderr retained", run: `: "$(printf '%s' "$TOKEN" >&2)"`, logs: 1},
		{name: "predicate substitution stderr", run: `test -n "$(printf '%s' "$TOKEN" >&2)"`, logs: 1},
		{name: "xtrace no-op retained", run: `set -x; : "$TOKEN"`, logs: 1},
		{name: "xtrace predicate retained", run: `set -x; test -n "$TOKEN"`, logs: 1},
		{name: "following output retained", run: `: "$TOKEN"; test -n "$TOKEN"; echo "$TOKEN"`, logs: 1},
		{name: "overridden predicate", run: `test() { echo "$1"; }; test "$TOKEN"`, logs: 1},
		{name: "numeric predicate may print operand", run: `test "$TOKEN" -eq 1`, gaps: 1},
		{name: "unknown unary predicate", run: `[ -invalid "$TOKEN" ]`, gaps: 1},
		{name: "missing closing bracket", run: `[ -n "$TOKEN"`, gaps: 1},
		{name: "malformed predicate", run: `test "$TOKEN" unexpected extra`, gaps: 1},
		{name: "dynamic operator", run: `test "-n$(printf x)" "$TOKEN"`, gaps: 1},
		{name: "unquoted operand may change arity", run: `test -n $TOKEN`, gaps: 1},
		{name: "unquoted substituted operand", run: `test -n $(printf '%s' "$TOKEN")`, gaps: 1},
		{name: "glob may change arity", run: `test -n "$TOKEN" *`, gaps: 1},
		{name: "quoted array may change arity", run: `VALUES=("$TOKEN" other); test -n "${VALUES[@]}"`, gaps: 1},
		{name: "quoted positional array may change arity", run: `check() { test -n "$@"; }; check "$TOKEN" other`, gaps: 1},
		{name: "dynamic command is not a no-op", run: `true"$(printf x)" "$TOKEN"`, gaps: 1},
		{name: "dynamic command substitution stderr", run: `true"$(printf '%s' "$TOKEN" >&2)"`, logs: 1},
		{name: "secret command name", run: `"$TOKEN"`, gaps: 1},
		{name: "unknown consumer", run: `unknown "$TOKEN"`, gaps: 1},
		{name: "unknown stdin consumer", run: `printf '%s' "$TOKEN" | node helper.cjs`, gaps: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := checkSecretScript(t, tt.run)
			require.Len(t, result.Gaps, tt.gaps, "%+v", result.Gaps)
			require.Len(t, result.Findings, tt.logs, "%+v", result.Findings)
			for _, finding := range result.Findings {
				assert.Equal(t, Log, finding.Site.Kind)
			}
		})
	}
}

func TestShellCoverageGapLineAndDeduplication(t *testing.T) {
	result := checkSecretScript(t, "COPY=$TOKEN\nunknown \"$COPY\"\nunknown \"$TOKEN\"\n")
	require.Len(t, result.Gaps, 1)
	assert.Equal(t, "jobs.test.steps[0].run", result.Gaps[0].Path)
	assert.Equal(t, 2, result.Gaps[0].Line)
	assert.Equal(t, "external command data flow is not modeled", result.Gaps[0].Reason)
	assert.NotContains(t, result.Gaps[0].Reason, "TOKEN")
}

func checkSecretScript(t *testing.T, run string) Result {
	t.Helper()
	var steps strings.Builder
	steps.WriteString("      - env:\n          TOKEN: ${{ secrets.TEST_TOKEN }}\n        run: |\n")
	for line := range strings.SplitSeq(run, "\n") {
		steps.WriteString("          " + line + "\n")
	}
	return checkSteps(t, steps.String())
}
