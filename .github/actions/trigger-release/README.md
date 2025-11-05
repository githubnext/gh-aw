# Trigger Release Action

A composite GitHub Action that safely triggers a release with comprehensive checks and automated testing.

## Features

- ✅ **Permission Validation**: Ensures only users with admin or maintainer permissions can trigger releases
- ✅ **Fork Protection**: Prevents releases from being triggered on forked repositories
- ✅ **Branch Protection**: Ensures releases are only triggered from the `main` branch
- ✅ **Automated Testing**: Runs full build and test suite before releasing
- ✅ **Non-Interactive**: Uses `--yes` flag for automated, non-interactive releases

## Usage

### As a Composite Action

Use this action in your workflow:

```yaml
name: Manual Release

on:
  workflow_dispatch:

permissions:
  contents: write
  packages: write
  id-token: write
  attestations: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v5
        with:
          fetch-depth: 0

      - name: Trigger Release
        uses: ./.github/actions/trigger-release
```

### Direct Call from Workflow

You can also call the action directly in your existing release workflow:

```yaml
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0
          
      - uses: ./.github/actions/trigger-release
```

## Requirements

The action requires:

1. **Permissions**: The workflow must have appropriate permissions set
2. **Checkout**: Must checkout the repository with full history (`fetch-depth: 0`)
3. **User Permissions**: Triggering user must have admin or maintainer role
4. **Repository**: Must not be a fork
5. **Branch**: Must be running on the `main` branch

## What It Does

The action performs the following steps in order:

1. **Check User Permissions**: Validates that the user triggering the workflow has admin or maintainer permissions
2. **Check Repository Status**: Ensures the repository is not a fork
3. **Check Branch**: Verifies the workflow is running on the `main` branch
4. **Setup Environment**: Sets up Node.js and Go build environments
5. **Install Dependencies**: Installs npm dependencies
6. **Build**: Builds the project using `make build`
7. **Test**: Runs the test suite using `make test`
8. **Release**: Executes `make release` with `YES=1` flag (non-interactive mode)

## Error Handling

The action will fail with descriptive error messages if:

- User doesn't have admin or maintainer permissions
- Repository is a fork
- Not running on the `main` branch
- Build fails
- Tests fail
- Release process encounters errors

## Examples

### Workflow Dispatch (Manual Trigger)

```yaml
name: Manual Release

on:
  workflow_dispatch:

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0
          
      - uses: ./.github/actions/trigger-release
```

### Scheduled Release

```yaml
name: Weekly Release

on:
  schedule:
    - cron: '0 9 * * 1'  # Every Monday at 9 AM UTC

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0
          
      - uses: ./.github/actions/trigger-release
```

## Related

- See [release.yml](../../workflows/release.yml) for the tag-based release workflow
- See [Makefile](../../../Makefile) for the `release` target definition
- See [scripts/changeset.js](../../../scripts/changeset.js) for the release script implementation
