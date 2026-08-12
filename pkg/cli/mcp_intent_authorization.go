package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/intent"
	"github.com/github/gh-aw/pkg/intent/authz"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	intentPolicyEnforcementEnv = "GH_AW_INTENT_POLICY_ENFORCEMENT"
	intentPolicyPathEnv        = "GH_AW_INTENT_POLICY_PATH"
	intentLabelsEnv            = "GH_AW_INTENT_LABELS"
	intentHumanApprovedEnv     = "GH_AW_INTENT_HUMAN_APPROVED"
	intentPassedChecksEnv      = "GH_AW_INTENT_REQUIRED_CHECKS_PASSED"
	intentAttemptEnv           = "GH_AW_INTENT_ATTEMPT"
	intentDefaultBranchEnv     = "GH_AW_INTENT_DEFAULT_BRANCH"
)

var mcpIntentAuthzLog = logger.New("mcp:intent_authorization")

type intentPolicyFile struct {
	Rules []intent.PolicyRule `json:"rules"`
}

func intentPolicyEnforcementEnabled() bool {
	return os.Getenv(intentPolicyEnforcementEnv) == "true"
}

func intentAuthorizationMiddleware() mcp.Middleware {
	compiler, err := loadIntentPolicyCompiler()
	if err != nil {
		mcpIntentAuthzLog.Printf("intent policy enforcement disabled: %v", err)
		return passthroughMiddleware
	}
	authorizer := authz.Authorizer{}
	return intentAuthorizationMiddlewareForPolicy(compiler, authorizer.AuthorizeTool)
}

func intentAuthorizationMiddlewareForPolicy(compiler intent.PolicyCompiler, authorize func(intent.ExecutionPolicy, string, authz.ToolContext) error) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}
			toolName := extractMCPToolName(req)
			policy := compiler.Compile(intent.IntentRecord{
				Status: intent.AttributionMapped,
				Labels: splitCSVEnv(intentLabelsEnv),
			}, currentRepositoryContext())
			if err := authorize(policy, toolName, toolContextForMCPTool(toolName)); err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				}, nil
			}
			return next(ctx, method, req)
		}
	}
}

func passthroughMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return next
}

func loadIntentPolicyCompiler() (intent.PolicyCompiler, error) {
	path := os.Getenv(intentPolicyPathEnv)
	if path == "" {
		path = filepath.Join(".github", "intent-policy.json")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return intent.PolicyCompiler{}, err
	}
	var cfg intentPolicyFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return intent.PolicyCompiler{}, err
	}
	if len(cfg.Rules) == 0 {
		return intent.PolicyCompiler{}, errors.New("intent policy has no rules")
	}
	return intent.PolicyCompiler{Rules: cfg.Rules}, nil
}

func currentRepositoryContext() intent.RepositoryContext {
	repo := os.Getenv("GITHUB_REPOSITORY")
	owner, name, _ := strings.Cut(repo, "/")
	return intent.RepositoryContext{Owner: owner, Org: owner, Name: name}
}

func toolContextForMCPTool(toolName string) authz.ToolContext {
	return authz.ToolContext{
		IsWrite:       isIntentWriteTool(toolName),
		IsAutoMerge:   toolName == "merge_pull_request",
		Branch:        currentBranch(),
		DefaultBranch: os.Getenv(intentDefaultBranchEnv),
		Approved:      os.Getenv(intentHumanApprovedEnv) == "true",
		PassedChecks:  splitCSVEnv(intentPassedChecksEnv),
		Attempt:       intentAttempt(),
	}
}

func isIntentWriteTool(toolName string) bool {
	switch toolName {
	case "add", "update", "fix", "merge_pull_request", "create_or_update_file", "push_files", "delete_file":
		return true
	default:
		return false
	}
}

func intentAttempt() int {
	raw := os.Getenv(intentAttemptEnv)
	if raw == "" {
		return 1
	}
	attempt, err := strconv.Atoi(raw)
	if err != nil || attempt < 1 {
		return 1
	}
	return attempt
}

func splitCSVEnv(name string) []string {
	raw := os.Getenv(name)
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func currentBranch() string {
	for _, name := range []string{"GITHUB_HEAD_REF", "GITHUB_REF_NAME"} {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
