//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestNewPermissions(t *testing.T) {
	p := NewPermissions()
	if p == nil {
		t.Fatal("NewPermissions() returned nil")
	}
	if p.shorthand != "" {
		t.Errorf("expected empty shorthand, got %q", p.shorthand)
	}
	if p.permissions == nil {
		t.Error("expected permissions map to be initialized")
	}
	if len(p.permissions) != 0 {
		t.Errorf("expected empty permissions map, got %d entries", len(p.permissions))
	}
}

func TestNewPermissionsShorthand(t *testing.T) {
	tests := []struct {
		name      string
		fn        func() *Permissions
		shorthand string
	}{
		{"read-all", NewPermissionsReadAll, "read-all"},
		{"write-all", NewPermissionsWriteAll, "write-all"},
		{"none", NewPermissionsNone, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.fn()
			if p.shorthand != tt.shorthand {
				t.Errorf("expected shorthand %q, got %q", tt.shorthand, p.shorthand)
			}
		})
	}
}

func TestNewPermissionsFromMap(t *testing.T) {
	perms := map[PermissionScope]PermissionLevel{
		PermissionContents: PermissionRead,
		PermissionIssues:   PermissionWrite,
	}

	p := NewPermissionsFromMap(perms)
	if p.shorthand != "" {
		t.Errorf("expected empty shorthand, got %q", p.shorthand)
	}
	if len(p.permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(p.permissions))
	}

	level, exists := p.Get(PermissionContents)
	if !exists || level != PermissionRead {
		t.Errorf("expected contents: read, got %v (exists: %v)", level, exists)
	}

	level, exists = p.Get(PermissionIssues)
	if !exists || level != PermissionWrite {
		t.Errorf("expected issues: write, got %v (exists: %v)", level, exists)
	}
}

func TestPermissionsSet(t *testing.T) {
	p := NewPermissions()
	p.Set(PermissionContents, PermissionRead)

	level, exists := p.Get(PermissionContents)
	if !exists || level != PermissionRead {
		t.Errorf("expected contents: read, got %v (exists: %v)", level, exists)
	}

	// Test setting on shorthand converts to map
	p2 := NewPermissionsReadAll()
	p2.Set(PermissionIssues, PermissionWrite)
	if p2.shorthand != "" {
		t.Error("expected shorthand to be cleared after Set")
	}
	level, exists = p2.Get(PermissionIssues)
	if !exists || level != PermissionWrite {
		t.Errorf("expected issues: write, got %v (exists: %v)", level, exists)
	}
}

// TestPermissionsSetPreservesShorthandPermissions verifies that calling Set() on a Permissions
// with a shorthand value (read-all, write-all, none) preserves the shorthand-implied permissions
// instead of discarding them. This is the regression test for the bug where adding
// copilot-requests: write to a read-all workflow dropped all other read permissions.
func TestPermissionsSetPreservesShorthandPermissions(t *testing.T) {
	tests := []struct {
		name            string
		base            *Permissions
		setScope        PermissionScope
		setLevel        PermissionLevel
		checkScope      PermissionScope
		wantPreserved   PermissionLevel
		wantPreservedOK bool
	}{
		{
			name:            "read-all: adding write scope preserves contents: read",
			base:            NewPermissionsReadAll(),
			setScope:        PermissionCopilotRequests,
			setLevel:        PermissionWrite,
			checkScope:      PermissionContents,
			wantPreserved:   PermissionRead,
			wantPreservedOK: true,
		},
		{
			name:            "read-all: adding write scope preserves issues: read",
			base:            NewPermissionsReadAll(),
			setScope:        PermissionCopilotRequests,
			setLevel:        PermissionWrite,
			checkScope:      PermissionIssues,
			wantPreserved:   PermissionRead,
			wantPreservedOK: true,
		},
		{
			name:            "read-all: adding write scope preserves pull-requests: read",
			base:            NewPermissionsReadAll(),
			setScope:        PermissionCopilotRequests,
			setLevel:        PermissionWrite,
			checkScope:      PermissionPullRequests,
			wantPreserved:   PermissionRead,
			wantPreservedOK: true,
		},
		{
			name:            "write-all: adding extra write scope preserves contents: write",
			base:            NewPermissionsWriteAll(),
			setScope:        PermissionCopilotRequests,
			setLevel:        PermissionWrite,
			checkScope:      PermissionContents,
			wantPreserved:   PermissionWrite,
			wantPreservedOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Set(tt.setScope, tt.setLevel)

			// Verify the newly set scope
			gotNew, existsNew := tt.base.Get(tt.setScope)
			if !existsNew || gotNew != tt.setLevel {
				t.Errorf("Set scope %s: got %v (exists=%v), want %v", tt.setScope, gotNew, existsNew, tt.setLevel)
			}

			// Verify the preserved scope
			gotPrev, existsPrev := tt.base.Get(tt.checkScope)
			if existsPrev != tt.wantPreservedOK || gotPrev != tt.wantPreserved {
				t.Errorf("Preserved scope %s: got %v (exists=%v), want %v (exists=%v)",
					tt.checkScope, gotPrev, existsPrev, tt.wantPreserved, tt.wantPreservedOK)
			}
		})
	}
}

