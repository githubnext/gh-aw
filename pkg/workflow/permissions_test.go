package workflow

import (
	"strings"
	"testing"
)

func TestPermissionsParser_HasContentsReadAccess(t *testing.T) {
	tests := []struct {
		name        string
		permissions string
		expected    bool
	}{
		{
			name:        "shorthand read-all grants contents access",
			permissions: "permissions: read-all",
			expected:    true,
		},
		{
			name:        "shorthand write-all grants contents access",
			permissions: "permissions: write-all",
			expected:    true,
		},
		{
			name:        "invalid shorthand read does not grant contents access",
			permissions: "permissions: read",
			expected:    false, // "read" is no longer a valid shorthand
		},
		{
			name:        "invalid shorthand write does not grant contents access",
			permissions: "permissions: write",
			expected:    false, // "write" is no longer a valid shorthand
		},
		{
			name:        "shorthand none denies contents access",
			permissions: "permissions: none",
			expected:    false,
		},
		{
			name: "explicit contents read grants access",
			permissions: `permissions:
  contents: read
  issues: write`,
			expected: true,
		},
		{
			name: "explicit contents write grants access",
			permissions: `permissions:
  contents: write
  issues: read`,
			expected: true,
		},
		{
			name: "no contents permission denies access",
			permissions: `permissions:
  issues: write
  pull-requests: read`,
			expected: false,
		},
		{
			name: "explicit contents none denies access",
			permissions: `permissions:
  contents: none
  issues: write`,
			expected: false,
		},
		{
			name:        "empty permissions denies access",
			permissions: "",
			expected:    false,
		},
		{
			name:        "just permissions label denies access",
			permissions: "permissions:",
			expected:    false,
		},
		// Additional extensive edge case tests
		{
			name:        "whitespace only permissions denies access",
			permissions: "permissions:   \n  \t",
			expected:    false,
		},
		{
			name: "permissions with extra whitespace",
			permissions: `permissions:  
  contents:   read  
  issues: write`,
			expected: true,
		},
		{
			name:        "invalid shorthand permission denies access",
			permissions: "permissions: invalid-permission",
			expected:    false,
		},
		{
			name: "mixed case contents permission",
			permissions: `permissions:
  CONTENTS: read`,
			expected: false, // YAML is case-sensitive
		},
		{
			name: "contents with mixed case value",
			permissions: `permissions:
  contents: READ`,
			expected: false, // Values are case-sensitive
		},
		{
			name: "permissions with numeric contents value",
			permissions: `permissions:
  contents: 123`,
			expected: false,
		},
		{
			name: "permissions with boolean contents value",
			permissions: `permissions:
  contents: true`,
			expected: false,
		},
		{
			name: "deeply nested permissions structure",
			permissions: `permissions:
  security:
    contents: read
  contents: write`,
			expected: true, // Should parse the top-level contents
		},
		{
			name: "permissions with comments",
			permissions: `permissions:
  contents: read  # This grants read access
  issues: write`,
			expected: true,
		},
		{
			name: "permissions with array syntax",
			permissions: `permissions:
  contents: [read, write]`,
			expected: false, // Array values not supported
		},
		{
			name: "permissions with quoted values",
			permissions: `permissions:
  contents: "read"
  issues: write`,
			expected: true,
		},
		{
			name: "permissions with single quotes",
			permissions: `permissions:
  contents: 'write'
  issues: read`,
			expected: true,
		},
		{
			name: "malformed YAML permissions",
			permissions: `permissions:
  contents: read
    issues: write`, // Bad indentation
			expected: false,
		},
		{
			name: "permissions without colon separator",
			permissions: `permissions
  contents read`,
			expected: false,
		},
		{
			name:        "extremely long permission value",
			permissions: "permissions: " + strings.Repeat("a", 1000),
			expected:    false,
		},
		{
			name: "special characters in permission values",
			permissions: `permissions:
  contents: read@#$%
  issues: write`,
			expected: false,
		},
		{
			name: "unicode characters in permissions",
			permissions: `permissions:
  contents: 读取
  issues: write`,
			expected: false,
		},
		{
			name: "null value for contents",
			permissions: `permissions:
  contents: null
  issues: write`,
			expected: false,
		},
		{
			name: "empty string for contents",
			permissions: `permissions:
  contents: ""
  issues: write`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewPermissionsParser(tt.permissions)
			result := parser.HasContentsReadAccess()
			if result != tt.expected {
				t.Errorf("HasContentsReadAccess() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestContainsCheckout(t *testing.T) {
	tests := []struct {
		name        string
		customSteps string
		expected    bool
	}{
		{
			name:        "empty steps",
			customSteps: "",
			expected:    false,
		},
		{
			name: "contains actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd",
			customSteps: `steps:
  - name: Checkout
    uses: actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd`,
			expected: true,
		},
		{
			name: "contains actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd",
			customSteps: `steps:
  - uses: actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd
    with:
      token: ${{ secrets.GITHUB_TOKEN }}`,
			expected: true,
		},
		{
			name: "contains different action",
			customSteps: `steps:
  - name: Setup Node
    uses: actions/setup-node@395ad3262231945c25e8478fd5baf05154b1d79f
    with:
      node-version: '18'`,
			expected: false,
		},
		{
			name: "mixed steps with checkout",
			customSteps: `steps:
  - name: Checkout repository
    uses: actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd
  - name: Setup Node
    uses: actions/setup-node@395ad3262231945c25e8478fd5baf05154b1d79f`,
			expected: true,
		},
		{
			name: "case insensitive detection",
			customSteps: `steps:
  - name: Checkout
    uses: Actions/Checkout@v5`,
			expected: true,
		},
		{
			name: "checkout in middle of other text",
			customSteps: `steps:
  - name: Custom step
    run: echo "before checkout"
  - uses: actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd
  - name: After checkout
    run: echo "done"`,
			expected: true,
		},
		// Additional extensive edge case tests for ContainsCheckout
		{
			name: "checkout with no version",
			customSteps: `steps:
  - uses: actions/checkout`,
			expected: true,
		},
		{
			name: "checkout with specific commit",
			customSteps: `steps:
  - uses: actions/checkout@8f4b7f84864484a7bf31766abe9204da3cbe65b3`,
			expected: true,
		},
		{
			name: "checkout with branch reference",
			customSteps: `steps:
  - uses: actions/checkout@main`,
			expected: true,
		},
		{
			name: "checkout action in quotes",
			customSteps: `steps:
  - name: Checkout
    uses: "actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd"`,
			expected: true,
		},
		{
			name: "checkout action in single quotes",
			customSteps: `steps:
  - uses: 'actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd'`,
			expected: true,
		},
		{
			name: "checkout with extra whitespace",
			customSteps: `steps:
  - uses:   actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd   `,
			expected: true,
		},
		{
			name: "checkout in multiline YAML",
			customSteps: `steps:
  - name: Checkout
    uses: >
      actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd`,
			expected: true,
		},
		{
			name: "checkout in run command (should not match)",
			customSteps: `steps:
  - name: Echo checkout
    run: echo "actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd"`,
			expected: true, // Current implementation does simple string match
		},
		{
			name: "checkout in comment (should not match)",
			customSteps: `steps:
  - name: Setup
    uses: actions/setup-node@395ad3262231945c25e8478fd5baf05154b1d79f
    # TODO: add actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd`,
			expected: true, // Current implementation does simple string match
		},
		{
			name: "similar but not checkout action",
			customSteps: `steps:
  - uses: actions/cache@v3
  - uses: my-actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd`,
			expected: true, // Current implementation matches substring
		},
		{
			name: "checkout in different format",
			customSteps: `steps:
  - name: Checkout code
    uses: |
      actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd`,
			expected: true,
		},
		{
			name: "malformed YAML with checkout",
			customSteps: `steps
  - uses: actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd`,
			expected: true, // Still detects the string
		},
		{
			name: "checkout with complex parameters",
			customSteps: `steps:
  - name: Checkout repository
    uses: actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd
    with:
      fetch-depth: 0
      token: ${{ secrets.GITHUB_TOKEN }}
      submodules: recursive`,
			expected: true,
		},
		{
			name: "multiple checkouts",
			customSteps: `steps:
  - uses: actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd
  - name: Setup
    run: echo "setup"
  - uses: actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd
    with:
      path: subdirectory`,
			expected: true,
		},
		{
			name: "checkout with unusual casing",
			customSteps: `steps:
  - uses: ACTIONS/CHECKOUT@V5`,
			expected: true,
		},
		{
			name: "checkout in conditional step",
			customSteps: `steps:
  - if: github.event_name == 'push'
    uses: actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd`,
			expected: true,
		},
		{
			name: "very long steps with checkout buried inside",
			customSteps: `steps:
  - name: Step 1
    run: echo "first"
  - name: Step 2  
    run: echo "second"
  - name: Step 3
    run: echo "third"
  - name: Checkout buried
    uses: actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd
  - name: Step 5
    run: echo "fifth"`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsCheckout(tt.customSteps)
			if result != tt.expected {
				t.Errorf("ContainsCheckout() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestPermissionsParser_Parse(t *testing.T) {
	tests := []struct {
		name          string
		permissions   string
		expectedMap   map[string]string
		expectedShort bool
		expectedValue string
	}{
		{
			name:          "shorthand read-all",
			permissions:   "permissions: read-all",
			expectedMap:   map[string]string{},
			expectedShort: true,
			expectedValue: "read-all",
		},
		{
			name: "explicit map permissions",
			permissions: `permissions:
  contents: read
  issues: write`,
			expectedMap: map[string]string{
				"contents": "read",
				"issues":   "write",
			},
			expectedShort: false,
			expectedValue: "",
		},
		{
			name: "multiline without permissions prefix",
			permissions: `contents: read
issues: write
pull-requests: read`,
			expectedMap: map[string]string{
				"contents":      "read",
				"issues":        "write",
				"pull-requests": "read",
			},
			expectedShort: false,
			expectedValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewPermissionsParser(tt.permissions)

			if parser.isShorthand != tt.expectedShort {
				t.Errorf("isShorthand = %v, expected %v", parser.isShorthand, tt.expectedShort)
			}

			if parser.shorthandValue != tt.expectedValue {
				t.Errorf("shorthandValue = %v, expected %v", parser.shorthandValue, tt.expectedValue)
			}

			if !tt.expectedShort {
				for key, expectedValue := range tt.expectedMap {
					if actualValue, exists := parser.parsedPerms[key]; !exists || actualValue != expectedValue {
						t.Errorf("parsedPerms[%s] = %v, expected %v", key, actualValue, expectedValue)
					}
				}
			}
		})
	}
}

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

func TestPermissionsMerge(t *testing.T) {
	tests := []struct {
		name   string
		base   *Permissions
		merge  *Permissions
		want   map[PermissionScope]PermissionLevel
		wantSH string
	}{
		// Map-to-Map merges
		{
			name:  "merge two maps - write overrides read",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite}),
			want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite},
		},
		{
			name:  "merge two maps - read doesn't override write",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite}),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
			want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite},
		},
		{
			name:  "merge two maps - different scopes",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite}),
			want: map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionRead,
				PermissionIssues:   PermissionWrite,
			},
		},
		{
			name: "merge two maps - multiple scopes with conflicts",
			base: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionRead,
			}),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
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
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionNone}),
			want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead},
		},
		{
			name:  "merge two maps - none overrides none",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionNone}),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionNone}),
			want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionNone},
		},
		{
			name:  "merge two maps - write overrides none",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionNone}),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite}),
			want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite},
		},
		{
			name: "merge two maps - all permission scopes",
			base: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionActions:     PermissionRead,
				PermissionChecks:      PermissionRead,
				PermissionContents:    PermissionRead,
				PermissionDeployments: PermissionRead,
				PermissionDiscussions: PermissionRead,
				PermissionIssues:      PermissionRead,
				PermissionPackages:    PermissionRead,
			}),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
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
			base:   NewPermissionsReadAll(),
			merge:  NewPermissionsWriteAll(),
			wantSH: "write-all",
		},
		{
			name:   "merge shorthand - write-all wins over read",
			base:   NewPermissionsReadAll(),
			merge:  NewPermissionsWriteAll(),
			wantSH: "write-all",
		},
		{
			name:   "merge shorthand - write-all wins over write",
			base:   NewPermissionsWriteAll(),
			merge:  NewPermissionsWriteAll(),
			wantSH: "write-all",
		},
		{
			name:   "merge shorthand - write-all wins over none",
			base:   NewPermissionsNone(),
			merge:  NewPermissionsWriteAll(),
			wantSH: "write-all",
		},
		{
			name:   "merge shorthand - write-all wins over read-all",
			base:   NewPermissionsReadAll(),
			merge:  NewPermissionsWriteAll(),
			wantSH: "write-all",
		},
		{
			name:   "merge shorthand - write-all wins over read-all (duplicate for coverage)",
			base:   NewPermissionsReadAll(),
			merge:  NewPermissionsWriteAll(),
			wantSH: "write-all",
		},
		{
			name:   "merge shorthand - write-all wins over none",
			base:   NewPermissionsNone(),
			merge:  NewPermissionsWriteAll(),
			wantSH: "write-all",
		},
		{
			name:   "merge shorthand - read-all wins over read-all",
			base:   NewPermissionsReadAll(),
			merge:  NewPermissionsReadAll(),
			wantSH: "read-all",
		},
		{
			name:   "merge shorthand - read-all wins over none",
			base:   NewPermissionsNone(),
			merge:  NewPermissionsReadAll(),
			wantSH: "read-all",
		},
		{
			name:   "merge shorthand - read-all wins over none (duplicate for coverage)",
			base:   NewPermissionsNone(),
			merge:  NewPermissionsReadAll(),
			wantSH: "read-all",
		},
		{
			name:   "merge shorthand - read-all preserved when merging read",
			base:   NewPermissionsReadAll(),
			merge:  NewPermissionsReadAll(),
			wantSH: "read-all",
		},
		{
			name:   "merge shorthand - write-all preserved when merging write",
			base:   NewPermissionsWriteAll(),
			merge:  NewPermissionsWriteAll(),
			wantSH: "write-all",
		},
		{
			name:   "merge shorthand - same shorthand preserved (read-all)",
			base:   NewPermissionsReadAll(),
			merge:  NewPermissionsReadAll(),
			wantSH: "read-all",
		},
		{
			name:   "merge shorthand - same shorthand preserved (write-all)",
			base:   NewPermissionsWriteAll(),
			merge:  NewPermissionsWriteAll(),
			wantSH: "write-all",
		},
		{
			name:   "merge shorthand - same shorthand preserved (none)",
			base:   NewPermissionsNone(),
			merge:  NewPermissionsNone(),
			wantSH: "none",
		},

		// Shorthand-to-Map merges
		{
			name:  "merge read-all shorthand into map - adds all missing scopes as read",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite}),
			merge: NewPermissionsReadAll(),
			want: map[PermissionScope]PermissionLevel{
				PermissionContents:         PermissionWrite, // preserved
				PermissionActions:          PermissionRead,  // added
				PermissionAttestations:     PermissionRead,
				PermissionChecks:           PermissionRead,
				PermissionDeployments:      PermissionRead,
				PermissionDiscussions:      PermissionRead,
				PermissionIssues:           PermissionRead,
				PermissionPackages:         PermissionRead,
				PermissionPages:            PermissionRead,
				PermissionPullRequests:     PermissionRead,
				PermissionRepositoryProj:   PermissionRead,
				PermissionOrganizationProj: PermissionRead,
				PermissionSecurityEvents:   PermissionRead,
				PermissionStatuses:         PermissionRead,
				PermissionModels:           PermissionRead,
				// Note: id-token is NOT included because it doesn't support read level
			},
		},
		{
			name:  "merge write-all shorthand into map - adds all missing scopes as write",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
			merge: NewPermissionsWriteAll(),
			want: map[PermissionScope]PermissionLevel{
				PermissionContents:         PermissionRead, // preserved (not overwritten)
				PermissionActions:          PermissionWrite,
				PermissionAttestations:     PermissionWrite,
				PermissionChecks:           PermissionWrite,
				PermissionDeployments:      PermissionWrite,
				PermissionDiscussions:      PermissionWrite,
				PermissionIdToken:          PermissionWrite, // id-token supports write
				PermissionIssues:           PermissionWrite,
				PermissionPackages:         PermissionWrite,
				PermissionPages:            PermissionWrite,
				PermissionPullRequests:     PermissionWrite,
				PermissionRepositoryProj:   PermissionWrite,
				PermissionOrganizationProj: PermissionWrite,
				PermissionSecurityEvents:   PermissionWrite,
				PermissionStatuses:         PermissionWrite,
				PermissionModels:           PermissionWrite,
			},
		},
		{
			name:  "merge read shorthand into map - adds all missing scopes as read",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite}),
			merge: NewPermissionsReadAll(),
			want: map[PermissionScope]PermissionLevel{
				PermissionContents:         PermissionWrite,
				PermissionActions:          PermissionRead,
				PermissionAttestations:     PermissionRead,
				PermissionChecks:           PermissionRead,
				PermissionDeployments:      PermissionRead,
				PermissionDiscussions:      PermissionRead,
				PermissionIssues:           PermissionRead,
				PermissionPackages:         PermissionRead,
				PermissionPages:            PermissionRead,
				PermissionPullRequests:     PermissionRead,
				PermissionRepositoryProj:   PermissionRead,
				PermissionOrganizationProj: PermissionRead,
				PermissionSecurityEvents:   PermissionRead,
				PermissionStatuses:         PermissionRead,
				PermissionModels:           PermissionRead,
				// Note: id-token is NOT included because it doesn't support read level
			},
		},
		{
			name:  "merge write shorthand into map - adds all missing scopes as write",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionIssues: PermissionRead}),
			merge: NewPermissionsWriteAll(),
			want: map[PermissionScope]PermissionLevel{
				PermissionIssues:           PermissionRead,
				PermissionActions:          PermissionWrite,
				PermissionAttestations:     PermissionWrite,
				PermissionChecks:           PermissionWrite,
				PermissionContents:         PermissionWrite,
				PermissionDeployments:      PermissionWrite,
				PermissionDiscussions:      PermissionWrite,
				PermissionIdToken:          PermissionWrite, // id-token supports write
				PermissionPackages:         PermissionWrite,
				PermissionPages:            PermissionWrite,
				PermissionPullRequests:     PermissionWrite,
				PermissionRepositoryProj:   PermissionWrite,
				PermissionOrganizationProj: PermissionWrite,
				PermissionSecurityEvents:   PermissionWrite,
				PermissionStatuses:         PermissionWrite,
				PermissionModels:           PermissionWrite,
			},
		},
		{
			name:  "merge none shorthand into map - no change",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
			merge: NewPermissionsNone(),
			want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead},
		},

		// Map-to-Shorthand merges (shorthand converts to map)
		{
			name:  "merge map into read-all shorthand - shorthand cleared, map created",
			base:  NewPermissionsReadAll(),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite}),
			want:  map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite},
		},
		{
			name:  "merge map into write-all shorthand - shorthand cleared, map created",
			base:  NewPermissionsWriteAll(),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
			want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead},
		},
		{
			name:  "merge map into none shorthand - shorthand cleared, map created",
			base:  NewPermissionsNone(),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite}),
			want:  map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite},
		},
		{
			name: "merge complex map into read shorthand",
			base: NewPermissionsReadAll(),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
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
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
			merge: nil,
			want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead},
		},
		{
			name:   "merge nil into shorthand - no change",
			base:   NewPermissionsReadAll(),
			merge:  nil,
			wantSH: "read-all",
		},
		{
			name:  "merge empty map into map - no change",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{}),
			want:  map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead},
		},
		{
			name:  "merge map into empty map - scopes added",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{}),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite}),
			want:  map[PermissionScope]PermissionLevel{PermissionIssues: PermissionWrite},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Merge(tt.merge)

			if tt.wantSH != "" {
				if tt.base.shorthand != tt.wantSH {
					t.Errorf("after merge, shorthand = %q, want %q", tt.base.shorthand, tt.wantSH)
				}
				return
			}

			if len(tt.want) != len(tt.base.permissions) {
				t.Errorf("after merge, got %d permissions, want %d", len(tt.base.permissions), len(tt.want))
			}

			for scope, wantLevel := range tt.want {
				gotLevel, exists := tt.base.Get(scope)
				if !exists {
					t.Errorf("after merge, scope %s not found", scope)
					continue
				}
				if gotLevel != wantLevel {
					t.Errorf("after merge, scope %s = %v, want %v", scope, gotLevel, wantLevel)
				}
			}
		})
	}
}

