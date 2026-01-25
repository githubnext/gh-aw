package workflow

import (
"github.com/githubnext/gh-aw/pkg/logger"
)

// Shared loggers used across compiler orchestrator modules
var detectionLog = logger.New("workflow:detection")
var orchestratorLog = logger.New("workflow:compiler_orchestrator")
