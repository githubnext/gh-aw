package taint

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func nativeJavaScriptOptions(t *testing.T) *JavaScriptOptions {
	t.Helper()
	dir, err := filepath.Abs("javascript")
	require.NoError(t, err)
	if _, err := os.Stat(filepath.Join(dir, "node_modules", "@typescript", "native-preview", "package.json")); err != nil {
		t.Skip("native JS integration requires npm ci --prefix pkg/workflow/taint/javascript")
	}
	return &JavaScriptOptions{Context: t.Context(), Parser: &TypeScriptParser{ModuleDirectory: dir}}
}

func TestNativeJavaScriptTaint(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	secret := []Source{{Reference: "secrets.test_token", Path: "env.TOKEN"}}
	tests := []struct {
		name   string
		source string
		kind   Kind
		gap    bool
	}{
		{"stdin log", `const fs = require("fs"); console.log(fs.readFileSync(0, "utf8"));`, Log, false},
		{"stdin buffer conversion", `const {readFileSync} = require("node:fs"); console.log(readFileSync(0).toString("utf8"));`, Log, false},
		{"stdin JSON", `const fs=require("fs"); const obj=JSON.parse(fs.readFileSync(0,"utf8")); console.log(obj.token);`, Log, false},
		{"environment alias", `const value = process.env.TOKEN; console.error(value);`, Log, false},
		{"environment case sensitive", `console.log(process.env.token);`, "", false},
		{"computed environment", `const key = "TOKEN"; console.log(process.env[key]);`, Log, false},
		{"dynamic environment", `console.log(process.env[unknown]);`, Log, false},
		{"destructured environment", `const {TOKEN: token} = process.env; console.log(token);`, Log, false},
		{"object alias", `const data={credential:process.env.TOKEN}; const alias=data; console.log(alias.credential);`, Log, false},
		{"object safe member", `const data={credential:process.env.TOKEN, safe:"ok"}; console.log(data.safe);`, "", false},
		{"local function", `function transform(x){return JSON.stringify(x)} console.log(transform(process.env.TOKEN));`, Log, false},
		{"arrow callback", `process.stdin.on("data", chunk => console.log(chunk));`, Log, false},
		{"authentication", `fetch("https://example.test", {headers:{Authorization:process.env.TOKEN}});`, "", false},
		{"authentication and safe body", `fetch("https://example.test", {headers:{"x-api-key":process.env.TOKEN}, body:"ok"});`, "", false},
		{"network body", `fetch("https://example.test", {body:process.env.TOKEN});`, Network, false},
		{"network header", `fetch("https://example.test", {headers:{"x-data":process.env.TOKEN}});`, Network, false},
		{"network URL", `fetch("https://example.test/?token="+process.env.TOKEN);`, Network, false},
		{"mask is not sanitizer", `core.setSecret(process.env.TOKEN); console.log(process.env.TOKEN);`, Log, false},
		{"mask registration", `core.setSecret(process.env.TOKEN);`, "", false},
		{"predicate", `console.log(process.env.TOKEN !== "");`, "", false},
		{"boolean coercion", `console.log(Boolean(process.env.TOKEN));`, "", false},
		{"logical fallback", `console.log(process.env.TOKEN || "");`, Log, false},
		{"summary", `require("fs").appendFileSync(process.env.GITHUB_STEP_SUMMARY, process.env.TOKEN);`, Summary, false},
		{"output", `core.setOutput("token", process.env.TOKEN);`, Output, false},
		{"file roundtrip", `const fs=require("fs"); fs.writeFileSync("token.txt", process.env.TOKEN); console.log(fs.readFileSync("token.txt","utf8"));`, Log, false},
		{"unknown call", `unknown(process.env.TOKEN);`, "", true},
		{"unknown transform retains taint", `console.log(unknown(process.env.TOKEN));`, Log, true},
		{"unknown external module", `require("unavailable").consume(process.env.TOKEN);`, "", true},
		{"dynamic import", `require(process.env.MODULE).consume(process.env.TOKEN);`, "", true},
		{"function not executed", `function unused(){console.log(process.env.TOKEN)}`, "", false},
		{"never execute JavaScript", `require("child_process").execSync("false");`, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AnalyzeJavaScript(JavaScriptInput{
				Name: "entry.cjs", Source: tt.source, Stdin: secret, Env: map[string][]Source{"TOKEN": secret},
			}, SecretLeakRule(), *options)
			require.NoError(t, err)
			if tt.kind == "" {
				assert.Empty(t, result.Findings)
			} else {
				require.NotEmpty(t, result.Findings)
				assert.Equal(t, tt.kind, result.Findings[0].Site.Kind)
				assert.Equal(t, secret, result.Findings[0].Sources)
			}
			assert.Equal(t, tt.gap, len(result.Gaps) > 0, "%+v", result.Gaps)
		})
	}
}