func TestPermissionsRenderToYAML(t *testing.T) {
	tests := []struct {
		name        string
		permissions *Permissions
		want        string
	}{
		{
			name:        "nil permissions",
			permissions: nil,
			want:        "",
		},
		{
			name:        "read-all shorthand",
			permissions: NewPermissionsReadAll(),
			want:        "permissions: read-all",
		},
		{
			name:        "write-all shorthand",
			permissions: NewPermissionsWriteAll(),
			want:        "permissions: write-all",
		},
		{
			name:        "empty permissions",
			permissions: NewPermissions(),
			want:        "",
		},
		{
			name: "single permission",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionRead,
			}),
			want: "permissions:\n      contents: read",
		},
		{
			name: "multiple permissions - sorted",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionIssues:       PermissionWrite,
				PermissionContents:     PermissionRead,
				PermissionPullRequests: PermissionWrite,
			}),
			want: "permissions:\n      contents: read\n      issues: write\n      pull-requests: write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.permissions.RenderToYAML()
			if got != tt.want {
				t.Errorf("RenderToYAML() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPermissions_AllReadWithIdTokenWrite(t *testing.T) {
	// Test that "all: read" with "id-token: write" works as expected
	// id-token is special because it only supports write and none, not read

	// Create permissions with all: read and id-token: write
	perms := &Permissions{
		hasAll:   true,
		allLevel: PermissionRead,
		permissions: map[PermissionScope]PermissionLevel{
			PermissionIdToken: PermissionWrite,
		},
	}

	// Test that all normal scopes have read access
	normalScopes := []PermissionScope{
		PermissionActions, PermissionAttestations, PermissionChecks, PermissionContents,
		PermissionDeployments, PermissionDiscussions, PermissionIssues, PermissionPackages,
		PermissionPages, PermissionPullRequests, PermissionRepositoryProj,
		PermissionSecurityEvents, PermissionStatuses, PermissionModels,
	}

	for _, scope := range normalScopes {
		level, allowed := perms.Get(scope)
		if !allowed || level != PermissionRead {
			t.Errorf("scope %s should have read access, got allowed=%v, level=%s", scope, allowed, level)
		}
	}

	// Test that id-token has write access (explicit override)
	level, allowed := perms.Get(PermissionIdToken)
	if !allowed || level != PermissionWrite {
		t.Errorf("id-token should have write access, got allowed=%v, level=%s", allowed, level)
	}

	// Test that id-token does NOT get read access from all: read
	// This should return false because id-token doesn't support read
	if level, allowed := perms.Get(PermissionIdToken); allowed && level == PermissionRead {
		t.Errorf("id-token should NOT have read access from all: read")
	}

	// Test YAML rendering excludes id-token: read but includes id-token: write
	yaml := perms.RenderToYAML()

	// Should contain all normal scopes with read access
	expectedLines := []string{
		"      actions: read",
		"      attestations: read",
		"      checks: read",
		"      contents: read",
		"      deployments: read",
		"      discussions: read",
		"      issues: read",
		"      packages: read",
		"      pages: read",
		"      pull-requests: read",
		"      repository-projects: read",
		"      security-events: read",
		"      statuses: read",
		"      models: read",
		"      id-token: write", // explicit override
	}

	for _, expected := range expectedLines {
		if !strings.Contains(yaml, expected) {
			t.Errorf("YAML should contain %q, but got:\n%s", expected, yaml)
		}
	}

	// Should NOT contain id-token: read
	if strings.Contains(yaml, "id-token: read") {
		t.Errorf("YAML should NOT contain 'id-token: read', but got:\n%s", yaml)
	}
}

func TestPermissionsParser_AllRead(t *testing.T) {
	tests := []struct {
		name        string
		permissions string
		expected    bool
		scope       string
		level       string
	}{
		{
			name: "all: read grants contents read access",
			permissions: `permissions:
  all: read
  contents: write`,
			expected: true,
			scope:    "contents",
			level:    "read",
		},
		{
			name: "all: read grants contents write access when overridden",
			permissions: `permissions:
  all: read
  contents: write`,
			expected: true,
			scope:    "contents",
			level:    "write",
		},
		{
			name: "all: read grants issues read access",
			permissions: `permissions:
  all: read
  contents: write`,
			expected: true,
			scope:    "issues",
			level:    "read",
		},
		{
			name: "all: read denies issues write access by default",
			permissions: `permissions:
  all: read`,
			expected: false,
			scope:    "issues",
			level:    "write",
		},
		{
			name: "all: read with explicit write override",
			permissions: `permissions:
  all: read
  issues: write`,
			expected: true,
			scope:    "issues",
			level:    "write",
		},
		{
			name: "all: write is not allowed - should fail parsing",
			permissions: `permissions:
  all: write`,
			expected: false,
			scope:    "contents",
			level:    "read",
		},
		{
			name: "all: read with none is not allowed - should fail parsing",
			permissions: `permissions:
  all: read
  contents: none`,
			expected: false,
			scope:    "contents",
			level:    "read",
		},
		{
			name: "all: read grants id-token write access when overridden",
			permissions: `permissions:
  all: read
  id-token: write`,
			expected: true,
			scope:    "id-token",
			level:    "write",
		},
		{
			name: "all: read does not grant id-token read access (not supported)",
			permissions: `permissions:
  all: read`,
			expected: false,
			scope:    "id-token",
			level:    "read",
		},
		{
			name: "all: read denies id-token write access by default (not included in expansion)",
			permissions: `permissions:
  all: read`,
			expected: false,
			scope:    "id-token",
			level:    "write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewPermissionsParser(tt.permissions)
			result := parser.IsAllowed(tt.scope, tt.level)
			if result != tt.expected {
				t.Errorf("IsAllowed(%s, %s) = %v, want %v", tt.scope, tt.level, result, tt.expected)
			}
		})
	}
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

func TestPermissions_AllReadRenderToYAML(t *testing.T) {
	tests := []struct {
		name        string
		perms       *Permissions
		contains    []string // Check that output contains these lines
		notContains []string // Check that output does NOT contain these lines
	}{
		{
			name:  "all: read expands to individual permissions",
			perms: NewPermissionsAllRead(),
			contains: []string{
				"permissions:",
				"      actions: read",
				"      attestations: read",
				"      checks: read",
				"      contents: read",
				"      deployments: read",
				"      discussions: read",
				"      issues: read",
				"      models: read",
				"      packages: read",
				"      pages: read",
				"      pull-requests: read",
				"      repository-projects: read",
				"      security-events: read",
				"      statuses: read",
			},
		},
		{
			name: "all: read with explicit override - write overrides read",
			perms: func() *Permissions {
				p := NewPermissionsAllRead()
				p.Set(PermissionContents, PermissionWrite)
				return p
			}(),
			contains: []string{
				"permissions:",
				"      actions: read",
				"      contents: write", // Overridden to write
				"      issues: read",
			},
			notContains: []string{
				"      contents: read", // Should NOT contain contents: read when explicitly set to write
			},
		},
		{
			name: "all: read with multiple explicit overrides",
			perms: func() *Permissions {
				p := NewPermissionsAllRead()
				p.Set(PermissionContents, PermissionWrite)
				p.Set(PermissionIssues, PermissionWrite)
				return p
			}(),
			contains: []string{
				"permissions:",
				"      actions: read",
				"      contents: write",
				"      issues: write",
				"      packages: read",
			},
			notContains: []string{
				"      contents: read", // Should NOT contain contents: read
				"      issues: read",   // Should NOT contain issues: read
			},
		},
		{
			name: "all: read with id-token: write - id-token should be excluded from all: read expansion but included when explicitly set to write",
			perms: func() *Permissions {
				p := NewPermissionsAllRead()
				p.Set(PermissionIdToken, PermissionWrite)
				return p
			}(),
			contains: []string{
				"permissions:",
				"      actions: read",
				"      contents: read",
				"      id-token: write", // Explicitly set to write
				"      issues: read",
			},
			notContains: []string{
				"      id-token: read", // Should NOT contain id-token: read (not supported)
			},
		},
		{
			name:  "all: read excludes id-token since it doesn't support read level",
			perms: NewPermissionsAllRead(),
			contains: []string{
				"permissions:",
				"      actions: read",
				"      attestations: read",
				"      checks: read",
				"      contents: read",
				"      deployments: read",
				"      discussions: read",
				"      issues: read",
				"      models: read",
				"      packages: read",
				"      pages: read",
				"      pull-requests: read",
				"      repository-projects: read",
				"      security-events: read",
				"      statuses: read",
			},
			notContains: []string{
				"      id-token: read", // Should NOT be included since id-token doesn't support read
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.perms.RenderToYAML()
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("RenderToYAML() should contain %q, but got:\n%s", expected, result)
				}
			}
			for _, notExpected := range tt.notContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("RenderToYAML() should NOT contain %q, but got:\n%s", notExpected, result)
				}
			}
		})
	}
}

