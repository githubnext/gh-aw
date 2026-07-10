//go:build !integration

package workflow

import (
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	formalErrorRepoNotAllowed    = -32001
	formalErrorInsufficientRole  = -32002
	formalErrorPrivateRepoDenied = -32003
	formalErrorBlockedUser       = -32004
	formalErrorToolNotAllowed    = -32005
	formalErrorIntegrityTooLow   = -32006
)

type formalToolConfig struct {
	Repos        []string
	Roles        []string
	PrivateRepos *bool
	AllowedTools []string
	BlockedUsers []string
	MinIntegrity string
}

type formalAccessRequest struct {
	Repository       string
	UserRole         string
	IsPrivate        bool
	ToolName         string
	UserLogin        string
	ContentIntegrity string
}

func TestFormal_ExactMatchAllow(t *testing.T) {
	allowed := formalEvaluateAccess(formalToolConfig{Repos: []string{"github/gh-aw"}}, formalAccessRequest{Repository: "github/gh-aw"})
	denied := formalEvaluateAccess(formalToolConfig{Repos: []string{"github/gh-aw"}}, formalAccessRequest{Repository: "github/other"})

	assert.True(t, allowed.allow)
	assert.False(t, denied.allow)
	assert.Equal(t, formalErrorRepoNotAllowed, denied.errorCode)
}

func TestFormal_WildcardMatch(t *testing.T) {
	assert.True(t, formalEvaluateAccess(formalToolConfig{Repos: []string{"github/*"}}, formalAccessRequest{Repository: "github/gh-aw"}).allow)
	assert.True(t, formalEvaluateAccess(formalToolConfig{Repos: []string{"github/gh-*"}}, formalAccessRequest{Repository: "github/gh-aw"}).allow)
	assert.False(t, formalEvaluateAccess(formalToolConfig{Repos: []string{"github/*"}}, formalAccessRequest{Repository: "microsoft/vscode"}).allow)
}

func TestFormal_EmptyReposDenyAll(t *testing.T) {
	assert.False(t, formalEvaluateAccess(formalToolConfig{}, formalAccessRequest{Repository: "github/gh-aw"}).allow)
	assert.False(t, formalEvaluateAccess(formalToolConfig{Repos: []string{}}, formalAccessRequest{Repository: "github/gh-aw"}).allow)
}

func TestFormal_RoleFilter(t *testing.T) {
	cfg := formalToolConfig{Repos: []string{"*/*"}, Roles: []string{"write", "admin"}}
	assert.True(t, formalEvaluateAccess(cfg, formalAccessRequest{Repository: "github/gh-aw", UserRole: "write"}).allow)
	denied := formalEvaluateAccess(cfg, formalAccessRequest{Repository: "github/gh-aw", UserRole: "read"})
	assert.False(t, denied.allow)
	assert.Equal(t, formalErrorInsufficientRole, denied.errorCode)
}

func TestFormal_PrivateRepoControl(t *testing.T) {
	allowPrivate := true
	denyPrivate := false

	assert.True(t, formalEvaluateAccess(formalToolConfig{Repos: []string{"myorg/*"}, PrivateRepos: &allowPrivate}, formalAccessRequest{Repository: "myorg/private", IsPrivate: true}).allow)
	assert.True(t, formalEvaluateAccess(formalToolConfig{Repos: []string{"myorg/*"}, PrivateRepos: &denyPrivate}, formalAccessRequest{Repository: "myorg/public", IsPrivate: false}).allow)
	denied := formalEvaluateAccess(formalToolConfig{Repos: []string{"myorg/*"}, PrivateRepos: &denyPrivate}, formalAccessRequest{Repository: "myorg/private", IsPrivate: true})
	assert.False(t, denied.allow)
	assert.Equal(t, formalErrorPrivateRepoDenied, denied.errorCode)
}