func TestPermissionsGet(t *testing.T) {
	tests := []struct {
		name        string
		permissions *Permissions
		scope       PermissionScope
		wantLevel   PermissionLevel
		wantExists  bool
	}{
		{
			name:        "read-all shorthand",
			permissions: NewPermissionsReadAll(),
			scope:       PermissionContents,
			wantLevel:   PermissionRead,
			wantExists:  true,
		},
		{
			name:        "write-all shorthand",
			permissions: NewPermissionsWriteAll(),
			scope:       PermissionIssues,
			wantLevel:   PermissionWrite,
			wantExists:  true,
		},
		{
			name:        "none shorthand",
			permissions: NewPermissionsNone(),
			scope:       PermissionContents,
			wantLevel:   PermissionNone,
			wantExists:  true,
		},
		{
			name: "specific permission",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionRead,
			}),
			scope:      PermissionContents,
			wantLevel:  PermissionRead,
			wantExists: true,
		},
		{
			name:        "non-existent permission",
			permissions: NewPermissions(),
			scope:       PermissionContents,
			wantLevel:   "",
			wantExists:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, exists := tt.permissions.Get(tt.scope)
			if exists != tt.wantExists {
				t.Errorf("Get() exists = %v, want %v", exists, tt.wantExists)
			}
			if level != tt.wantLevel {
				t.Errorf("Get() level = %v, want %v", level, tt.wantLevel)
			}
		})
	}
}

type permissionsFactory func() *Permissions

type permissionsMergeTestCase struct {
	name   string
	base   permissionsFactory
	merge  permissionsFactory
	want   map[PermissionScope]PermissionLevel
	wantSH string
}

type permissionsMergeGroup struct {
	name  string
	tests []permissionsMergeTestCase
}