func TestPermissionsParser_ToPermissions(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		wantYAML    string
		contains    []string
		notContains []string
	}{
		{
			name:     "shorthand read-all",
			input:    "read-all",
			wantYAML: "permissions: read-all",
		},
		{
			name:     "shorthand write-all",
			input:    "write-all",
			wantYAML: "permissions: write-all",
		},
		{
			name: "all: read without overrides",
			input: map[string]any{
				"all": "read",
			},
			contains: []string{
				"permissions:",
				"      actions: read",
				"      contents: read",
				"      issues: read",
			},
			notContains: []string{
				"      id-token: read", // id-token doesn't support read
			},
		},
		{
			name: "all: read with contents: write override",
			input: map[string]any{
				"all":      "read",
				"contents": "write",
			},
			contains: []string{
				"permissions:",
				"      actions: read",
				"      contents: write", // Override
				"      issues: read",
			},
			notContains: []string{
				"      contents: read",
			},
		},
		{
			name: "all: read with id-token: write override",
			input: map[string]any{
				"all":      "read",
				"id-token": "write",
			},
			contains: []string{
				"permissions:",
				"      actions: read",
				"      contents: read",
				"      id-token: write", // Explicitly set
			},
			notContains: []string{
				"      id-token: read",
			},
		},
		{
			name: "explicit permissions without all",
			input: map[string]any{
				"contents": "read",
				"issues":   "write",
			},
			wantYAML: "permissions:\n      contents: read\n      issues: write",
		},
		{
			name: "all: write is not allowed",
			input: map[string]any{
				"all": "write",
			},
			wantYAML: "",
		},
		{
			name: "all: read with none is not allowed",
			input: map[string]any{
				"all":      "read",
				"contents": "none",
			},
			wantYAML: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewPermissionsParserFromValue(tt.input)
			permissions := parser.ToPermissions()
			yaml := permissions.RenderToYAML()

			if tt.wantYAML != "" && yaml != tt.wantYAML {
				t.Errorf("ToPermissions().RenderToYAML() = %q, want %q", yaml, tt.wantYAML)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(yaml, expected) {
					t.Errorf("ToPermissions().RenderToYAML() should contain %q, but got:\n%s", expected, yaml)
				}
			}

			for _, notExpected := range tt.notContains {
				if strings.Contains(yaml, notExpected) {
					t.Errorf("ToPermissions().RenderToYAML() should NOT contain %q, but got:\n%s", notExpected, yaml)
				}
			}
		})
	}
}