func TestFormal_BlockedUserDeny(t *testing.T) {
	cfg := formalToolConfig{Repos: []string{"github/gh-aw"}, Roles: []string{"write"}, BlockedUsers: []string{"bad-actor"}}
	assert.True(t, formalEvaluateAccess(cfg, formalAccessRequest{
		Repository: "github/gh-aw", UserRole: "write", UserLogin: "good-actor", ContentIntegrity: "approved",
	}).allow)
	denied := formalEvaluateAccess(cfg, formalAccessRequest{
		Repository: "github/gh-aw", UserRole: "write", UserLogin: "bad-actor", ContentIntegrity: "approved",
	})
	assert.False(t, denied.allow)
	assert.Equal(t, formalErrorBlockedUser, denied.errorCode)
}

func TestFormal_ToolNameFilter(t *testing.T) {
	cfg := formalToolConfig{Repos: []string{"*/*"}, AllowedTools: []string{"issue_read"}}
	assert.True(t, formalEvaluateAccess(cfg, formalAccessRequest{Repository: "github/gh-aw", ToolName: "issue_read"}).allow)
	assert.True(t, formalEvaluateAccess(formalToolConfig{Repos: []string{"*/*"}}, formalAccessRequest{Repository: "github/gh-aw", ToolName: "delete_repo"}).allow)
	denied := formalEvaluateAccess(cfg, formalAccessRequest{Repository: "github/gh-aw", ToolName: "delete_repo"})
	assert.False(t, denied.allow)
	assert.Equal(t, formalErrorToolNotAllowed, denied.errorCode)
}

func TestFormal_IntegrityLevelOrder(t *testing.T) {
	assert.True(t, formalEvaluateAccess(formalToolConfig{Repos: []string{"*/*"}, MinIntegrity: "approved"}, formalAccessRequest{Repository: "github/gh-aw", ContentIntegrity: "approved"}).allow)
	assert.True(t, formalEvaluateAccess(formalToolConfig{Repos: []string{"*/*"}, MinIntegrity: "approved"}, formalAccessRequest{Repository: "github/gh-aw", ContentIntegrity: "merged"}).allow)
	denied := formalEvaluateAccess(formalToolConfig{Repos: []string{"*/*"}, MinIntegrity: "approved"}, formalAccessRequest{Repository: "github/gh-aw", ContentIntegrity: "unapproved"})
	assert.False(t, denied.allow)
	assert.Equal(t, formalErrorIntegrityTooLow, denied.errorCode)
}

func TestFormal_CombinedFiltersAllAllow(t *testing.T) {
	allowPrivate := true
	cfg := formalToolConfig{
		Repos:        []string{"github/gh-aw"},
		Roles:        []string{"write"},
		PrivateRepos: &allowPrivate,
		AllowedTools: []string{"issue_read"},
		MinIntegrity: "approved",
	}
	assert.True(t, formalEvaluateAccess(cfg, formalAccessRequest{
		Repository: "github/gh-aw", UserRole: "write", IsPrivate: true, ToolName: "issue_read", UserLogin: "good-user", ContentIntegrity: "approved",
	}).allow)
}

func TestFormal_ErrorCodeFirstFailingGuard(t *testing.T) {
	denyPrivate := false
	cfg := formalToolConfig{
		Repos:        []string{"github/gh-aw"},
		Roles:        []string{"write"},
		PrivateRepos: &denyPrivate,
		AllowedTools: []string{"issue_read"},
		MinIntegrity: "approved",
	}

	denied := formalEvaluateAccess(cfg, formalAccessRequest{
		Repository: "github/other", UserRole: "read", IsPrivate: true, ToolName: "delete_repo", UserLogin: "good-user", ContentIntegrity: "none",
	})

	assert.False(t, denied.allow)
	assert.Equal(t, formalErrorRepoNotAllowed, denied.errorCode)
}

func TestFormal_BlockedUserSafetyProperty(t *testing.T) {
	denyPrivate := false
	cfg := formalToolConfig{
		Repos:        []string{"*/*"},
		Roles:        []string{"admin"},
		PrivateRepos: &denyPrivate,
		AllowedTools: []string{"issue_read"},
		BlockedUsers: []string{"blocked"},
		MinIntegrity: "merged",
	}

	denied := formalEvaluateAccess(cfg, formalAccessRequest{
		Repository: "any/repo", UserRole: "read", IsPrivate: true, ToolName: "delete_repo", UserLogin: "blocked", ContentIntegrity: "none",
	})

	assert.False(t, denied.allow)
	assert.Equal(t, formalErrorBlockedUser, denied.errorCode)
}

