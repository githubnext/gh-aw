//go:build !integration

package workflow

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type formalIntegrityItem struct {
	AuthorLogin   string
	BaseIntegrity string
	Labels        []string
}

type formalIntegrityGuardConfig struct {
	BlockedUsers   []string
	TrustedUsers   []string
	ApprovalLabels []string
	MinIntegrity   string
}

type formalIntegrityDecision struct {
	allow     bool
	errorCode int
}

func TestFormal_BlockedTerminatesElevation(t *testing.T) {
	cfg := formalIntegrityGuardConfig{
		BlockedUsers:   []string{"bad-actor"},
		TrustedUsers:   []string{"bad-actor"},
		ApprovalLabels: []string{"human-reviewed"},
		MinIntegrity:   "none",
	}
	item := formalIntegrityItem{AuthorLogin: "bad-actor", BaseIntegrity: "none", Labels: []string{"human-reviewed"}}

	effective := formalEffectiveIntegrity(item, cfg)
	decision := formalIntegrityAccessDecision(item, cfg)

	assert.Equal(t, "blocked", effective)
	assert.False(t, decision.allow)
	assert.Equal(t, formalErrorBlockedUser, decision.errorCode)
}

func TestFormal_TrustedUserElevatesToApproved(t *testing.T) {
	cfg := formalIntegrityGuardConfig{TrustedUsers: []string{"trusted-user"}}

	assert.Equal(t, "approved", formalEffectiveIntegrity(formalIntegrityItem{AuthorLogin: "trusted-user", BaseIntegrity: "none"}, cfg))
	assert.Equal(t, "approved", formalEffectiveIntegrity(formalIntegrityItem{AuthorLogin: "trusted-user", BaseIntegrity: "unapproved"}, cfg))
	assert.Equal(t, "approved", formalEffectiveIntegrity(formalIntegrityItem{AuthorLogin: "trusted-user", BaseIntegrity: "approved"}, cfg))
	assert.Equal(t, "merged", formalEffectiveIntegrity(formalIntegrityItem{AuthorLogin: "trusted-user", BaseIntegrity: "merged"}, cfg))
}

func TestFormal_ApprovalLabelElevatesToApproved(t *testing.T) {
	cfg := formalIntegrityGuardConfig{ApprovalLabels: []string{"human-reviewed"}, MinIntegrity: "approved"}
	item := formalIntegrityItem{AuthorLogin: "external", BaseIntegrity: "unapproved", Labels: []string{"human-reviewed"}}

	assert.Equal(t, "approved", formalEffectiveIntegrity(item, cfg))
	assert.True(t, formalIntegrityAccessDecision(item, cfg).allow)
}

func TestFormal_DefaultIsBaseIntegrity(t *testing.T) {
	cfg := formalIntegrityGuardConfig{}
	levels := []string{"none", "unapproved", "approved", "merged"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			item := formalIntegrityItem{AuthorLogin: "external", BaseIntegrity: level, Labels: []string{"no-match"}}
			assert.Equal(t, level, formalEffectiveIntegrity(item, cfg))
		})
	}
}

func TestFormal_ElevationNeverLowersIntegrity(t *testing.T) {
	levels := []string{"none", "unapproved", "approved", "merged"}
	configs := []formalIntegrityGuardConfig{
		{TrustedUsers: []string{"trusted"}},
		{ApprovalLabels: []string{"human-reviewed"}},
		{TrustedUsers: []string{"trusted"}, ApprovalLabels: []string{"human-reviewed"}},
	}

	for _, cfg := range configs {
		for _, level := range levels {
			item := formalIntegrityItem{AuthorLogin: "trusted", BaseIntegrity: level, Labels: []string{"human-reviewed"}}
			effective := formalEffectiveIntegrity(item, cfg)
			assert.GreaterOrEqual(t, formalEffectiveIntegrityRank(effective), formalEffectiveIntegrityRank(level))
		}
	}
}

