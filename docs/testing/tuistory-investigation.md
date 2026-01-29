# Tuistory Investigation for Testing "add" Interactive Command

## Overview

This document investigates using [tuistory](https://github.com/remorses/tuistory) to test the `gh aw add` interactive command implemented in `pkg/cli/add_interactive.go`.

## Background

### Current State

The `add` command in gh-aw has an interactive mode that:
- Walks users through adding agentic workflows to their repository
- Uses `charmbracelet/huh` for interactive forms
- Prompts for AI engine selection, API keys, and GitHub configuration
- Creates pull requests and manages secrets
- Currently cannot be tested in automated environments (CI/tests)

The interactive mode is triggered when:
- User runs `gh aw add owner/repo/workflow` in a TTY
- No automation flags are provided (`--pr`, `--force`, etc.)
- Not in CI environment (`CI` env var not set)
- Not in test mode (`GO_TEST_MODE` env var not set)

### Current Testing Approach

Existing tests in `pkg/cli/interactive_test.go`:
- Focus on unit testing of individual functions
- Cannot test the interactive flow end-to-end
- Skip interactive tests in CI (`GO_TEST_MODE` check)
- Test business logic but not user interactions

Example from current tests:
```go
func TestCreateWorkflowInteractively_InAutomatedEnvironment(t *testing.T) {
    os.Setenv("GO_TEST_MODE", "true")
    
    err := CreateWorkflowInteractively("test-workflow", false, false)
    if err == nil {
        t.Error("Expected error in automated environment, got nil")
    }
}
```

## About Tuistory

### What is Tuistory?

Tuistory is described as "Playwright for terminal user interfaces" - it enables end-to-end testing of terminal applications by:

1. **Launching terminal sessions** with specific commands
2. **Interacting with the terminal** (typing, pressing keys, clicking)
3. **Waiting for text patterns** (including regex support)
4. **Capturing snapshots** of terminal output
5. **Managing sessions** independently

### Key Features

- **Session Management**: Multiple named sessions can run concurrently
- **Text Matching**: Support for exact strings and regex patterns
- **Keyboard Input**: Full keyboard support including modifiers (Ctrl, Alt, Shift)
- **Visual Testing**: Snapshot capabilities for terminal output
- **Timeout Handling**: Configurable timeouts for async operations
- **Daemon Architecture**: Background daemon manages terminal sessions

### API Examples

```javascript
import { launchTerminal } from 'tuistory'

// Launch a terminal session
const session = await launchTerminal({
  command: 'gh',
  args: ['aw', 'add', 'githubnext/agentics/ci-doctor'],
  cols: 100,
  rows: 30,
})

// Wait for prompts
await session.waitForText('Which AI engine', { timeout: 10000 })

// Type responses
await session.type('claude')
await session.press('enter')

// Wait for next prompt
await session.waitForText('Enter API key')
await session.type('sk-test-key')
await session.press('enter')

// Verify output
const finalText = await session.text()
expect(finalText).toContain('Successfully added workflow')

// Cleanup
session.close()
```

## Evaluation of Tuistory for gh-aw

### Pros

1. **True End-to-End Testing**
   - Tests the actual binary as users would run it
   - No need to mock terminal interactions
   - Validates the complete user experience

2. **Language Agnostic**
   - gh-aw is Go, but tests can be in JavaScript/TypeScript
   - Follows the pattern used in gh-aw for JavaScript tests (see `actions/setup/js/*.test.cjs`)

3. **CI Compatible**
   - Runs in headless environments
   - No TTY required (creates pseudo-terminals)
   - Can be integrated into existing CI workflows

4. **Comprehensive Interaction Testing**
   - Test form navigation (arrow keys, tab)
   - Test input validation
   - Test error handling and recovery
   - Test multi-step workflows

5. **Visual Regression Testing**
   - Snapshot testing for terminal output
   - Can verify formatting, colors, and layout
   - Detect unintended UI changes

### Cons

1. **Additional Test Infrastructure**
   - Requires Node.js/npm in test environment (already present)
   - Adds JavaScript/TypeScript test files to Go project
   - Need to maintain both Go and JS test suites

2. **Complexity**
   - More moving parts than unit tests
   - Harder to debug failures
   - Longer test execution time

3. **Environment Dependencies**
   - Requires building the binary first
   - May need GitHub CLI authentication
   - Filesystem and git repository setup

4. **Test Stability**
   - Timing-sensitive (need proper waits)
   - Text-matching can be brittle
   - May need careful handling of prompts

### Comparison with Alternatives

#### Alternative 1: Refactor for Testability
**Approach**: Extract business logic from interactive code, mock huh forms

**Pros**:
- Pure Go tests
- Faster execution
- Easier to debug

**Cons**:
- Doesn't test actual user interaction
- Won't catch UI/UX issues
- Significant refactoring required
- Loses validation of real-world usage

#### Alternative 2: Expect-like Libraries for Go
**Options**: `github.com/Netflix/go-expect`, `github.com/hinshun/vt10x`

**Pros**:
- Go-native solutions
- Direct integration with test suite

**Cons**:
- Less mature than tuistory
- More verbose API
- Limited documentation
- Not actively maintained (go-expect last updated 2020)

#### Alternative 3: Manual Testing Only
**Approach**: Document manual test procedures

**Pros**:
- No additional infrastructure
- Simpler codebase

**Cons**:
- No automation
- Regression risk
- Time-consuming
- Not scalable

## Recommendation

### Proposed Approach: Hybrid Testing Strategy

Implement a **two-tier testing approach**:

1. **Unit Tests (Go)** - Current approach, enhanced
   - Test individual functions in isolation
   - Mock external dependencies (gh CLI, git, filesystem)
   - Fast, reliable, easy to debug
   - Cover edge cases and error conditions

2. **Integration Tests (Tuistory + JavaScript)** - New addition
   - Test critical user flows end-to-end
   - Focus on happy paths and common scenarios
   - Run less frequently (e.g., pre-release, on main branch)
   - Validate user experience and terminal interactions

### Implementation Plan

#### Phase 1: Setup (Week 1)
1. Add tuistory as dev dependency to project
2. Create test infrastructure:
   - `tests/integration/` directory for tuistory tests
   - Helper utilities for common operations
   - CI workflow integration
3. Document testing approach

#### Phase 2: Core Tests (Week 2)
1. Test basic interactive flow:
   - Add workflow with default options
   - Engine selection
   - API key input
2. Test error scenarios:
   - Invalid repository
   - Missing authentication
   - Network failures
3. Test navigation:
   - Arrow key selection
   - Tab completion
   - Back/cancel operations

#### Phase 3: Advanced Tests (Week 3)
1. Test PR creation flow
2. Test secret management
3. Test workflow execution
4. Add snapshot tests for output validation

### Example Test Structure

```
tests/
├── integration/
│   ├── package.json           # tuistory and test dependencies
│   ├── setup.js               # Test helpers and utilities
│   ├── add-interactive.test.js # Main test file
│   └── fixtures/              # Test data and expected outputs
│       ├── sample-repo/
│       └── expected-output/
```

### Sample Test Case

```javascript
// tests/integration/add-interactive.test.js
import { describe, test, expect, beforeEach, afterEach } from 'vitest'
import { launchTerminal } from 'tuistory'
import { setupTestRepo, cleanupTestRepo } from './setup.js'

describe('gh aw add interactive', () => {
  let testRepo
  
  beforeEach(async () => {
    testRepo = await setupTestRepo()
  })
  
  afterEach(async () => {
    await cleanupTestRepo(testRepo)
  })
  
  test('adds workflow with copilot engine', async () => {
    const session = await launchTerminal({
      command: 'gh',
      args: ['aw', 'add', 'githubnext/agentics/ci-doctor'],
      cwd: testRepo.path,
      cols: 100,
      rows: 30,
    })
    
    // Wait for welcome message
    await session.waitForText('Welcome to GitHub Agentic Workflows', {
      timeout: 5000
    })
    
    // Wait for engine selection prompt
    await session.waitForText('Which AI engine', { timeout: 5000 })
    
    // Select Copilot (should be first option)
    await session.press('enter')
    
    // Wait for API key prompt (if needed)
    const text = await session.text()
    if (text.includes('Enter API key')) {
      await session.type(process.env.TEST_COPILOT_KEY)
      await session.press('enter')
    }
    
    // Wait for confirmation prompt
    await session.waitForText('Create pull request', { timeout: 10000 })
    await session.press('enter')
    
    // Verify success message
    await session.waitForText('Successfully added workflow', {
      timeout: 20000
    })
    
    // Capture final state
    const finalOutput = await session.text()
    expect(finalOutput).toContain('ci-doctor.md')
    
    session.close()
  }, 60000) // 60 second timeout for full flow
  
  test('handles missing authentication gracefully', async () => {
    const session = await launchTerminal({
      command: 'gh',
      args: ['aw', 'add', 'githubnext/agentics/ci-doctor'],
      cwd: testRepo.path,
      env: { ...process.env, GH_TOKEN: '' }, // Clear auth
    })
    
    await session.waitForText('not logged in', { timeout: 5000 })
    await session.waitForText('gh auth login', { timeout: 2000 })
    
    const errorOutput = await session.text()
    expect(errorOutput).toContain('gh auth login')
    
    session.close()
  })
})
```

## Integration with CI

### GitHub Actions Workflow

```yaml
name: Integration Tests

on:
  push:
    branches: [main]
  pull_request:

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      
      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
      
      - name: Build gh-aw
        run: make build
      
      - name: Install test dependencies
        working-directory: tests/integration
        run: npm install
      
      - name: Run integration tests
        working-directory: tests/integration
        run: npm test
        env:
          TEST_COPILOT_KEY: ${{ secrets.TEST_COPILOT_KEY }}
```

## Risks and Mitigations

### Risk 1: Test Flakiness
**Mitigation**:
- Use generous timeouts
- Implement retry logic for transient failures
- Clear terminal state between tests
- Use stable text patterns for matching

### Risk 2: Maintenance Burden
**Mitigation**:
- Focus on critical paths only
- Share test utilities across tests
- Good documentation for test patterns
- Regular review and refactoring

### Risk 3: CI Resource Usage
**Mitigation**:
- Run integration tests selectively (not on every PR)
- Parallelize where possible
- Cache dependencies
- Set reasonable timeouts

### Risk 4: Environment Dependencies
**Mitigation**:
- Use test fixtures and mock data
- Provide setup/teardown helpers
- Document environment requirements
- Consider using containers for isolation

## Success Criteria

1. **Coverage**: Test at least 3 critical user flows
2. **Reliability**: Tests pass consistently (>95% success rate)
3. **Performance**: Full test suite runs in <5 minutes
4. **Maintainability**: Clear documentation and reusable utilities
5. **CI Integration**: Automated runs on main branch and releases

## Next Steps

1. **Immediate** (This PR):
   - Document findings in this investigation
   - Create basic proof-of-concept test
   - Validate tuistory integration

2. **Short-term** (Next sprint):
   - Implement Phase 1 (setup)
   - Write core test cases
   - Integrate with CI

3. **Long-term** (Future):
   - Expand test coverage
   - Add snapshot testing
   - Document best practices

## References

- [Tuistory GitHub Repository](https://github.com/remorses/tuistory)
- [Charmbracelet Huh Documentation](https://github.com/charmbracelet/huh)
- [gh-aw Interactive Command Implementation](../../pkg/cli/add_interactive.go)
- [Existing Test Patterns](../../pkg/cli/interactive_test.go)

## Conclusion and Current Status

### Investigation Status: ⚠️ Blocked - Package Not Available

**Current Findings (as of January 2026):**

The tuistory npm package (v1.0.0) is published but **empty** - it contains only a package.json file with no actual code. This appears to be an incomplete publication or work-in-progress by the maintainer.

```bash
$ npm install tuistory@1.0.0
# Package installs but contains no code
$ ls node_modules/tuistory/
package.json  # Only file present, no index.js or source code
```

### Updated Recommendation

**Status: Hold on tuistory integration until package is properly published**

While tuistory's design and API (as documented on GitHub) would be an excellent solution for testing the interactive add command, we cannot proceed with integration until the package is properly published to npm.

### Alternative Actions

1. **Short-term (Current Sprint)**
   - Continue with unit testing approach for business logic
   - Document manual testing procedures for interactive flows
   - Monitor tuistory repository for updates

2. **Alternative Testing Tools to Evaluate**
   - **expect-test** (Go native): Consider for basic terminal interaction testing
   - **Manual E2E tests**: Document comprehensive manual test procedures
   - **Refactor for testability**: Extract business logic from interactive UI code

3. **Long-term (If tuistory becomes available)**
   - Re-evaluate integration once package is complete
   - Follow the implementation plan outlined in this document
   - The hybrid testing strategy remains valid

### Benefits of This Investigation

Even though we cannot proceed with tuistory immediately, this investigation has:
- Validated the testing approach and architecture
- Documented integration patterns for future use
- Created reusable test infrastructure (`tests/integration/` directory)
- Established proof-of-concept test structure

### Next Steps

1. **Monitor tuistory**: Watch the GitHub repository for updates
2. **Document workarounds**: Create manual testing documentation
3. **Investigate alternatives**: Research other terminal testing libraries
4. **Consider contribution**: Potentially contribute to tuistory if it aligns with our needs

The hybrid testing strategy (Go unit tests + eventual integration tests) remains the recommended approach once appropriate tooling becomes available.
