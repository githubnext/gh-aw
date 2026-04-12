//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnforceSafeUpdate(t *testing.T) {
	tests := []struct {
		name        string
		manifest    *GHAWManifest
		secretNames []string
		actionRefs  []string
		wantErr     bool
		wantErrMsgs []string
	}{
		{
			name:        "nil manifest (no lock file) blocks secrets on first compile",
			manifest:    nil,
			secretNames: []string{"MY_SECRET"},
			actionRefs:  []string{},
			wantErr:     true,
			wantErrMsgs: []string{"MY_SECRET", "safe update mode"},
		},
		{
			name:        "nil manifest (no lock file) blocks custom actions on first compile",
			manifest:    nil,
			secretNames: []string{},
			actionRefs:  []string{"my-org/my-action@abc1234 # v1"},
			wantErr:     true,
			wantErrMsgs: []string{"my-org/my-action", "safe update mode"},
		},
		{
			name:        "nil manifest (no lock file) allows GITHUB_TOKEN on first compile",
			manifest:    nil,
			secretNames: []string{"GITHUB_TOKEN"},
			actionRefs:  []string{},
			wantErr:     false,
		},
		{
			name:        "nil manifest (no lock file) with no secrets or actions passes",
			manifest:    nil,
			secretNames: []string{},
			actionRefs:  []string{},
			wantErr:     false,
		},
		{
			name:        "empty secrets and actions with existing manifest passes",
			manifest:    &GHAWManifest{Version: 1, Secrets: []string{}, Actions: []GHAWManifestAction{}},
			secretNames: []string{},
			actionRefs:  []string{},
			wantErr:     false,
		},
		{
			name:        "GITHUB_TOKEN always allowed even when not in manifest",
			manifest:    &GHAWManifest{Version: 1, Secrets: []string{}, Actions: []GHAWManifestAction{}},
			secretNames: []string{"GITHUB_TOKEN"},
			actionRefs:  []string{},
			wantErr:     false,
		},
		{
			name:        "GITHUB_TOKEN with secrets. prefix always allowed",
			manifest:    &GHAWManifest{Version: 1, Secrets: []string{}, Actions: []GHAWManifestAction{}},
			secretNames: []string{"secrets.GITHUB_TOKEN"},
			actionRefs:  []string{},
			wantErr:     false,
		},
		{
			name: "known secret passes",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{"MY_SECRET"},
				Actions: []GHAWManifestAction{},
			},
			secretNames: []string{"MY_SECRET"},
			actionRefs:  []string{},
			wantErr:     false,
		},
		{
			name: "new restricted secret causes failure",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{"EXISTING_SECRET"},
				Actions: []GHAWManifestAction{},
			},
			secretNames: []string{"EXISTING_SECRET", "NEW_SECRET"},
			actionRefs:  []string{},
			wantErr:     true,
			wantErrMsgs: []string{"NEW_SECRET", "safe update mode"},
		},
		{
			name: "multiple new secrets listed in error",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
				Actions: []GHAWManifestAction{},
			},
			secretNames: []string{"SECRET_A", "SECRET_B"},
			actionRefs:  []string{},
			wantErr:     true,
			wantErrMsgs: []string{"SECRET_A", "SECRET_B"},
		},
		{
			name: "GITHUB_TOKEN plus known secret passes",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{"MY_SECRET"},
				Actions: []GHAWManifestAction{},
			},
			secretNames: []string{"GITHUB_TOKEN", "MY_SECRET"},
			actionRefs:  []string{},
			wantErr:     false,
		},
		{
			name: "empty manifest blocks any new secret except GITHUB_TOKEN",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
				Actions: []GHAWManifestAction{},
			},
			secretNames: []string{"SOME_SECRET"},
			actionRefs:  []string{},
			wantErr:     true,
			wantErrMsgs: []string{"SOME_SECRET"},
		},
		// Action enforcement tests.
		{
			name: "known action passes",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
				Actions: []GHAWManifestAction{{Repo: "my-org/my-action", SHA: "abc1234", Version: "v1"}},
			},
			secretNames: []string{},
			actionRefs:  []string{"my-org/my-action@abc1234 # v1"},
			wantErr:     false,
		},
		{
			name: "known action with different SHA (pin update) passes",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
				Actions: []GHAWManifestAction{{Repo: "my-org/my-action", SHA: "abc1234", Version: "v1"}},
			},
			secretNames: []string{},
			actionRefs:  []string{"my-org/my-action@def5678 # v2"},
			wantErr:     false,
		},
		{
			name: "new unapproved action causes failure",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
				Actions: []GHAWManifestAction{{Repo: "actions/checkout", SHA: "abc1234", Version: "v4"}},
			},
			secretNames: []string{},
			actionRefs:  []string{"actions/checkout@abc1234 # v4", "evil-org/steal-secrets@deadbeef # v1"},
			wantErr:     true,
			wantErrMsgs: []string{"evil-org/steal-secrets", "safe update mode", "New unapproved action"},
		},
		{
			name: "removed previously-approved action causes failure",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
				Actions: []GHAWManifestAction{
					{Repo: "actions/checkout", SHA: "abc1234", Version: "v4"},
					{Repo: "my-org/setup-tool", SHA: "def5678", Version: "v3"},
				},
			},
			secretNames: []string{},
			actionRefs:  []string{"actions/checkout@abc1234 # v4"},
			wantErr:     true,
			wantErrMsgs: []string{"my-org/setup-tool", "Previously-approved action"},
		},
		{
			name: "both added and removed actions reported together",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
				Actions: []GHAWManifestAction{{Repo: "my-org/approved-action", SHA: "abc1234", Version: "v4"}},
			},
			secretNames: []string{},
			actionRefs:  []string{"evil-org/bad-action@deadbeef # v1"},
			wantErr:     true,
			wantErrMsgs: []string{"evil-org/bad-action", "my-org/approved-action"},
		},
		{
			name: "new secret and new action both reported in one error",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
				Actions: []GHAWManifestAction{},
			},
			secretNames: []string{"NEW_SECRET"},
			actionRefs:  []string{"new-org/new-action@abc # v1"},
			wantErr:     true,
			wantErrMsgs: []string{"NEW_SECRET", "new-org/new-action"},
		},
		// actions/ org exemption tests.
		{
			name:        "nil manifest: new actions/checkout is allowed on first compile",
			manifest:    nil,
			secretNames: []string{},
			actionRefs:  []string{"actions/checkout@abc1234 # v4"},
			wantErr:     false,
		},
		{
			name: "new actions/ action (not in manifest) is always allowed",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
				Actions: []GHAWManifestAction{},
			},
			secretNames: []string{},
			actionRefs:  []string{"actions/setup-node@abc1234 # v4"},
			wantErr:     false,
		},
		{
			name: "removal of actions/ action from manifest is not flagged",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
				Actions: []GHAWManifestAction{{Repo: "actions/checkout", SHA: "abc1234", Version: "v4"}},
			},
			secretNames: []string{},
			actionRefs:  []string{},
			wantErr:     false,
		},
		{
			name: "actions/ action allowed alongside non-actions/ violation",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
				Actions: []GHAWManifestAction{},
			},
			secretNames: []string{},
			actionRefs:  []string{"actions/checkout@abc1234 # v4", "evil-org/bad-action@deadbeef # v1"},
			wantErr:     true,
			wantErrMsgs: []string{"evil-org/bad-action"},
		},
		// gh-aw infrastructure action exemption tests.
		{
			name: "gh aw upgrade: gh-aw-actions/setup added after manifest had gh-aw/actions/setup",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
				Actions: []GHAWManifestAction{
					{Repo: "github/gh-aw/actions/setup", SHA: "abc1234", Version: "v0.66.1"},
				},
			},
			secretNames: []string{},
			actionRefs:  []string{"github/gh-aw-actions/setup@def5678 # v0.68.1"},
			wantErr:     false,
		},
		{
			name:        "gh-aw-actions allowed on first compile with nil manifest",
			manifest:    nil,
			secretNames: []string{},
			actionRefs:  []string{"github/gh-aw-actions/setup@abc1234 # v0.68.1"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnforceSafeUpdate(tt.manifest, tt.secretNames, tt.actionRefs)
			if tt.wantErr {
				require.Error(t, err, "expected safe update enforcement error")
				for _, msg := range tt.wantErrMsgs {
					assert.Contains(t, err.Error(), msg, "error message should contain %q", msg)
				}
			} else {
				assert.NoError(t, err, "unexpected safe update enforcement error")
			}
		})
	}
}

