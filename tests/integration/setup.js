import { fileURLToPath } from 'url'
import { dirname, join } from 'path'
import { mkdtemp, rm, mkdir, writeFile } from 'fs/promises'
import { tmpdir } from 'os'
import { execSync } from 'child_process'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

// Get the root directory of the gh-aw project
export const PROJECT_ROOT = join(__dirname, '..', '..')

// Path to the gh-aw binary
export const GH_AW_BINARY = join(PROJECT_ROOT, 'gh-aw')

/**
 * Setup a temporary test repository
 * @returns {Promise<{path: string, cleanup: () => Promise<void>}>}
 */
export async function setupTestRepo() {
  const tempDir = await mkdtemp(join(tmpdir(), 'gh-aw-test-'))
  
  try {
    // Initialize git repository
    execSync('git init', { cwd: tempDir, stdio: 'ignore' })
    execSync('git config user.email "test@example.com"', { cwd: tempDir, stdio: 'ignore' })
    execSync('git config user.name "Test User"', { cwd: tempDir, stdio: 'ignore' })
    
    // Create initial commit
    await writeFile(join(tempDir, 'README.md'), '# Test Repository\n')
    execSync('git add .', { cwd: tempDir, stdio: 'ignore' })
    execSync('git commit -m "Initial commit"', { cwd: tempDir, stdio: 'ignore' })
    
    // Create .github/workflows directory
    await mkdir(join(tempDir, '.github', 'workflows'), { recursive: true })
    
    return {
      path: tempDir,
      cleanup: async () => {
        await rm(tempDir, { recursive: true, force: true })
      }
    }
  } catch (error) {
    // Cleanup on error
    await rm(tempDir, { recursive: true, force: true })
    throw error
  }
}

/**
 * Cleanup a test repository
 * @param {{cleanup: () => Promise<void>}} testRepo 
 */
export async function cleanupTestRepo(testRepo) {
  if (testRepo && testRepo.cleanup) {
    await testRepo.cleanup()
  }
}

/**
 * Build the gh-aw binary if it doesn't exist
 */
export async function ensureBinaryExists() {
  try {
    execSync(`test -f ${GH_AW_BINARY}`, { stdio: 'ignore' })
  } catch {
    console.log('Building gh-aw binary...')
    execSync('make build', { cwd: PROJECT_ROOT, stdio: 'inherit' })
  }
}

/**
 * Wait for a condition to be true with timeout
 * @param {() => boolean | Promise<boolean>} condition 
 * @param {number} timeout 
 * @param {number} interval 
 */
export async function waitFor(condition, timeout = 5000, interval = 100) {
  const startTime = Date.now()
  
  while (Date.now() - startTime < timeout) {
    if (await condition()) {
      return true
    }
    await new Promise(resolve => setTimeout(resolve, interval))
  }
  
  throw new Error(`Timeout waiting for condition after ${timeout}ms`)
}

/**
 * Get test environment variables
 */
export function getTestEnv() {
  return {
    ...process.env,
    // Disable colors for easier text matching
    NO_COLOR: '1',
    // Indicate we're in test mode
    GO_TEST_MODE: 'false', // Set to false to allow interactive mode
    // Clear CI flag to allow interactive mode
    CI: '',
  }
}
