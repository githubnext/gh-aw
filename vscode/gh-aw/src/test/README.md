# VSCode Extension Tests

This directory contains automated tests for the GitHub Agentic Workflows VSCode extension.

## Test Structure

- `extension.test.ts` - Tests for extension activation and basic functionality
- `language-features.test.ts` - Tests for language features like hover, completion, and validation
- `extension-functions.test.ts` - Unit tests for individual extension functions

## Running Tests

From the VSCode extension directory:

```bash
npm test
```

Or from the project root:

```bash
make vscode-test
```

## Test Framework

The tests use:
- Mocha as the test runner
- VSCode Test Electron for integration testing
- TypeScript for type safety

## Test Coverage

The tests cover:
- Extension activation and registration
- Language detection for agentic workflow files
- Hover and completion provider functionality
- Schema validation
- Frontmatter detection
- File path recognition