func TestBuildSafeUpdateError(t *testing.T) {
	t.Run("secrets only", func(t *testing.T) {
		violations := []string{"NEW_SECRET", "ANOTHER_SECRET"}
		err := buildSafeUpdateError(violations, nil, nil)
		require.Error(t, err, "should return an error")

		msg := err.Error()
		assert.Contains(t, msg, "safe update mode", "error message")
		assert.Contains(t, msg, "NEW_SECRET", "violation in message")
		assert.Contains(t, msg, "ANOTHER_SECRET", "violation in message")
		assert.Contains(t, msg, "interactive agentic flow", "remediation guidance")
	})

	t.Run("added actions only", func(t *testing.T) {
		err := buildSafeUpdateError(nil, []string{"evil-org/bad-action"}, nil)
		require.Error(t, err, "should return an error")
		msg := err.Error()
		assert.Contains(t, msg, "evil-org/bad-action", "action in message")
		assert.Contains(t, msg, "New unapproved action", "section header in message")
	})

	t.Run("removed actions only", func(t *testing.T) {
		err := buildSafeUpdateError(nil, nil, []string{"actions/setup-node"})
		require.Error(t, err, "should return an error")
		msg := err.Error()
		assert.Contains(t, msg, "actions/setup-node", "action in message")
		assert.Contains(t, msg, "Previously-approved action", "section header in message")
	})

	t.Run("mixed violations", func(t *testing.T) {
		err := buildSafeUpdateError(
			[]string{"MY_SECRET"},
			[]string{"evil-org/bad-action"},
			[]string{"actions/checkout"},
		)
		require.Error(t, err, "should return an error")
		msg := err.Error()
		assert.Contains(t, msg, "MY_SECRET", "secret in message")
		assert.Contains(t, msg, "evil-org/bad-action", "added action in message")
		assert.Contains(t, msg, "actions/checkout", "removed action in message")
	})
}

