package workflow

import (
"github.com/githubnext/gh-aw/pkg/logger"
)

// This file contains the shared logger for all safe_outputs_* files.
// The safe_outputs functionality has been split into multiple focused files:
// - safe_outputs_config.go: Configuration parsing and validation
// - safe_outputs_steps.go: Step builders for GitHub Script and custom actions
// - safe_outputs_env.go: Environment variable helpers
// - safe_outputs_jobs.go: Job assembly and orchestration

var safeOutputsLog = logger.New("workflow:safe_outputs")
