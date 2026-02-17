//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFeaturesFromContent_ValidFeatures(t *testing.T) {
	content := `---
features:
  feature1: true
  feature2: false
  feature3: "enabled"
---

# Test Workflow
`
	result, err := extractFeaturesFromContent(content)
	require.NoError(t, err, "Should not error on valid features")
	assert.NotEmpty(t, result, "Should return non-empty result")
	assert.Contains(t, result, "feature1", "Should contain feature1")
	assert.Contains(t, result, "feature2", "Should contain feature2")
	assert.Contains(t, result, "feature3", "Should contain feature3")
}

func TestExtractFeaturesFromContent_NoFeatures(t *testing.T) {
	content := `---
engine: copilot
on: issues
---

# Test Workflow
`
	result, err := extractFeaturesFromContent(content)
	require.NoError(t, err, "Should not error when no features")
	assert.Equal(t, "{}", result, "Should return empty object when no features")
}

func TestExtractFeaturesFromContent_EmptyFeatures(t *testing.T) {
	content := `---
features: {}
---

# Test Workflow
`
	result, err := extractFeaturesFromContent(content)
	require.NoError(t, err, "Should not error on empty features")
	assert.Equal(t, "{}", result, "Should return empty object for empty features")
}

func TestExtractFeaturesFromContent_NoFrontmatter(t *testing.T) {
	content := `# Test Workflow

This workflow has no frontmatter.
`
	result, err := extractFeaturesFromContent(content)
	require.NoError(t, err, "Should not error when no frontmatter")
	assert.Equal(t, "{}", result, "Should return empty object when no frontmatter")
}

func TestExtractFeaturesFromContent_ComplexFeatures(t *testing.T) {
	content := `---
features:
  string-feature: "value"
  bool-feature: true
  number-feature: 42
  nested-feature:
    key1: value1
    key2: value2
  array-feature:
    - item1
    - item2
---

# Test Workflow
`
	result, err := extractFeaturesFromContent(content)
	require.NoError(t, err, "Should not error on complex features")
	assert.NotEmpty(t, result, "Should return non-empty result")
	assert.Contains(t, result, "string-feature", "Should contain string-feature")
	assert.Contains(t, result, "bool-feature", "Should contain bool-feature")
	assert.Contains(t, result, "number-feature", "Should contain number-feature")
	assert.Contains(t, result, "nested-feature", "Should contain nested-feature")
	assert.Contains(t, result, "array-feature", "Should contain array-feature")
}

func TestExtractFeaturesFromContent_MalformedYAML(t *testing.T) {
	content := `---
features: [this is not valid yaml
---

# Test Workflow
`
	result, err := extractFeaturesFromContent(content)
	// Should return empty object on error, not propagate error
	require.NoError(t, err, "Should not error on malformed YAML")
	assert.Equal(t, "{}", result, "Should return empty object on malformed YAML")
}

func TestExtractFeaturesFromContent_FeaturesWithSpecialCharacters(t *testing.T) {
	content := `---
features:
  feature-with-dash: true
  feature_with_underscore: false
  feature.with.dot: "enabled"
  feature:with:colon: 123
---

# Test Workflow
`
	result, err := extractFeaturesFromContent(content)
	require.NoError(t, err, "Should not error on features with special characters")
	assert.NotEmpty(t, result, "Should return non-empty result")
	// The exact keys depend on YAML parsing, but we should get some features
	assert.NotEqual(t, "{}", result, "Should contain features")
}

func TestExtractFeaturesFromContent_OnlyFeaturesInFrontmatter(t *testing.T) {
	content := `---
features:
  test-feature: true
---

# Test Workflow
`
	result, err := extractFeaturesFromContent(content)
	require.NoError(t, err, "Should not error")
	assert.NotEmpty(t, result, "Should return features")
	assert.Contains(t, result, "test-feature", "Should contain test-feature")
}

func TestExtractFeaturesFromContent_MultipleFrontmatterFields(t *testing.T) {
	content := `---
engine: copilot
on: issues
features:
  test-feature: true
  another-feature: false
permissions:
  contents: read
---

# Test Workflow
`
	result, err := extractFeaturesFromContent(content)
	require.NoError(t, err, "Should not error")
	assert.NotEmpty(t, result, "Should return features")
	assert.Contains(t, result, "test-feature", "Should contain test-feature")
	assert.Contains(t, result, "another-feature", "Should contain another-feature")
	// Should NOT contain other frontmatter fields
	assert.NotContains(t, result, "engine", "Should not contain non-feature fields")
	assert.NotContains(t, result, "permissions", "Should not contain non-feature fields")
}

func TestExtractFeaturesFromContent_BooleanValues(t *testing.T) {
	content := `---
features:
  enabled-feature: true
  disabled-feature: false
  yes-feature: yes
  no-feature: no
  on-feature: on
  off-feature: off
---

# Test Workflow
`
	result, err := extractFeaturesFromContent(content)
	require.NoError(t, err, "Should not error")
	assert.NotEmpty(t, result, "Should return features")
	// All these should be parsed as boolean features
	assert.Contains(t, result, "enabled-feature", "Should contain enabled-feature")
	assert.Contains(t, result, "disabled-feature", "Should contain disabled-feature")
}

func TestExtractFeaturesFromContent_NumericValues(t *testing.T) {
	content := `---
features:
  int-feature: 42
  float-feature: 3.14
  negative-feature: -10
  zero-feature: 0
---

# Test Workflow
`
	result, err := extractFeaturesFromContent(content)
	require.NoError(t, err, "Should not error")
	assert.NotEmpty(t, result, "Should return features")
	assert.Contains(t, result, "int-feature", "Should contain int-feature")
	assert.Contains(t, result, "float-feature", "Should contain float-feature")
	assert.Contains(t, result, "negative-feature", "Should contain negative-feature")
	assert.Contains(t, result, "zero-feature", "Should contain zero-feature")
}
