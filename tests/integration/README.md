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

**Proof of Concept** - Basic tests demonstrating tuistory integration.

The current tests validate:
- Tuistory can launch and interact with the gh-aw binary
- Terminal text can be captured and verified
- Test infrastructure is in place

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
