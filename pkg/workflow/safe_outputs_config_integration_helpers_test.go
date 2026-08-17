//go:build integration

package workflow

import "strings"

// unescapeLockJSON converts the YAML-escaped JSON embedded in compiled lock files
// (for example `GH_AW_SAFE_OUTPUTS_CONFIG: "{\"group\":true}"`) back into plain JSON
// so that tests can assert on raw JSON fragments such as `"group":true`.
func unescapeLockJSON(compiled string) string {
	return strings.ReplaceAll(compiled, `\"`, `"`)
}
