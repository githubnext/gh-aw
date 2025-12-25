package workflow

import (
"github.com/githubnext/gh-aw/pkg/logger"
)

var jsLog = logger.New("workflow:js")

// init registers scripts from js.go with the DefaultScriptRegistry
// Note: Embedded scripts have been removed - scripts are now provided by actions/setup at runtime
func init() {
jsLog.Print("Script registration completed (embedded scripts removed)")
}

// Legacy getter functions - these return empty strings since embedded scripts were removed
// Scripts are now provided by the actions/setup action at runtime

func getAddReactionAndEditCommentScript() string {
return ""
}

func getAssignIssueScript() string {
return ""
}

func getAddCopilotReviewerScript() string {
return ""
}

func getCheckMembershipScript() string {
return ""
}

func getSafeOutputsMCPServerScript() string {
return ""
}

// GetJavaScriptSources returns an empty map since embedded scripts have been removed.
// Scripts are now provided by the actions/setup action at runtime.
func GetJavaScriptSources() map[string]string {
return map[string]string{}
}
