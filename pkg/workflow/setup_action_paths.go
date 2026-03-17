package workflow

import "github.com/github/gh-aw/pkg/constants"

// SetupActionDestination is the path where the setup action copies script files
// on the agent runner (e.g. ${{ runner.temp }}/gh-aw/actions).
const SetupActionDestination = constants.GhAwRootDir + "/actions"

// SafeOutputsDir is the directory for safe-outputs files on the runner
const SafeOutputsDir = constants.GhAwRootDir + "/safeoutputs"

// GhAwMCPScriptsDir is the directory for MCP scripts files on the runner
const GhAwMCPScriptsDir = constants.GhAwRootDir + "/mcp-scripts"

// GhAwBinaryPath is the path to the gh-aw binary on the runner
const GhAwBinaryPath = constants.GhAwRootDir + "/gh-aw"

// SafeJobsDownloadDir is the directory for safe job files on the runner
const SafeJobsDownloadDir = constants.GhAwRootDir + "/safe-jobs/"