var permissionsMergeTests = []permissionsMergeTestCase{
	{
		name:  "merge two maps - write overrides read",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite}),
		want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite},
	},
	{
		name:  "merge two maps - read doesn't override write",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite}),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
		want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite},
	},
	{
		name:  "merge two maps - different scopes",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite}),
		want: map[PermissionScope]PermissionLevel{
			PermissionContents: PermissionRead,
			PermissionIssues:   PermissionWrite,
		},
	},
	{
		name: "merge two maps - multiple scopes with conflicts",
		base: testPermissionsMap(map[PermissionScope]PermissionLevel{
			PermissionContents:     PermissionRead,
			PermissionIssues:       PermissionWrite,
			PermissionPullRequests: PermissionRead,
		}),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{
			PermissionContents:    PermissionWrite,
			PermissionIssues:      PermissionRead,
			PermissionDiscussions: PermissionWrite,
		}),
		want: map[PermissionScope]PermissionLevel{
			PermissionContents:     PermissionWrite, // write wins
			PermissionIssues:       PermissionWrite, // write preserved
			PermissionPullRequests: PermissionRead,  // kept from base
			PermissionDiscussions:  PermissionWrite, // added from merge
		},
	},
	{
		name:  "merge two maps - none overrides read",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionNone}),
		want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead},
	},
	{
		name:  "merge two maps - none overrides none",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionNone}),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionNone}),
		want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionNone},
	},
	{
		name:  "merge two maps - write overrides none",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionNone}),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite}),
		want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite},
	},
	{
		name: "merge two maps - all permission scopes",
		base: testPermissionsMap(map[PermissionScope]PermissionLevel{
			PermissionActions:     PermissionRead,
			PermissionChecks:      PermissionRead,
			PermissionContents:    PermissionRead,
			PermissionDeployments: PermissionRead,
			PermissionDiscussions: PermissionRead,
			PermissionIssues:      PermissionRead,
			PermissionPackages:    PermissionRead,
		}),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{
			PermissionPages:          PermissionWrite,
			PermissionPullRequests:   PermissionWrite,
			PermissionRepositoryProj: PermissionWrite,
			PermissionSecurityEvents: PermissionWrite,
			PermissionStatuses:       PermissionWrite,
			PermissionModels:         PermissionWrite,
		}),
		want: map[PermissionScope]PermissionLevel{
			PermissionActions:        PermissionRead,
			PermissionChecks:         PermissionRead,
			PermissionContents:       PermissionRead,
			PermissionDeployments:    PermissionRead,
			PermissionDiscussions:    PermissionRead,
			PermissionIssues:         PermissionRead,
			PermissionPackages:       PermissionRead,
			PermissionPages:          PermissionWrite,
			PermissionPullRequests:   PermissionWrite,
			PermissionRepositoryProj: PermissionWrite,
			PermissionSecurityEvents: PermissionWrite,
			PermissionStatuses:       PermissionWrite,
			PermissionModels:         PermissionWrite,
		},
	},

	// Shorthand-to-Shorthand merges
	{
		name:   "merge shorthand - write-all wins over read-all",
		base:   testPermissionsReadAll(),
		merge:  testPermissionsWriteAll(),
		wantSH: "write-all",
	},
	{
		name:   "merge shorthand - write-all wins over read",
		base:   testPermissionsReadAll(),
		merge:  testPermissionsWriteAll(),
		wantSH: "write-all",
	},
	{
		name:   "merge shorthand - write-all wins over write",
		base:   testPermissionsWriteAll(),
		merge:  testPermissionsWriteAll(),
		wantSH: "write-all",
	},
	{
		name:   "merge shorthand - write-all wins over none",
		base:   testPermissionsNone(),
		merge:  testPermissionsWriteAll(),
		wantSH: "write-all",
	},
	{
		name:   "merge shorthand - write-all wins over read-all",
		base:   testPermissionsReadAll(),
		merge:  testPermissionsWriteAll(),
		wantSH: "write-all",
	},
	{
		name:   "merge shorthand - write-all wins over read-all (duplicate for coverage)",
		base:   testPermissionsReadAll(),
		merge:  testPermissionsWriteAll(),
		wantSH: "write-all",
	},
	{
		name:   "merge shorthand - write-all wins over none",
		base:   testPermissionsNone(),
		merge:  testPermissionsWriteAll(),
		wantSH: "write-all",
	},
	{
		name:   "merge shorthand - read-all wins over read-all",
		base:   testPermissionsReadAll(),
		merge:  testPermissionsReadAll(),
		wantSH: "read-all",
	},
	{
		name:   "merge shorthand - read-all wins over none",
		base:   testPermissionsNone(),
		merge:  testPermissionsReadAll(),
		wantSH: "read-all",
	},
	{
		name:   "merge shorthand - read-all wins over none (duplicate for coverage)",
		base:   testPermissionsNone(),
		merge:  testPermissionsReadAll(),
		wantSH: "read-all",
	},
	{
		name:   "merge shorthand - read-all preserved when merging read",
		base:   testPermissionsReadAll(),
		merge:  testPermissionsReadAll(),
		wantSH: "read-all",
	},
	{
		name:   "merge shorthand - write-all preserved when merging write",
		base:   testPermissionsWriteAll(),
		merge:  testPermissionsWriteAll(),
		wantSH: "write-all",
	},
	{
		name:   "merge shorthand - same shorthand preserved (read-all)",
		base:   testPermissionsReadAll(),
		merge:  testPermissionsReadAll(),
		wantSH: "read-all",
	},
	{
		name:   "merge shorthand - same shorthand preserved (write-all)",
		base:   testPermissionsWriteAll(),
		merge:  testPermissionsWriteAll(),
		wantSH: "write-all",
	},
	{
		name:   "merge shorthand - same shorthand preserved (none)",
		base:   testPermissionsNone(),
		merge:  testPermissionsNone(),
		wantSH: "none",
	},

	// Shorthand-to-Map merges
	{
		name:  "merge read-all shorthand into map - adds all missing scopes as read",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite}),
		merge: testPermissionsReadAll(),
		want: map[PermissionScope]PermissionLevel{
			PermissionContents:            PermissionWrite, // preserved
			PermissionActions:             PermissionRead,  // added
			PermissionAttestations:        PermissionRead,
			PermissionChecks:              PermissionRead,
			PermissionDeployments:         PermissionRead,
			PermissionDiscussions:         PermissionRead,
			PermissionIssues:              PermissionRead,
			PermissionMetadata:            PermissionRead,
			PermissionPackages:            PermissionRead,
			PermissionPages:               PermissionRead,
			PermissionPullRequests:        PermissionRead,
			PermissionRepositoryProj:      PermissionRead,
			PermissionSecurityEvents:      PermissionRead,
			PermissionStatuses:            PermissionRead,
			PermissionModels:              PermissionRead,
			PermissionVulnerabilityAlerts: PermissionRead,
			// Note: id-token is NOT included because it doesn't support read level
			// Note: organization-projects is NOT included because it's a GitHub App-only scope
		},
	},
	{
		name:  "merge write-all shorthand into map - adds all missing scopes as write",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
		merge: testPermissionsWriteAll(),
		want: map[PermissionScope]PermissionLevel{
			PermissionContents:            PermissionRead, // preserved (not overwritten)
			PermissionActions:             PermissionWrite,
			PermissionAttestations:        PermissionWrite,
			PermissionChecks:              PermissionWrite,
			PermissionDeployments:         PermissionWrite,
			PermissionDiscussions:         PermissionWrite,
			PermissionIdToken:             PermissionWrite, // id-token supports write
			PermissionIssues:              PermissionWrite,
			PermissionMetadata:            PermissionWrite,
			PermissionPackages:            PermissionWrite,
			PermissionPages:               PermissionWrite,
			PermissionPullRequests:        PermissionWrite,
			PermissionRepositoryProj:      PermissionWrite,
			PermissionSecurityEvents:      PermissionWrite,
			PermissionStatuses:            PermissionWrite,
			PermissionModels:              PermissionWrite,
			PermissionVulnerabilityAlerts: PermissionWrite,
			// Note: organization-projects is NOT included because it's a GitHub App-only scope
		},
	},
	{
		name:  "merge read shorthand into map - adds all missing scopes as read",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite}),
		merge: testPermissionsReadAll(),
		want: map[PermissionScope]PermissionLevel{
			PermissionContents:            PermissionWrite,
			PermissionActions:             PermissionRead,
			PermissionAttestations:        PermissionRead,
			PermissionChecks:              PermissionRead,
			PermissionDeployments:         PermissionRead,
			PermissionDiscussions:         PermissionRead,
			PermissionIssues:              PermissionRead,
			PermissionMetadata:            PermissionRead,
			PermissionPackages:            PermissionRead,
			PermissionPages:               PermissionRead,
			PermissionPullRequests:        PermissionRead,
			PermissionRepositoryProj:      PermissionRead,
			PermissionSecurityEvents:      PermissionRead,
			PermissionStatuses:            PermissionRead,
			PermissionModels:              PermissionRead,
			PermissionVulnerabilityAlerts: PermissionRead,
			// Note: id-token is NOT included because it doesn't support read level
			// Note: organization-projects is NOT included because it's a GitHub App-only scope
		},
	},
	{
		name:  "merge write shorthand into map - adds all missing scopes as write",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionIssues: PermissionRead}),
		merge: testPermissionsWriteAll(),
		want: map[PermissionScope]PermissionLevel{
			PermissionIssues:              PermissionRead,
			PermissionActions:             PermissionWrite,
			PermissionAttestations:        PermissionWrite,
			PermissionChecks:              PermissionWrite,
			PermissionContents:            PermissionWrite,
			PermissionDeployments:         PermissionWrite,
			PermissionDiscussions:         PermissionWrite,
			PermissionIdToken:             PermissionWrite, // id-token supports write
			PermissionMetadata:            PermissionWrite,
			PermissionPackages:            PermissionWrite,
			PermissionPages:               PermissionWrite,
			PermissionPullRequests:        PermissionWrite,
			PermissionRepositoryProj:      PermissionWrite,
			PermissionSecurityEvents:      PermissionWrite,
			PermissionStatuses:            PermissionWrite,
			PermissionModels:              PermissionWrite,
			PermissionVulnerabilityAlerts: PermissionWrite,
		},
	},
	{
		name:  "merge none shorthand into map - no change",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
		merge: testPermissionsNone(),
		want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead},
	},

	// Map-to-Shorthand merges (shorthand converts to map)
	{
		name:  "merge map into read-all shorthand - shorthand cleared, map created",
		base:  testPermissionsReadAll(),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite}),
		want:  map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite},
	},
	{
		name:  "merge map into write-all shorthand - shorthand cleared, map created",
		base:  testPermissionsWriteAll(),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
		want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead},
	},
	{
		name:  "merge map into none shorthand - shorthand cleared, map created",
		base:  testPermissionsNone(),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite}),
		want:  map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite},
	},
	{
		name: "merge complex map into read shorthand",
		base: testPermissionsReadAll(),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{
			PermissionContents:     PermissionWrite,
			PermissionIssues:       PermissionRead,
			PermissionPullRequests: PermissionWrite,
		}),
		want: map[PermissionScope]PermissionLevel{
			PermissionContents:     PermissionWrite,
			PermissionIssues:       PermissionRead,
			PermissionPullRequests: PermissionWrite,
		},
	},

	// Nil and edge cases
	{
		name:  "merge nil into map - no change",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
		merge: nil,
		want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead},
	},
	{
		name:   "merge nil into shorthand - no change",
		base:   testPermissionsReadAll(),
		merge:  nil,
		wantSH: "read-all",
	},
	{
		name:  "merge empty map into map - no change",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{}),
		want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead},
	},
	{
		name:  "merge map into empty map - scopes added",
		base:  testPermissionsMap(map[PermissionScope]PermissionLevel{}),
		merge: testPermissionsMap(map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite}),
		want:  map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite},
	},
}