func TestJavaScriptRepositoryModules(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	options.Files = fstest.MapFS{
		"helpers/consume.cjs": {Data: []byte(`exports.consume = token => fetch("https://example.test", {headers:{authorization: token}});`)},
		"helpers/leak.cjs":    {Data: []byte(`module.exports = {consume: token => console.error(token)};`)},
	}
	for _, module := range []string{"consume", "leak"} {
		input := JavaScriptInput{Name: "entry.cjs", Source: `require("./helpers/` + module + `.cjs").consume(process.env.TOKEN);`, Env: map[string][]Source{"TOKEN": {{Reference: "secrets.token", Path: "env.TOKEN"}}}}
		result, err := AnalyzeJavaScript(input, SecretLeakRule(), *options)
		require.NoError(t, err)
		assert.Empty(t, result.Gaps)
		assert.Equal(t, module == "leak", len(result.Findings) > 0, "%+v", result)
	}
}

func TestNativeJavaScriptParserFailures(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	for _, source := range []string{"const = ;", strings.Repeat("x", 1024*1024+1)} {
		result, err := AnalyzeJavaScript(JavaScriptInput{Name: "entry.cjs", Source: source}, SecretLeakRule(), *options)
		require.NoError(t, err)
		require.NotEmpty(t, result.Gaps)
		assert.Empty(t, result.Findings)
		assert.NotContains(t, result.Gaps[0].Reason, source[:8])
	}
	result, err := AnalyzeJavaScript(JavaScriptInput{Name: "entry.cjs"}, SecretLeakRule(), JavaScriptOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, result.Gaps)
	_, err = (&TypeScriptParser{ModuleDirectory: t.TempDir()}).Parse(context.Background(), "const token = 1;")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "const token")
}

func TestJavaScriptWorkflowEntryPoints(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	options.Files = fstest.MapFS{
		"helpers/leak.cjs": {Data: []byte(`const fs=require("fs"); console.log(fs.readFileSync(0,"utf8"));`)},
	}
	tests := []string{
		"      - env:\n          TOKEN: ${{ secrets.TOKEN }}\n        run: node -e 'console.log(process.env.TOKEN)'\n",
		"      - shell: node {0}\n        run: console.log('${{ secrets.TOKEN }}')\n",
		"      - env:\n          TOKEN: ${{ secrets.TOKEN }}\n        run: printf '%s' \"$TOKEN\" | node helpers/leak.cjs\n",
		"      - uses: actions/github-script@v9\n        with:\n          script: console.log('${{ secrets.TOKEN }}')\n",
	}
	for _, steps := range tests {
		workflow, err := actions.Parse([]byte("jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n" + steps))
		require.NoError(t, err)
		rule := SecretLeakRule()
		rule.JavaScript = options
		result, err := Analyze(workflow, rule)
		require.NoError(t, err)
		require.NotEmpty(t, result.Findings, "%s", steps)
		assert.Empty(t, result.Gaps)
	}
}
