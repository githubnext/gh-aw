# gh-aw Integration Tests with Tuistory

This directory contains end-to-end integration tests for the interactive commands in gh-aw, using [tuistory](https://github.com/remorses/tuistory) for terminal interaction testing.

## Prerequisites

- Node.js 18+ and npm
- gh-aw binary built (`make build` in project root)
- For full interactive tests: GitHub CLI authentication

## Setup

```bash
npm install
```

## Running Tests

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with UI
npm run test:ui
```

## Test Structure

- `setup.js` - Shared test utilities and helpers
- `add-interactive.test.js` - Tests for `gh aw add` interactive command
- `package.json` - Test dependencies (tuistory, vitest)

## Current Status

**⚠️ BLOCKED - Tuistory Package Not Available**

The tuistory npm package (v1.0.0) is published but contains no actual code - only a package.json file. This appears to be an incomplete publication.

**Investigation Complete**: Test infrastructure and approach documented in [docs/testing/tuistory-investigation.md](../../docs/testing/tuistory-investigation.md).

**Next Steps**: 
- Monitor tuistory repository for updates
- Consider alternative testing approaches
- Document manual testing procedures in the meantime

## Future Work

See [docs/testing/tuistory-investigation.md](../../docs/testing/tuistory-investigation.md) for the full implementation plan, including:

- Full interactive flow tests
- Engine selection and API key handling
- PR creation and secret management
- Error scenario coverage
- CI integration

## Notes

- Tests use `GO_TEST_MODE=false` to allow interactive mode
- Terminal colors are disabled (`NO_COLOR=1`) for easier text matching
- Each test gets a fresh temporary git repository
- Binary is automatically built if missing