var permissionsMergeGroups = []permissionsMergeGroup{
	{name: "map to map", tests: permissionsMergeTests[0:8]},
	{name: "shorthand to shorthand", tests: permissionsMergeTests[8:23]},
	{name: "shorthand to map", tests: permissionsMergeTests[23:28]},
	{name: "map to shorthand", tests: permissionsMergeTests[28:32]},
	{name: "nil and edge cases", tests: permissionsMergeTests[32:36]},
}

func TestPermissionsMerge(t *testing.T) {
	for _, group := range permissionsMergeGroups {
		group := group
		t.Run(group.name, func(t *testing.T) {
			for _, tt := range group.tests {
				tt := tt
				t.Run(tt.name, func(t *testing.T) {
					testPermissionsMergeCase(t, tt)
				})
			}
		})
	}
}

func testPermissionsMergeCase(t *testing.T, tt permissionsMergeTestCase) {
	t.Helper()

	base := tt.base()
	var merge *Permissions
	if tt.merge != nil {
		merge = tt.merge()
	}
	base.Merge(merge)
	if tt.wantSH != "" {
		if base.shorthand != tt.wantSH {
			t.Errorf("after merge, shorthand = %q, want %q", base.shorthand, tt.wantSH)
		}
		return
	}
	if len(tt.want) != len(base.permissions) {
		t.Errorf("after merge, got %d permissions, want %d", len(base.permissions), len(tt.want))
	}
	for scope, wantLevel := range tt.want {
		gotLevel, exists := base.Get(scope)
		if !exists {
			t.Errorf("after merge, scope %s not found", scope)
			continue
		}
		if gotLevel != wantLevel {
			t.Errorf("after merge, scope %s = %v, want %v", scope, gotLevel, wantLevel)
		}
	}
}

