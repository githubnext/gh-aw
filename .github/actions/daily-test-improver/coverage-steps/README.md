# Daily Test Improver Coverage Steps

Runs comprehensive test coverage analysis for gh-aw repository and writes a comprehensive step summary with collapsible details.

## Features

- ✅ **Step Summary**: Real-time coverage analysis in GitHub Actions UI
- ✅ **Collapsible Details**: HTML details/summary tags for progressive disclosure
- ✅ **Coverage Metrics**: Package-level breakdown, low coverage areas, zero coverage functions
- ✅ **Status Indicators**: Visual status (✅/❌/⚠️) for each step
- ✅ **Timing Information**: Duration tracking for all steps
- ✅ **Markdown Best Practices**: Follows same patterns as agentic workflows

## Usage

```yaml
- name: Run Coverage Analysis
  uses: ./.github/actions/daily-test-improver/coverage-steps
```

## What It Does

1. Verifies Go and Node.js installations
2. Installs dependencies
3. Runs Go tests with coverage
4. Generates coverage reports and breakdowns
5. Identifies low and zero coverage areas
6. Runs JavaScript tests with coverage (optional)
7. Prepares coverage artifacts for download

## Step Summary Example

The action generates a beautiful step summary like:

```markdown
## 🧪 Test Coverage Analysis

**Status:** ✅ All steps completed
**Total Duration:** 156.7s
**Steps:** 11/11 successful

### Coverage Summary

**Go Overall Coverage:** 68.4%

<details>
<summary><b>📦 Package Coverage Breakdown (42 packages)</b></summary>

| Package | Coverage |
|---------|----------|
| github.com/githubnext/gh-aw/pkg/cli | 72.5% |
...
</details>

<details>
<summary><b>⚠️  Low Coverage Areas (30 functions)</b></summary>

| Coverage | Function |
|----------|----------|
| 12.5% | pkg/cli/init.go:InitCommand |
...
</details>

**Functions with Zero Coverage:** 87

### Execution Steps
| Step | Duration | Status |
|------|----------|--------|
| Run Go Tests with Coverage | 89.3s | ✅ |
...
```

## Development

### Building

```bash
cd .github/actions/daily-test-improver/coverage-steps
npm install
npm run build
```

This will bundle the action using `@vercel/ncc`.

### Dependencies

- `@actions/core` - GitHub Actions core library
- `@actions/exec` - Execute commands and capture output

## License

MIT
