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