func TestCollectActionViolations(t *testing.T) {
	tests := []struct {
		name        string
		manifest    *GHAWManifest
		actionRefs  []string
		wantAdded   []string
		wantRemoved []string
	}{
		{
			name:        "no changes passes",
			manifest:    &GHAWManifest{Actions: []GHAWManifestAction{{Repo: "actions/checkout", SHA: "abc"}}},
			actionRefs:  []string{"actions/checkout@abc # v4"},
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name:        "SHA change on same repo passes",
			manifest:    &GHAWManifest{Actions: []GHAWManifestAction{{Repo: "actions/checkout", SHA: "abc"}}},
			actionRefs:  []string{"actions/checkout@def # v5"},
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name:        "new repo is an addition",
			manifest:    &GHAWManifest{Actions: []GHAWManifestAction{{Repo: "actions/checkout", SHA: "abc"}}},
			actionRefs:  []string{"actions/checkout@abc", "new-org/new-action@xyz"},
			wantAdded:   []string{"new-org/new-action"},
			wantRemoved: nil,
		},
		{
			name:        "missing repo is a removal",
			manifest:    &GHAWManifest{Actions: []GHAWManifestAction{{Repo: "my-org/custom-action", SHA: "abc"}, {Repo: "my-org/setup-tool", SHA: "def"}}},
			actionRefs:  []string{"my-org/custom-action@abc"},
			wantAdded:   nil,
			wantRemoved: []string{"my-org/setup-tool"},
		},
		{
			name:        "empty manifest with no new actions passes",
			manifest:    &GHAWManifest{Actions: []GHAWManifestAction{}},
			actionRefs:  []string{},
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name:        "new actions/ action is not an addition violation",
			manifest:    &GHAWManifest{Actions: []GHAWManifestAction{}},
			actionRefs:  []string{"actions/checkout@abc1234 # v4"},
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name:        "removal of actions/ action from manifest is not a removal violation",
			manifest:    &GHAWManifest{Actions: []GHAWManifestAction{{Repo: "actions/checkout", SHA: "abc1234", Version: "v4"}}},
			actionRefs:  []string{},
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name: "actions/ action not flagged, non-actions/ action is flagged",
			manifest: &GHAWManifest{Actions: []GHAWManifestAction{
				{Repo: "actions/checkout", SHA: "abc1234", Version: "v4"},
			}},
			actionRefs:  []string{"actions/setup-node@def5678 # v4", "evil-org/bad-action@deadbeef # v1"},
			wantAdded:   []string{"evil-org/bad-action"},
			wantRemoved: nil,
		},
		// gh-aw infrastructure action exemption tests.
		{
			name:        "new github/gh-aw-actions/ action is not an addition violation",
			manifest:    &GHAWManifest{Actions: []GHAWManifestAction{}},
			actionRefs:  []string{"github/gh-aw-actions/setup@abc1234 # v0.68.1"},
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name:        "new github/gh-aw/actions/ action is not an addition violation",
			manifest:    &GHAWManifest{Actions: []GHAWManifestAction{}},
			actionRefs:  []string{"github/gh-aw/actions/setup@abc1234 # v0.68.1"},
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name:        "removal of github/gh-aw-actions/ action from manifest is not a removal violation",
			manifest:    &GHAWManifest{Actions: []GHAWManifestAction{{Repo: "github/gh-aw-actions/setup", SHA: "abc1234", Version: "v0.66.1"}}},
			actionRefs:  []string{},
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name: "gh-aw-actions replacement of gh-aw/actions is not a violation",
			manifest: &GHAWManifest{Actions: []GHAWManifestAction{
				{Repo: "github/gh-aw/actions/setup", SHA: "abc1234", Version: "v0.66.1"},
			}},
			actionRefs:  []string{"github/gh-aw-actions/setup@def5678 # v0.68.1"},
			wantAdded:   nil,
			wantRemoved: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			added, removed := collectActionViolations(tt.manifest, tt.actionRefs)
			assert.Equal(t, tt.wantAdded, added, "added actions")
			assert.Equal(t, tt.wantRemoved, removed, "removed actions")
		})
	}
}