func testPermissionsMap(perms map[PermissionScope]PermissionLevel) permissionsFactory {
	return func() *Permissions {
		return NewPermissionsFromMap(perms)
	}
}

func testPermissionsReadAll() permissionsFactory {
	return NewPermissionsReadAll
}

func testPermissionsWriteAll() permissionsFactory {
	return NewPermissionsWriteAll
}

func testPermissionsNone() permissionsFactory {
	return NewPermissionsNone
}

func TestPermissions_AllRead(t *testing.T) {
	tests := []struct {
		name     string
		perms    *Permissions
		scope    PermissionScope
		expected PermissionLevel
		exists   bool
	}{
		{
			name:     "all: read returns read for contents",
			perms:    NewPermissionsAllRead(),
			scope:    PermissionContents,
			expected: PermissionRead,
			exists:   true,
		},
		{
			name:     "all: read returns read for issues",
			perms:    NewPermissionsAllRead(),
			scope:    PermissionIssues,
			expected: PermissionRead,
			exists:   true,
		},
		{
			name: "all: read with explicit override",
			perms: func() *Permissions {
				p := NewPermissionsAllRead()
				p.Set(PermissionContents, PermissionWrite)
				return p
			}(),
			scope:    PermissionContents,
			expected: PermissionWrite,
			exists:   true,
		},
		{
			name:     "all: read does not include id-token (not supported at read level)",
			perms:    NewPermissionsAllRead(),
			scope:    PermissionIdToken,
			expected: "",    // Should be empty since the permission doesn't exist
			exists:   false, // Should not exist because id-token doesn't support read
		},
		{
			name: "all: read with explicit id-token: write override",
			perms: func() *Permissions {
				p := NewPermissionsAllRead()
				p.Set(PermissionIdToken, PermissionWrite)
				return p
			}(),
			scope:    PermissionIdToken,
			expected: PermissionWrite,
			exists:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, exists := tt.perms.Get(tt.scope)
			if exists != tt.exists {
				t.Errorf("Get(%s) exists = %v, want %v", tt.scope, exists, tt.exists)
			}
			if level != tt.expected {
				t.Errorf("Get(%s) = %v, want %v", tt.scope, level, tt.expected)
			}
		})
	}
}

