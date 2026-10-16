package cli

import (
	"path/filepath"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var compileWorkflowFileContextLog = logger.New("cli:compile_workflow_file_context")

func setWorkflowFileContext(compiler *workflow.Compiler, filePath string) {
	relPath, err := getRepositoryRelativePath(filePath)
	if err != nil {
		compileWorkflowFileContextLog.Printf("Warning: failed to get repository-relative path for %s: %v", filePath, err)
		relPath = filepath.Base(filePath)
	}
	compiler.SetWorkflowIdentifier(relPath)

	fileRepoSlug := getRepositorySlugFromRemoteForPath(filePath)
	if fileRepoSlug == "" {
		return
	}
	if compiler.IsRepositorySlugLocked() {
		compileWorkflowFileContextLog.Printf("Repository slug from file remote (%s) ignored: overridden via --schedule-seed (%s)", fileRepoSlug, compiler.GetRepositorySlug())
		return
	}
	compiler.SetRepositorySlugIfUnlocked(fileRepoSlug)
	compileWorkflowFileContextLog.Printf("Repository slug for file set: %s", fileRepoSlug)
}