func TestFormal_IntegrityAccessDecisionTable(t *testing.T) {
	cases := []struct {
		name          string
		cfg           formalIntegrityGuardConfig
		item          formalIntegrityItem
		wantIntegrity string
		wantAllow     bool
		wantCode      int
	}{
		{
			name:          "blocked user denied regardless of label",
			cfg:           formalIntegrityGuardConfig{BlockedUsers: []string{"spam-bot"}, ApprovalLabels: []string{"human-reviewed"}, MinIntegrity: "none"},
			item:          formalIntegrityItem{AuthorLogin: "spam-bot", BaseIntegrity: "merged", Labels: []string{"human-reviewed"}},
			wantIntegrity: "blocked",
			wantAllow:     false,
			wantCode:      formalErrorBlockedUser,
		},
		{
			name:          "trusted user elevates and passes min approved",
			cfg:           formalIntegrityGuardConfig{TrustedUsers: []string{"partner"}, MinIntegrity: "approved"},
			item:          formalIntegrityItem{AuthorLogin: "partner", BaseIntegrity: "none"},
			wantIntegrity: "approved",
			wantAllow:     true,
		},
		{
			name:          "approval label elevates and passes min approved",
			cfg:           formalIntegrityGuardConfig{ApprovalLabels: []string{"human-reviewed"}, MinIntegrity: "approved"},
			item:          formalIntegrityItem{AuthorLogin: "external", BaseIntegrity: "unapproved", Labels: []string{"human-reviewed"}},
			wantIntegrity: "approved",
			wantAllow:     true,
		},
		{
			name:          "no elevation fails min approved",
			cfg:           formalIntegrityGuardConfig{MinIntegrity: "approved"},
			item:          formalIntegrityItem{AuthorLogin: "external", BaseIntegrity: "unapproved"},
			wantIntegrity: "unapproved",
			wantAllow:     false,
			wantCode:      formalErrorIntegrityTooLow,
		},
		{
			name:          "merged remains merged and passes min merged",
			cfg:           formalIntegrityGuardConfig{TrustedUsers: []string{"maintainer"}, MinIntegrity: "merged"},
			item:          formalIntegrityItem{AuthorLogin: "maintainer", BaseIntegrity: "merged"},
			wantIntegrity: "merged",
			wantAllow:     true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			effective := formalEffectiveIntegrity(tc.item, tc.cfg)
			decision := formalIntegrityAccessDecision(tc.item, tc.cfg)

			assert.Equal(t, tc.wantIntegrity, effective)
			assert.Equal(t, tc.wantAllow, decision.allow)
			if !tc.wantAllow {
				assert.Equal(t, tc.wantCode, decision.errorCode)
			}
		})
	}
}

func TestFormal_EmptyTrustedUsersAndLabelsTreatedAsOmitted(t *testing.T) {
	cfg := formalIntegrityGuardConfig{TrustedUsers: []string{}, ApprovalLabels: []string{}, MinIntegrity: "approved"}
	item := formalIntegrityItem{AuthorLogin: "external", BaseIntegrity: "unapproved", Labels: []string{"human-reviewed"}}

	assert.Equal(t, "unapproved", formalEffectiveIntegrity(item, cfg))
	decision := formalIntegrityAccessDecision(item, cfg)
	assert.False(t, decision.allow)
	assert.Equal(t, formalErrorIntegrityTooLow, decision.errorCode)
}

func TestFormal_CaseInsensitiveUserMatching(t *testing.T) {
	blockedCfg := formalIntegrityGuardConfig{BlockedUsers: []string{"Bad-Actor"}, MinIntegrity: "none"}
	blockedItem := formalIntegrityItem{AuthorLogin: "bad-actor", BaseIntegrity: "merged"}

	assert.Equal(t, "blocked", formalEffectiveIntegrity(blockedItem, blockedCfg))
	assert.False(t, formalIntegrityAccessDecision(blockedItem, blockedCfg).allow)

	trustedCfg := formalIntegrityGuardConfig{TrustedUsers: []string{"Trusted-Partner"}, MinIntegrity: "approved"}
	trustedItem := formalIntegrityItem{AuthorLogin: "trusted-partner", BaseIntegrity: "none"}

	assert.Equal(t, "approved", formalEffectiveIntegrity(trustedItem, trustedCfg))
	assert.True(t, formalIntegrityAccessDecision(trustedItem, trustedCfg).allow)
}

