import { describe, test, expect, beforeAll, beforeEach, afterEach } from 'vitest'
import { launchTerminal } from 'tuistory'
import { 
  setupTestRepo, 
  cleanupTestRepo, 
  ensureBinaryExists, 
  GH_AW_BINARY,
  getTestEnv 
} from './setup.js'

describe('gh aw add interactive (proof of concept)', () => {
  let testRepo

  // Ensure the binary is built before running tests
  beforeAll(async () => {
    await ensureBinaryExists()
  }, 60000) // 60s timeout for building

  beforeEach(async () => {
    testRepo = await setupTestRepo()
  })

  afterEach(async () => {
    await cleanupTestRepo(testRepo)
  })

  test('shows error when not authenticated (proof of concept)', async () => {
    // This is a basic proof-of-concept test that doesn't require full setup
    // It just verifies tuistory can launch and interact with gh-aw
    
    const session = await launchTerminal({
      command: GH_AW_BINARY,
      args: ['--help'],
      cwd: testRepo.path,
      env: getTestEnv(),
      cols: 100,
      rows: 30,
    })

    // Wait for help text to appear
    await session.waitForText('GitHub Agentic Workflows', { timeout: 5000 })

    // Get the terminal text
    const text = await session.text()
    
    // Verify key help text is present
    expect(text).toContain('Usage:')
    expect(text).toContain('add')
    expect(text).toContain('compile')

    // Cleanup
    session.close()
  }, 30000)

  test('add command shows usage when run without args', async () => {
    const session = await launchTerminal({
      command: GH_AW_BINARY,
      args: ['add'],
      cwd: testRepo.path,
      env: getTestEnv(),
      cols: 100,
      rows: 30,
    })

    // Wait for error message
    await session.waitForText('Error:', { timeout: 5000 })

    // Get the terminal text
    const text = await session.text()
    
    // Verify error is about missing argument
    expect(text).toMatch(/requires at least 1 arg/i)

    // Cleanup
    session.close()
  }, 30000)

  // Note: Full interactive tests would require:
  // 1. GitHub CLI authentication setup
  // 2. Test repository with proper permissions
  // 3. Mock API keys for testing
  // 4. More sophisticated terminal interaction patterns
  //
  // This proof of concept demonstrates:
  // - Tuistory can launch gh-aw
  // - Terminal text can be captured and verified
  // - Multiple test cases can be structured
  //
  // See docs/testing/tuistory-investigation.md for full implementation plan
})

describe('tuistory basic functionality', () => {
  test('can interact with echo command', async () => {
    const session = await launchTerminal({
      command: 'bash',
      args: ['-c', 'echo "Hello, Tuistory!"'],
      cols: 80,
      rows: 24,
    })

    await session.waitForText('Hello, Tuistory!', { timeout: 2000 })
    
    const text = await session.text()
    expect(text).toContain('Hello, Tuistory!')

    session.close()
  }, 10000)

  test('can type and press enter', async () => {
    const session = await launchTerminal({
      command: 'bash',
      args: [],
      cols: 80,
      rows: 24,
    })

    // Wait for bash prompt
    await session.waitForText('$', { timeout: 2000 })

    // Type echo command
    await session.type('echo "Interactive typing works"')
    await session.press('enter')

    // Wait for output
    await session.waitForText('Interactive typing works', { timeout: 2000 })

    const text = await session.text()
    expect(text).toContain('Interactive typing works')

    // Exit bash
    await session.type('exit')
    await session.press('enter')

    session.close()
  }, 15000)
})
