//go:build !integration

package workflow

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestBundleAgenticCommandsScript(t *testing.T) {
	scripts := fstest.MapFS{
		"main.cjs":       {Data: []byte(`const { value } = require("./dependency.cjs"); module.exports = { main: async () => value };`)},
		"dependency.cjs": {Data: []byte(`module.exports = { value: "ok" };`)},
	}

	script, err := bundleAgenticCommandsScript(scripts, "main.cjs")
	require.NoError(t, err)
	require.Contains(t, script, `"dependency.cjs": (module, exports, require) => {`)
	require.Contains(t, script, `"main.cjs": (module, exports, require) => {`)
	require.Contains(t, script, `const __ghAwModuleCache = Object.create(null);`)
	require.Contains(t, script, `const { main: __ghAwMain } = __ghAwRequire("./main.cjs");`)
	require.Contains(t, script, `await __ghAwMain();`)
}

func TestBundleAgenticCommandsScriptFailsForMissingDependency(t *testing.T) {
	scripts := fstest.MapFS{
		"main.cjs": {Data: []byte(`require("./missing.cjs");`)},
	}

	_, err := bundleAgenticCommandsScript(scripts, "main.cjs")
	require.ErrorContains(t, err, `failed to read agentic commands module "missing.cjs"`)
}

func TestGetAgenticCommandsScript(t *testing.T) {
	script, err := getAgenticCommandsScript()
	require.NoError(t, err)
	require.Contains(t, script, `"route_slash_command.cjs": (module, exports, require) => {`)
	require.Contains(t, script, `"add_workflow_run_comment.cjs": (module, exports, require) => {`)
	require.Contains(t, script, `async function main()`)
	require.Contains(t, script, `await __ghAwMain();`)
}
