package workflow

import (
"encoding/json"
"testing"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

func TestGenerateSafeOutputsConfigIncludesTypeField(t *testing.T) {
workflowData := &WorkflowData{
SafeOutputs: &SafeOutputsConfig{
CreateIssues: &CreateIssuesConfig{
BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 5},
},
AddComments: &AddCommentsConfig{
BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 3},
Target:               "triggering",
},
CreateDiscussions: &CreateDiscussionsConfig{
BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 2},
},
},
}

configJSON := generateSafeOutputsConfig(workflowData)
require.NotEmpty(t, configJSON, "Config JSON should not be empty")

// Parse the JSON
var config map[string]any
err := json.Unmarshal([]byte(configJSON), &config)
require.NoError(t, err, "Should be valid JSON")

// Check create_issue has type field
createIssue, ok := config["create_issue"].(map[string]any)
require.True(t, ok, "create_issue should be present and be a map")
assert.Equal(t, "create-issue", createIssue["type"], "create_issue should have type field set to 'create-issue'")
assert.Equal(t, float64(5), createIssue["max"], "create_issue should have max field")

// Check add_comment has type field
addComment, ok := config["add_comment"].(map[string]any)
require.True(t, ok, "add_comment should be present and be a map")
assert.Equal(t, "add-comment", addComment["type"], "add_comment should have type field set to 'add-comment'")
assert.Equal(t, "triggering", addComment["target"], "add_comment should have target field")
assert.Equal(t, float64(3), addComment["max"], "add_comment should have max field")

// Check create_discussion has type field
createDiscussion, ok := config["create_discussion"].(map[string]any)
require.True(t, ok, "create_discussion should be present and be a map")
assert.Equal(t, "create-discussion", createDiscussion["type"], "create_discussion should have type field set to 'create-discussion'")
assert.Equal(t, float64(2), createDiscussion["max"], "create_discussion should have max field")
}