func TestFilterJobLevelPermissions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectEmpty bool
		contains    []string
		excludes    []string
	}{
		{
			name:        "empty input returns empty",
			input:       "",
			expectEmpty: true,
			contains:    []string{},
			excludes:    []string{},
		},
		{
			name:  "standard permissions are preserved",
			input: "permissions:\n  contents: read\n  issues: write",
			contains: []string{
				"permissions:",
				"  contents: read",
				"  issues: write",
			},
			excludes: []string{},
		},
		{
			name:  "vulnerability-alerts is preserved (GITHUB_TOKEN scope)",
			input: "permissions:\n  contents: read\n  pull-requests: read\n  security-events: read\n  vulnerability-alerts: read",
			contains: []string{
				"permissions:",
				"  contents: read",
				"  pull-requests: read",
				"  security-events: read",
				"  vulnerability-alerts: read",
			},
			excludes: []string{},
		},
		{
			name:  "multiple GitHub App-only scopes are filtered out but vulnerability-alerts is preserved",
			input: "permissions:\n  contents: read\n  issues: write\n  administration: read\n  members: read\n  vulnerability-alerts: read",
			contains: []string{
				"permissions:",
				"  contents: read",
				"  issues: write",
				"  vulnerability-alerts: read",
			},
			excludes: []string{"administration", "members"},
		},
		{
			name:        "only GitHub App-only scopes returns empty string",
			input:       "permissions:\n  members: read",
			expectEmpty: true,
			contains:    []string{},
			excludes:    []string{"members"},
		},
		{
			name:     "shorthand read-all is preserved unchanged",
			input:    "permissions: read-all",
			contains: []string{"permissions: read-all"},
			excludes: []string{},
		},
		{
			name:     "shorthand write-all is preserved unchanged",
			input:    "permissions: write-all",
			contains: []string{"permissions: write-all"},
			excludes: []string{},
		},
		{
			name:     "shorthand none is preserved unchanged",
			input:    "permissions: none",
			contains: []string{"permissions: none"},
			excludes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterJobLevelPermissions(tt.input)
			if tt.expectEmpty && result != "" {
				t.Errorf("filterJobLevelPermissions() should return empty string, but got:\n%q", result)
			}
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("filterJobLevelPermissions() should contain %q, but got:\n%q", expected, result)
				}
			}
			for _, excluded := range tt.excludes {
				if strings.Contains(result, excluded) {
					t.Errorf("filterJobLevelPermissions() should NOT contain %q, but got:\n%q", excluded, result)
				}
			}
		})
	}
}

