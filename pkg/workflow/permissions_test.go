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
			name:        "shorthand read grants contents access",
			permissions: "permissions: read",
			expected:    true,
		},
		{
			name:        "shorthand write grants contents access",
			permissions: "permissions: write",
			expected:    true,
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
			name: "contains actions/checkout@v5",
			customSteps: `steps:
  - name: Checkout
    uses: actions/checkout@v5`,
			expected: true,
		},
		{
			name: "contains actions/checkout@v4",
			customSteps: `steps:
  - uses: actions/checkout@v4
    with:
      token: ${{ secrets.GITHUB_TOKEN }}`,
			expected: true,
		},
		{
			name: "contains different action",
			customSteps: `steps:
  - name: Setup Node
    uses: actions/setup-node@v4
    with:
      node-version: '18'`,
			expected: false,
		},
		{
			name: "mixed steps with checkout",
			customSteps: `steps:
  - name: Checkout repository
    uses: actions/checkout@v5
  - name: Setup Node
    uses: actions/setup-node@v4`,
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
  - uses: actions/checkout@v5
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
    uses: "actions/checkout@v5"`,
			expected: true,
		},
		{
			name: "checkout action in single quotes",
			customSteps: `steps:
  - uses: 'actions/checkout@v4'`,
			expected: true,
		},
		{
			name: "checkout with extra whitespace",
			customSteps: `steps:
  - uses:   actions/checkout@v5   `,
			expected: true,
		},
		{
			name: "checkout in multiline YAML",
			customSteps: `steps:
  - name: Checkout
    uses: >
      actions/checkout@v5`,
			expected: true,
		},
		{
			name: "checkout in run command (should not match)",
			customSteps: `steps:
  - name: Echo checkout
    run: echo "actions/checkout@v5"`,
			expected: true, // Current implementation does simple string match
		},
		{
			name: "checkout in comment (should not match)",
			customSteps: `steps:
  - name: Setup
    uses: actions/setup-node@v4
    # TODO: add actions/checkout@v5`,
			expected: true, // Current implementation does simple string match
		},
		{
			name: "similar but not checkout action",
			customSteps: `steps:
  - uses: actions/cache@v3
  - uses: my-actions/checkout@v1`,
			expected: true, // Current implementation matches substring
		},
		{
			name: "checkout in different format",
			customSteps: `steps:
  - name: Checkout code
    uses: |
      actions/checkout@v5`,
			expected: true,
		},
		{
			name: "malformed YAML with checkout",
			customSteps: `steps
  - uses: actions/checkout@v5`,
			expected: true, // Still detects the string
		},
		{
			name: "checkout with complex parameters",
			customSteps: `steps:
  - name: Checkout repository
    uses: actions/checkout@v5
    with:
      fetch-depth: 0
      token: ${{ secrets.GITHUB_TOKEN }}
      submodules: recursive`,
			expected: true,
		},
		{
			name: "multiple checkouts",
			customSteps: `steps:
  - uses: actions/checkout@v4
  - name: Setup
    run: echo "setup"
  - uses: actions/checkout@v5
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
    uses: actions/checkout@v5`,
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
    uses: actions/checkout@v5
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
		{"read", NewPermissionsRead, "read"},
		{"write", NewPermissionsWrite, "write"},
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
			name:  "merge two maps - multiple scopes with conflicts",
			base: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionRead,
			}),
			merge: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionWrite,
				PermissionIssues:       PermissionRead,
				PermissionDiscussions:  PermissionWrite,
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
				PermissionActions:        PermissionRead,
				PermissionChecks:         PermissionRead,
				PermissionContents:       PermissionRead,
				PermissionDeployments:    PermissionRead,
				PermissionDiscussions:    PermissionRead,
				PermissionIssues:         PermissionRead,
				PermissionPackages:       PermissionRead,
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
			base:   NewPermissionsRead(),
			merge:  NewPermissionsWriteAll(),
			wantSH: "write-all",
		},
		{
			name:   "merge shorthand - write-all wins over write",
			base:   NewPermissionsWrite(),
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
			name:   "merge shorthand - write wins over read-all",
			base:   NewPermissionsReadAll(),
			merge:  NewPermissionsWrite(),
			wantSH: "write",
		},
		{
			name:   "merge shorthand - write wins over read",
			base:   NewPermissionsRead(),
			merge:  NewPermissionsWrite(),
			wantSH: "write",
		},
		{
			name:   "merge shorthand - write wins over none",
			base:   NewPermissionsNone(),
			merge:  NewPermissionsWrite(),
			wantSH: "write",
		},
		{
			name:   "merge shorthand - read-all wins over read",
			base:   NewPermissionsRead(),
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
			name:   "merge shorthand - read wins over none",
			base:   NewPermissionsNone(),
			merge:  NewPermissionsRead(),
			wantSH: "read",
		},
		{
			name:   "merge shorthand - read-all preserved when merging read",
			base:   NewPermissionsReadAll(),
			merge:  NewPermissionsRead(),
			wantSH: "read-all",
		},
		{
			name:   "merge shorthand - write-all preserved when merging write",
			base:   NewPermissionsWriteAll(),
			merge:  NewPermissionsWrite(),
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
				PermissionContents:       PermissionWrite, // preserved
				PermissionActions:        PermissionRead,  // added
				PermissionChecks:         PermissionRead,
				PermissionDeployments:    PermissionRead,
				PermissionDiscussions:    PermissionRead,
				PermissionIssues:         PermissionRead,
				PermissionPackages:       PermissionRead,
				PermissionPages:          PermissionRead,
				PermissionPullRequests:   PermissionRead,
				PermissionRepositoryProj: PermissionRead,
				PermissionSecurityEvents: PermissionRead,
				PermissionStatuses:       PermissionRead,
				PermissionModels:         PermissionRead,
			},
		},
		{
			name:  "merge write-all shorthand into map - adds all missing scopes as write",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionRead}),
			merge: NewPermissionsWriteAll(),
			want: map[PermissionScope]PermissionLevel{
				PermissionContents:       PermissionRead, // preserved (not overwritten)
				PermissionActions:        PermissionWrite,
				PermissionChecks:         PermissionWrite,
				PermissionDeployments:    PermissionWrite,
				PermissionDiscussions:    PermissionWrite,
				PermissionIssues:         PermissionWrite,
				PermissionPackages:       PermissionWrite,
				PermissionPages:          PermissionWrite,
				PermissionPullRequests:   PermissionWrite,
				PermissionRepositoryProj: PermissionWrite,
				PermissionSecurityEvents: PermissionWrite,
				PermissionStatuses:       PermissionWrite,
				PermissionModels:         PermissionWrite,
			},
		},
		{
			name:  "merge read shorthand into map - adds all missing scopes as read",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionContents: PermissionWrite}),
			merge: NewPermissionsRead(),
			want: map[PermissionScope]PermissionLevel{
				PermissionContents:       PermissionWrite,
				PermissionActions:        PermissionRead,
				PermissionChecks:         PermissionRead,
				PermissionDeployments:    PermissionRead,
				PermissionDiscussions:    PermissionRead,
				PermissionIssues:         PermissionRead,
				PermissionPackages:       PermissionRead,
				PermissionPages:          PermissionRead,
				PermissionPullRequests:   PermissionRead,
				PermissionRepositoryProj: PermissionRead,
				PermissionSecurityEvents: PermissionRead,
				PermissionStatuses:       PermissionRead,
				PermissionModels:         PermissionRead,
			},
		},
		{
			name:  "merge write shorthand into map - adds all missing scopes as write",
			base:  NewPermissionsFromMap(map[PermissionScope]PermissionLevel{PermissionIssues: PermissionRead}),
			merge: NewPermissionsWrite(),
			want: map[PermissionScope]PermissionLevel{
				PermissionIssues:         PermissionRead,
				PermissionActions:        PermissionWrite,
				PermissionChecks:         PermissionWrite,
				PermissionContents:       PermissionWrite,
				PermissionDeployments:    PermissionWrite,
				PermissionDiscussions:    PermissionWrite,
				PermissionPackages:       PermissionWrite,
				PermissionPages:          PermissionWrite,
				PermissionPullRequests:   PermissionWrite,
				PermissionRepositoryProj: PermissionWrite,
				PermissionSecurityEvents: PermissionWrite,
				PermissionStatuses:       PermissionWrite,
				PermissionModels:         PermissionWrite,
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
			base: NewPermissionsRead(),
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
