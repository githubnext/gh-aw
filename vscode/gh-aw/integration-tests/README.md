# VSCode Extension Tests

This directory contains automated tests for the GitHub Agentic Workflows VSCode extension.

## Test Structure

### Unit Tests (Vitest)
- `extension.test.ts` - Tests for extension activation and basic functionality with mocked VSCode APIs
- `extension-functions.test.ts` - Unit tests for individual extension functions

### Integration Tests (Mocha + VSCode Test Electron)
- `suite/extension.test.ts` - Integration tests requiring real VSCode environment
- `suite/language-features.test.ts` - Tests for language features with real VSCode APIs
- `suite/extension-functions.test.ts` - Integration tests for functions in VSCode context

## Running Tests

### Unit Tests (Recommended for development)
```bash
npm test              # Run unit tests with Vitest
npm run test:watch    # Run unit tests in watch mode
npm run test:coverage # Run unit tests with coverage
```

### Integration Tests (Slower, requires VSCode)
```bash
npm run test:integration
```

### From Project Root
```bash
make vscode-test                # Run unit tests
make vscode-test-integration    # Run integration tests
```

## Test Framework

### Unit Tests
- **Vitest** as the test runner
- **Mocked VSCode APIs** for fast unit testing
- **TypeScript** for type safety
- **Coverage reporting** with v8

### Integration Tests
- **Mocha** as the test runner  
- **VSCode Test Electron** for real VSCode environment testing
- **Full VSCode API access** for integration testing

## Test Coverage

The tests cover:
- Extension activation and registration
- Language detection for agentic workflow files
- Hover and completion provider functionality
- Schema validation
- Frontmatter detection
- File path recognition

## Development Workflow

1. **Start with unit tests** - Fast feedback loop with `npm run test:watch`
2. **Use integration tests** - For features that require real VSCode APIs
3. **Run both before commits** - Ensure comprehensive coverage