func TestFormal_UnsetMinIntegrityAlwaysAllowsNonBlocked(t *testing.T) {
	cfg := formalIntegrityGuardConfig{BlockedUsers: []string{"blocked"}}

	assert.True(t, formalIntegrityAccessDecision(formalIntegrityItem{AuthorLogin: "external", BaseIntegrity: "none"}, cfg).allow)
	assert.True(t, formalIntegrityAccessDecision(formalIntegrityItem{AuthorLogin: "external", BaseIntegrity: "unapproved"}, cfg).allow)
	assert.True(t, formalIntegrityAccessDecision(formalIntegrityItem{AuthorLogin: "external", BaseIntegrity: "approved"}, cfg).allow)
	assert.True(t, formalIntegrityAccessDecision(formalIntegrityItem{AuthorLogin: "external", BaseIntegrity: "merged"}, cfg).allow)

	blocked := formalIntegrityAccessDecision(formalIntegrityItem{AuthorLogin: "blocked", BaseIntegrity: "merged"}, cfg)
	assert.False(t, blocked.allow)
	assert.Equal(t, formalErrorBlockedUser, blocked.errorCode)
}

func formalEffectiveIntegrity(item formalIntegrityItem, cfg formalIntegrityGuardConfig) string {
	if containsFold(cfg.BlockedUsers, item.AuthorLogin) {
		return "blocked"
	}

	effectiveRank := formalEffectiveIntegrityRank(item.BaseIntegrity)
	if effectiveRank < 0 {
		effectiveRank = formalEffectiveIntegrityRank("none")
	}

	if containsFold(cfg.TrustedUsers, item.AuthorLogin) {
		effectiveRank = max(effectiveRank, formalEffectiveIntegrityRank("approved"))
	}
	if intersectsFold(cfg.ApprovalLabels, item.Labels) {
		effectiveRank = max(effectiveRank, formalEffectiveIntegrityRank("approved"))
	}

	return formalIntegrityLevelFromRank(effectiveRank)
}

func formalIntegrityAccessDecision(item formalIntegrityItem, cfg formalIntegrityGuardConfig) formalIntegrityDecision {
	effective := formalEffectiveIntegrity(item, cfg)
	if effective == "blocked" {
		return formalIntegrityDecision{allow: false, errorCode: formalErrorBlockedUser}
	}

	if strings.TrimSpace(cfg.MinIntegrity) == "" {
		return formalIntegrityDecision{allow: true}
	}

	if formalEffectiveIntegrityRank(effective) < formalEffectiveIntegrityRank(cfg.MinIntegrity) {
		return formalIntegrityDecision{allow: false, errorCode: formalErrorIntegrityTooLow}
	}

	return formalIntegrityDecision{allow: true}
}

func formalEffectiveIntegrityRank(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "blocked":
		return -1
	case "none":
		return 0
	case "unapproved":
		return 1
	case "approved":
		return 2
	case "merged":
		return 3
	default:
		return -2
	}
}

func formalIntegrityLevelFromRank(rank int) string {
	switch rank {
	case 3:
		return "merged"
	case 2:
		return "approved"
	case 1:
		return "unapproved"
	case 0:
		return "none"
	default:
		return "none"
	}
}

func containsFold(values []string, needle string) bool {
	return slices.ContainsFunc(values, func(value string) bool {
		return strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(needle))
	})
}

func intersectsFold(a, b []string) bool {
	for _, left := range a {
		if containsFold(b, left) {
			return true
		}
	}
	return false
}