func TestFormal_NoSpuriousAllowInvariant(t *testing.T) {
	allowPrivate := true
	cfg := formalToolConfig{
		Repos:        []string{"github/gh-aw"},
		Roles:        []string{"write"},
		PrivateRepos: &allowPrivate,
		AllowedTools: []string{"issue_read"},
		BlockedUsers: []string{"blocked"},
		MinIntegrity: "approved",
	}

	cases := []formalAccessRequest{
		{Repository: "github/other", UserRole: "write", ToolName: "issue_read", ContentIntegrity: "approved"},
		{Repository: "github/gh-aw", UserRole: "read", ToolName: "issue_read", ContentIntegrity: "approved"},
		{Repository: "github/gh-aw", UserRole: "write", ToolName: "delete_repo", ContentIntegrity: "approved"},
		{Repository: "github/gh-aw", UserRole: "write", ToolName: "issue_read", ContentIntegrity: "none"},
		{Repository: "github/gh-aw", UserRole: "write", UserLogin: "blocked", ToolName: "issue_read", ContentIntegrity: "approved"},
	}

	for _, req := range cases {
		assert.False(t, formalEvaluateAccess(cfg, req).allow)
	}
}

type formalDecision struct {
	allow     bool
	errorCode int
}

func formalEvaluateAccess(cfg formalToolConfig, req formalAccessRequest) formalDecision {
	if containsExact(cfg.BlockedUsers, req.UserLogin) {
		return formalDecision{errorCode: formalErrorBlockedUser}
	}
	if !formalRepositoryAllowed(cfg.Repos, req.Repository) {
		return formalDecision{errorCode: formalErrorRepoNotAllowed}
	}
	if len(cfg.Roles) > 0 && !containsExact(cfg.Roles, req.UserRole) {
		return formalDecision{errorCode: formalErrorInsufficientRole}
	}
	if cfg.PrivateRepos != nil && !*cfg.PrivateRepos && req.IsPrivate {
		return formalDecision{errorCode: formalErrorPrivateRepoDenied}
	}
	if len(cfg.AllowedTools) > 0 && req.ToolName != "" && !containsExact(cfg.AllowedTools, req.ToolName) {
		return formalDecision{errorCode: formalErrorToolNotAllowed}
	}
	if cfg.MinIntegrity != "" && formalIntegrityRank(req.ContentIntegrity) < formalIntegrityRank(cfg.MinIntegrity) {
		return formalDecision{errorCode: formalErrorIntegrityTooLow}
	}
	return formalDecision{allow: true}
}

func formalRepositoryAllowed(patterns []string, repository string) bool {
	if len(patterns) == 0 || repository == "" {
		return false
	}
	repoOwner, repoName, ok := strings.Cut(repository, "/")
	if !ok || repoOwner == "" || repoName == "" {
		return false
	}

	for _, pattern := range patterns {
		patternOwner, patternRepo, ok := strings.Cut(pattern, "/")
		if !ok {
			continue
		}
		switch {
		case patternOwner == "*" && patternRepo == "*":
			return true
		case patternOwner == repoOwner && patternRepo == "*":
			return true
		case patternOwner == repoOwner && patternRepo == repoName:
			return true
		case patternOwner == repoOwner && strings.Contains(patternRepo, "*"):
			matched, _ := path.Match(patternRepo, repoName)
			if matched {
				return true
			}
		}
	}
	return false
}

func formalIntegrityRank(level string) int {
	switch strings.ToLower(level) {
	case "none":
		return 0
	case "unapproved":
		return 1
	case "approved":
		return 2
	case "merged":
		return 3
	default:
		return -1
	}
}

func containsExact(values []string, needle string) bool {
	return slices.Contains(values, needle)
}
