package workflow

import _ "embed"

// Prompt file paths at runtime (copied by setup action to /opt/gh-aw/prompts)
const (
	promptsDir                     = "/opt/gh-aw/prompts"
	prContextPromptFile            = "pr_context_prompt.md"
	tempFolderPromptFile           = "temp_folder_prompt.md"
	playwrightPromptFile           = "playwright_prompt.md"
	markdownPromptFile             = "markdown.md"
	xpiaPromptFile                 = "xpia.md"
	cacheMemoryPromptFile          = "cache_memory_prompt.md"
	cacheMemoryPromptMultiFile     = "cache_memory_prompt_multi.md"
	repoMemoryPromptFile           = "repo_memory_prompt.md"
	repoMemoryPromptMultiFile      = "repo_memory_prompt_multi.md"
	safeOutputsPromptFile          = "safe_outputs_prompt.md"
	safeOutputsCreatePRFile        = "safe_outputs_create_pull_request.md"
	safeOutputsPushToBranchFile    = "safe_outputs_push_to_pr_branch.md"
	safeOutputsAutoCreateIssueFile = "safe_outputs_auto_create_issue.md"
)

// GitHub context prompt is kept embedded because it contains GitHub Actions expressions
// that need to be extracted at compile time. Moving this to a runtime file would require
// reading and parsing the file during compilation, which is more complex.
//
//go:embed prompts/github_context_prompt.md
var githubContextPromptText string
