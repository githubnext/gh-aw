package workflow

import "fmt"

// GhAwHome is the shell expression for GH_AW_HOME.
// Use this in bash `run:` contexts where shell variable expansion occurs.
// The job-level env sets GH_AW_HOME to /opt/gh-aw by default.
const GhAwHome = "${GH_AW_HOME}"

// GhAwHomeJS is the JavaScript expression for GH_AW_HOME.
// Use this inside require() or other JS expressions in github-script steps.
// The job-level env sets GH_AW_HOME to /opt/gh-aw by default.
const GhAwHomeJS = "process.env.GH_AW_HOME"

// SetupActionDestination is the path where the setup action copies script files
// on the agent runner (e.g. /opt/gh-aw/actions).
// This is a shell expression expanded at runtime.
const SetupActionDestination = GhAwHome + "/actions"

// JsRequireGhAw generates a JavaScript require() argument expression for a file
// under GH_AW_HOME. The relativePath should be like "actions/foo.cjs".
func JsRequireGhAw(relativePath string) string {
	return fmt.Sprintf("%s + '/%s'", GhAwHomeJS, relativePath)
}
