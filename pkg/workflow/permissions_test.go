package workflow

import (
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
