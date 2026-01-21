# Daily Perf Improver Build Steps

Sets up the performance development environment for gh-aw and writes a comprehensive step summary with collapsible details.

## Features

- ✅ **Step Summary**: Real-time progress in GitHub Actions UI
- ✅ **Collapsible Details**: HTML details/summary tags for progressive disclosure
- ✅ **Status Indicators**: Visual status (✅/❌) for each step
- ✅ **Timing Information**: Duration tracking for all steps
- ✅ **Markdown Best Practices**: Follows same patterns as agentic workflows

## Usage

```yaml
- name: Setup Performance Environment
  uses: ./.github/actions/daily-perf-improver/build-steps
```

## What It Does

1. Verifies Go and Node.js installations
2. Installs Go dependencies and development tools
3. Installs npm global and local dependencies
4. Downloads GitHub Actions schema
5. Builds gh-aw binary
6. Creates performance testing directories

## Step Summary Example

The action generates a beautiful step summary like:

```markdown
## 📦 Performance Development Environment Setup

**Status:** ✅ All steps completed
**Total Duration:** 225.4s
**Steps:** 11/11 successful

### Summary
| Step | Duration | Status |
|------|----------|--------|
| Install Go Dependencies | 45.2s | ✅ |
| Build gh-aw Binary | 78.9s | ✅ |
...

### Detailed Results
<details>
<summary><b>✅ Install Go Dependencies (45.2s)</b></summary>

[detailed output]
</details>
```

## Development

### Building

```bash
cd .github/actions/daily-perf-improver/build-steps
npm install
npm run build
```

This will bundle the action using `@vercel/ncc`.

### Dependencies

- `@actions/core` - GitHub Actions core library
- `@actions/exec` - Execute commands and capture output

## License

MIT
