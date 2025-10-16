# Cache Memory Documentation Validation Tools - Usage Guide

This guide explains how to use the validation tools created for the cache-memory documentation.

## Quick Start

### Validate Documentation Snippets

Run the shell script to validate all code snippets in the documentation:

```bash
./scripts/validate_cache_memory_docs.sh
```

This will:
- Extract all code snippets from the documentation
- Compile each snippet in an isolated environment
- Validate configuration and best practices
- Generate a detailed report

### Run Automated Tests

Run the Go tests as part of the test suite:

```bash
# Run just the documentation validation tests
go test -v -run TestCacheMemoryDocumentation ./pkg/workflow/

# Run all cache memory related tests
go test -v -run TestCacheMemory ./pkg/workflow/

# Run as part of full test suite
make test-unit
```

## Validation Script Details

### What It Validates

The `validate_cache_memory_docs.sh` script performs comprehensive validation:

1. **Snippet Extraction**: Finds all ````aw` code blocks in the documentation
2. **Frontmatter Validation**: Checks YAML structure and required fields
3. **Compilation Testing**: Verifies each snippet can be compiled
4. **Configuration Validation**: 
   - Validates `cache-memory` configuration
   - Checks retention-days is within valid range (1-90)
   - Verifies cache keys are non-empty
   - Validates unique cache IDs in array notation
5. **Best Practices**: Checks adherence to coding standards
6. **Behavior Verification**: Confirms snippets match their documentation

### Output Format

The script provides color-coded output:
- ✓ **Green**: Passed validation
- ✗ **Red**: Failed validation (with details)
- ⊘ **Yellow**: Skipped (expected)

### Generated Reports

After execution, detailed reports are available at:
- `/tmp/gh-aw-docs-validation/reports/validation_report.txt` - Summary report
- `/tmp/gh-aw-docs-validation/reports/snippet_XX_compile.log` - Individual compilation logs

## Go Tests Details

### Test Coverage

The `cache_memory_docs_validation_test.go` file provides:

1. **TestCacheMemoryDocumentationSnippets**: 
   - Validates all snippets from the actual documentation file
   - Skips expected cases (markdown-only, imports requiring external files)
   - Tests compilation and configuration extraction

2. **TestCacheMemoryDocumentationExamples**:
   - Tests specific cache-memory patterns
   - Validates boundary conditions (retention-days 0, 100)
   - Ensures proper error handling

### Integration with CI/CD

These tests run automatically as part of:
```bash
make test-unit      # Unit tests only
make test           # All tests
make agent-finish   # Complete validation
```

## Common Use Cases

### Before Committing Documentation Changes

```bash
# Validate your changes
./scripts/validate_cache_memory_docs.sh

# Run automated tests
go test -v -run TestCacheMemoryDocumentation ./pkg/workflow/
```

### After Modifying Cache-Memory Code

```bash
# Ensure documentation still validates
./scripts/validate_cache_memory_docs.sh

# Run all cache-memory tests
go test -v -run TestCacheMemory ./pkg/workflow/
```

### Debugging a Failing Snippet

```bash
# Run the validation script to see which snippet fails
./scripts/validate_cache_memory_docs.sh

# Check the compilation log
cat /tmp/gh-aw-docs-validation/reports/snippet_XX_compile.log

# Look at the extracted snippet
cat /tmp/gh-aw-docs-validation/snippets/snippet_XX.md
```

## Expected Skips and Failures

Some snippets are expected to be skipped:

1. **Markdown-only snippets**: Natural language instructions without frontmatter
   - Example: "Please save this information to a file..."
   
2. **Import-only snippets**: Fragments that require external shared files
   - Example: Snippets demonstrating import merging behavior

3. **Fragment snippets**: Partial configurations shown for illustration
   - Example: Import file examples

These are marked as `SKIP` in the validation output and do not indicate problems.

## Troubleshooting

### "gh-aw binary not found"

**Solution**: Build the binary first:
```bash
make build
```

### "Documentation file not found"

**Solution**: Run from repository root:
```bash
cd /path/to/gh-aw
./scripts/validate_cache_memory_docs.sh
```

### "Compilation failed - workflow not found"

This is expected for import snippets. Check if the snippet uses `imports:` field without external files present.

### Tests fail with "Failed to parse workflow"

Check if the error mentions:
- "failed to resolve import" - Expected for import examples
- "no markdown content found" - Expected for frontmatter-only fragments
- Other errors - May indicate a real issue to investigate

## Adding New Validation Checks

### To the Shell Script

Edit `scripts/validate_cache_memory_docs.sh` and add validation logic to the relevant function:

```bash
validate_cache_memory_config() {
    # Add your validation logic here
    # Return 0 for success, 1 for failure
}
```

### To the Go Tests

Edit `pkg/workflow/cache_memory_docs_validation_test.go` and add test cases:

```go
func TestCacheMemoryNewFeature(t *testing.T) {
    // Your test implementation
}
```

## Maintenance

### Updating for New Documentation

The tools automatically discover all snippets in the documentation. When adding new examples:

1. Add them to `docs/src/content/docs/reference/cache-memory.md`
2. Run the validation script to verify
3. Update this guide if new patterns are introduced

### Keeping Tools in Sync

When cache-memory functionality changes:

1. Update the documentation examples
2. Run validation tools to catch issues
3. Update validation logic if needed
4. Update test expectations if behavior changes

## Best Practices

1. **Run validation before committing**: Catch issues early
2. **Check both script and tests**: They validate different aspects
3. **Review detailed logs**: Understand why validation failed
4. **Keep examples realistic**: Use patterns users will actually use
5. **Document expected behavior**: Help users understand what's valid

## Related Documentation

- [Cache Memory Documentation](src/content/docs/reference/cache-memory.md) - The validated documentation
- [Validation Report](CACHE_MEMORY_VALIDATION_REPORT.md) - Detailed validation results
- [Testing Guide](../TESTING.md) - Overall testing strategy