func TestEffectiveSafeUpdate(t *testing.T) {
	// effectiveSafeUpdate is equivalent to effectiveStrictMode:
	// it returns true when the compiler safeUpdate flag is set, OR when strict
	// mode is active (which defaults to true unless frontmatter sets strict: false).
	tests := []struct {
		name           string
		compilerFlag   bool
		rawFrontmatter map[string]any
		want           bool
	}{
		{
			name:         "compiler flag off, no frontmatter => true (strict default)",
			compilerFlag: false,
			want:         true, // strict mode defaults to true, so safe update is enabled
		},
		{
			name:         "compiler flag on => true",
			compilerFlag: true,
			want:         true,
		},
		{
			name:           "frontmatter strict: false, compiler flag off => false",
			compilerFlag:   false,
			rawFrontmatter: map[string]any{"strict": false},
			want:           false,
		},
		{
			name:           "frontmatter strict: false, compiler flag on => true",
			compilerFlag:   true,
			rawFrontmatter: map[string]any{"strict": false},
			want:           true, // CLI flag overrides frontmatter
		},
		{
			name:           "frontmatter strict: true, compiler flag off => true",
			compilerFlag:   false,
			rawFrontmatter: map[string]any{"strict": true},
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Compiler{safeUpdate: tt.compilerFlag}
			data := &WorkflowData{RawFrontmatter: tt.rawFrontmatter}
			got := c.effectiveSafeUpdate(data)
			assert.Equal(t, tt.want, got, "effectiveSafeUpdate result")
		})
	}
}
