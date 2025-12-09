# Golden Tests for `gh aw add`

This directory contains golden test files for the `gh aw add` command. Golden tests help ensure that generated scaffolding and compiled workflows remain consistent and that any changes are intentional and reviewed.

## What are Golden Tests?

Golden tests (also called snapshot tests) capture the expected output of a command and compare future runs against this "golden" output. This helps:

- Detect unintentional changes to generated files
- Ensure backward compatibility
- Review changes to scaffolding and templates
- Document expected behavior through examples

## Test Structure

Each test case has its own subdirectory containing:

- `*.md` - The generated workflow markdown files
- `*.lock.yml` - The compiled GitHub Actions YAML files

### Current Test Cases

1. **`add_simple_workflow/`** - Basic workflow addition
2. **`add_workflow_with_custom_name/`** - Using `-n` flag to customize workflow name
3. **`add_workflow_to_subdirectory/`** - Using `--dir` flag to add to subdirectory (e.g., `shared/`)
4. **`add_workflow_with_numbered_copies/`** - Using `-c` flag to create multiple numbered copies

## Running the Tests

### Normal Test Run

```bash
# Run all golden tests
go test -v -timeout=5m -tags 'integration' -run='^TestAddCommandGolden$' ./pkg/cli

# Run a specific test case
go test -v -timeout=5m -tags 'integration' -run='^TestAddCommandGolden/add_simple_workflow$' ./pkg/cli
```

### Updating Golden Files

When you intentionally change the workflow generation or compilation logic, you need to update the golden files:

```bash
# Update all golden files
UPDATE_GOLDEN=1 go test -v -timeout=5m -tags 'integration' -run='^TestAddCommandGolden$' ./pkg/cli

# Update specific test case
UPDATE_GOLDEN=1 go test -v -timeout=5m -tags 'integration' -run='^TestAddCommandGolden/add_simple_workflow$' ./pkg/cli
```

**Important:** Always review the changes to golden files before committing them!

```bash
# Review changes
git diff pkg/cli/testdata/add_golden/
```

## CI Integration

The golden tests are automatically run as part of the CI workflow. Any discrepancies between generated files and golden files will cause the CI to fail, ensuring that all changes are intentional and reviewed.

## Golden File Normalization

Golden files are normalized before comparison to remove dynamic content:

- Timestamps and dates are filtered out
- Version information is removed
- Commit SHAs are replaced with `<commit-sha>` placeholders

This ensures tests remain stable across different environments and time periods.

## Adding New Test Cases

To add a new golden test case:

1. Add a new test struct to `add_golden_test.go`
2. Define the test parameters (source workflow, flags, expected files)
3. Run with `UPDATE_GOLDEN=1` to generate golden files
4. Review the generated files
5. Run without `UPDATE_GOLDEN` to verify the test passes
6. Commit both the test code and golden files

Example:

```go
{
    name:         "add_workflow_with_append",
    workflowName: "test-workflow",
    sourceWorkflow: `...workflow content...`,
    // Add test-specific parameters here
    goldenFiles: []string{
        "test-workflow.md",
        "test-workflow.lock.yml",
    },
},
```

## Best Practices

1. **Keep golden files minimal** - Use simple, focused workflows for testing
2. **Review all changes** - Always examine diffs before committing updated golden files
3. **Test intentionality** - Only update golden files when changes are intentional
4. **Document changes** - Include context in commit messages about why golden files were updated
5. **Run locally first** - Test changes locally before pushing to CI

## Troubleshooting

### Test fails with "does not match golden file"

This means the generated output differs from the expected output. Either:
- Your changes are unintentional → fix the code
- Your changes are intentional → update golden files with `UPDATE_GOLDEN=1`

### Golden files are huge

Golden files include compiled `.lock.yml` files which can be large (100KB+). This is expected as they contain the full GitHub Actions workflow with all dependencies bundled.

### Tests pass locally but fail in CI

Ensure you've committed all golden files and that they're not in `.gitignore`. Golden files must be tracked in version control.
