package cli

import (
	"context"
	"testing"

	"github.com/github/gh-aw/pkg/intent"
	"github.com/github/gh-aw/pkg/intent/authz"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntentAuthorizationMiddlewareRejectsDeniedToolCall(t *testing.T) {
	compiler := intent.PolicyCompiler{Rules: []intent.PolicyRule{{
		ID: "deny-add",
		Set: intent.ExecutionPolicy{
			AllowedTools: []string{"compile", "add"},
			DeniedTools:  []string{"add"},
		},
	}}}
	middleware := intentAuthorizationMiddlewareForPolicy(compiler, (authz.Authorizer{}).AuthorizeTool)
	called := false
	handler := middleware(func(_ context.Context, _ string, _ mcp.Request) (mcp.Result, error) {
		called = true
		return &mcp.CallToolResult{}, nil
	})

	result, err := handler(context.Background(), "tools/call", fakeToolCallRequest("add"))

	require.NoError(t, err)
	assert.False(t, called)
	toolResult := result.(*mcp.CallToolResult)
	assert.True(t, toolResult.IsError)
	assert.Contains(t, toolResult.Content[0].(*mcp.TextContent).Text, "tool denied")
}

func TestIntentAuthorizationMiddlewareReadsPolicyAtExecutionTime(t *testing.T) {
	autoMerge := false
	compiler := intent.PolicyCompiler{Rules: []intent.PolicyRule{{
		ID: "runtime-fields",
		Set: intent.ExecutionPolicy{
			AllowedTools:     []string{"merge_pull_request"},
			AutoMergeAllowed: &autoMerge,
			MaxAttempts:      2,
		},
	}}}

	var observedPolicy intent.ExecutionPolicy
	middleware := intentAuthorizationMiddlewareForPolicy(compiler, func(policy intent.ExecutionPolicy, _ string, _ authz.ToolContext) error {
		observedPolicy = policy
		return nil
	})
	handler := middleware(func(_ context.Context, _ string, _ mcp.Request) (mcp.Result, error) {
		return &mcp.CallToolResult{}, nil
	})

	_, err := handler(context.Background(), "tools/call", fakeToolCallRequest("merge_pull_request"))

	require.NoError(t, err)
	assert.Equal(t, 2, observedPolicy.MaxAttempts)
	require.NotNil(t, observedPolicy.AutoMergeAllowed)
	assert.False(t, *observedPolicy.AutoMergeAllowed)
}
