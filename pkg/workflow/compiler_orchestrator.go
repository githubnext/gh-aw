package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

// Shared logger used across compiler orchestrator modules
var detectionLog = logger.New("workflow:detection")