func TestPermissions_HasContentsReadAccess(t *testing.T) {
	tests := []struct {
		name     string
		perms    *Permissions
		expected bool
	}{
		{
			name:     "nil permissions returns false",
			perms:    nil,
			expected: false,
		},
		{
			name:     "read-all shorthand grants contents read",
			perms:    NewPermissionsReadAll(),
			expected: true,
		},
		{
			name:     "write-all shorthand grants contents read",
			perms:    NewPermissionsWriteAll(),
			expected: true,
		},
		{
			name:     "none shorthand denies contents read",
			perms:    NewPermissionsNone(),
			expected: false,
		},
		{
			name:     "empty permissions denies contents read",
			perms:    NewPermissions(),
			expected: false,
		},
		{
			name:     "contents: read grants access",
			perms:    NewPermissionsContentsRead(),
			expected: true,
		},
		{
			name:     "contents: write grants access (write implies read)",
			perms:    NewPermissionsContentsWrite(),
			expected: true,
		},
		{
			name: "no contents permission denies access",
			perms: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionIssues: PermissionWrite,
			}),
			expected: false,
		},
		{
			name: "contents: none denies access",
			perms: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionNone,
			}),
			expected: false,
		},
		{
			name:     "all: read grants contents read",
			perms:    NewPermissionsAllRead(),
			expected: true,
		},
		{
			name: "all: write grants contents read (write implies read)",
			perms: func() *Permissions {
				p := NewPermissions()
				p.hasAll = true
				p.allLevel = PermissionWrite
				return p
			}(),
			expected: true,
		},
		{
			name: "all: read with explicit contents: none denies access",
			perms: func() *Permissions {
				p := NewPermissionsAllRead()
				p.Set(PermissionContents, PermissionNone)
				return p
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.perms.HasContentsReadAccess()
			if result != tt.expected {
				t.Errorf("HasContentsReadAccess() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFilterJobLevelPermissionsWithCache(t *testing.T) {
	// Verify that passing a pre-parsed *Permissions produces the same result
	// as parsing from the raw YAML string.
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "standard permissions with cache",
			input: "permissions:\n  contents: read\n  issues: write",
		},
		{
			name:  "shorthand read-all with cache",
			input: "permissions: read-all",
		},
		{
			name:  "shorthand none with cache",
			input: "permissions: none",
		},
		{
			name:  "App-only scopes filtered with cache",
			input: "permissions:\n  contents: read\n  members: read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compute expected result without cache
			withoutCache := filterJobLevelPermissions(tt.input)

			// Compute result using cached permissions
			cached := NewPermissionsParser(tt.input).ToPermissions()
			withCache := filterJobLevelPermissions(tt.input, cached)

			if withoutCache != withCache {
				t.Errorf("filterJobLevelPermissions() with cache produced different result:\nwithout cache: %q\nwith cache:    %q", withoutCache, withCache)
			}
		})
	}
}

func TestPermissions_Clone(t *testing.T) {
	t.Run("clone is independent - mutations do not affect original", func(t *testing.T) {
		original := NewPermissionsContentsRead()
		clone := original.Clone()

		// Mutate the clone
		clone.Set(PermissionIssues, PermissionWrite)

		// Original must be unchanged
		if _, exists := original.permissions[PermissionIssues]; exists {
			t.Error("Clone.Set() mutated the original Permissions object")
		}
	})

	t.Run("clone of shorthand is independent", func(t *testing.T) {
		original := NewPermissionsReadAll()
		clone := original.Clone()

		if clone.shorthand != original.shorthand {
			t.Errorf("Clone shorthand mismatch: got %q, want %q", clone.shorthand, original.shorthand)
		}

		// Mutating clone should not affect original
		clone.shorthand = "none"
		if original.shorthand != "read-all" {
			t.Errorf("Clone mutation affected original shorthand: %q", original.shorthand)
		}
	})

	t.Run("clone of nil returns empty permissions", func(t *testing.T) {
		var p *Permissions
		clone := p.Clone()
		if clone == nil {
			t.Fatal("Clone of nil should return non-nil empty Permissions")
		}
		if len(clone.permissions) != 0 || clone.shorthand != "" {
			t.Error("Clone of nil should return empty Permissions")
		}
	})

	t.Run("clone of all:read preserves hasAll flag", func(t *testing.T) {
		original := NewPermissionsAllRead()
		clone := original.Clone()
		if !clone.hasAll || clone.allLevel != PermissionRead {
			t.Errorf("Clone of all:read did not preserve hasAll/allLevel: hasAll=%v allLevel=%q", clone.hasAll, clone.allLevel)
		}
	})
}
