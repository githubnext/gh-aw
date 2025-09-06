package workflow

import "strings"

// generateGitPatchStep generates a step that creates and uploads a git patch of changes
func (c *Compiler) generateGitPatchStep(yaml *strings.Builder, data *WorkflowData) {
	c.generateGitPatchStepCore(yaml, data)
	c.generateUploadPatchStep(yaml)
}

// generateGitPatchStepCore generates the main git patch step
func (c *Compiler) generateGitPatchStepCore(yaml *strings.Builder, data *WorkflowData) {
	builder := NewYAMLBuilder()

	// Set up environment variables
	env := map[string]string{
		"GITHUB_AW_SAFE_OUTPUTS": "${{ env.GITHUB_AW_SAFE_OUTPUTS }}",
	}
	if data.SafeOutputs != nil && data.SafeOutputs.PushToBranch != nil {
		env["GITHUB_AW_PUSH_BRANCH"] = "\"" + data.SafeOutputs.PushToBranch.Branch + "\""
	}

	builder.WriteStepHeader("Generate git patch", "always()", env)

	// Generate the shell script
	scriptLines := c.buildGitPatchScript()
	builder.WriteShellScript(scriptLines)

	yaml.WriteString(builder.String())
}

// generateUploadPatchStep generates the upload artifact step
func (c *Compiler) generateUploadPatchStep(yaml *strings.Builder) {
	yaml.WriteString("      - name: Upload git patch\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: aw.patch\n")
	yaml.WriteString("          path: /tmp/aw.patch\n")
	yaml.WriteString("          if-no-files-found: ignore\n")
}

// buildGitPatchScript builds the shell script for git patch generation
func (c *Compiler) buildGitPatchScript() []string {
	script := []string{
		"# Check current git status",
		"echo \"Current git status:\"",
		"git status",
		"",
	}

	// Add branch extraction logic
	script = append(script, c.getBranchExtractionScript()...)
	script = append(script, "")

	// Add initial commit handling
	script = append(script, c.getInitialCommitScript()...)
	script = append(script, "")

	// Add branch-specific patch generation
	script = append(script, c.getBranchPatchScript()...)
	script = append(script, "")

	// Add HEAD-based patch generation
	script = append(script, c.getHeadPatchScript()...)
	script = append(script, "")

	// Add patch display logic
	script = append(script, c.getPatchDisplayScript()...)

	return script
}

// getBranchExtractionScript returns the branch name extraction logic
func (c *Compiler) getBranchExtractionScript() []string {
	return []string{
		"# Extract branch name from JSONL output",
		"BRANCH_NAME=\"\"",
		"if [ -f \"$GITHUB_AW_SAFE_OUTPUTS\" ]; then",
		"  echo \"Checking for branch name in JSONL output...\"",
		"  while IFS= read -r line; do",
		"    if [ -n \"$line\" ]; then",
		"      # Extract branch from create-pull-request line using simple grep and sed",
		"      if echo \"$line\" | grep -q '\"type\"[[:space:]]*:[[:space:]]*\"create-pull-request\"'; then",
		"        echo \"Found create-pull-request line: $line\"",
		"        # Extract branch value using sed",
		"        BRANCH_NAME=$(echo \"$line\" | sed -n 's/.*\"branch\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p')",
		"        if [ -n \"$BRANCH_NAME\" ]; then",
		"          echo \"Extracted branch name from create-pull-request: $BRANCH_NAME\"",
		"          break",
		"        fi",
		"      # Extract branch from push-to-branch line using simple grep and sed",
		"      elif echo \"$line\" | grep -q '\"type\"[[:space:]]*:[[:space:]]*\"push-to-branch\"'; then",
		"        echo \"Found push-to-branch line: $line\"",
		"        # For push-to-branch, we don't extract branch from JSONL since it's configured in the workflow",
		"        # The branch name should come from the environment variable GITHUB_AW_PUSH_BRANCH",
		"        if [ -n \"$GITHUB_AW_PUSH_BRANCH\" ]; then",
		"          BRANCH_NAME=\"$GITHUB_AW_PUSH_BRANCH\"",
		"          echo \"Using configured push-to-branch target: $BRANCH_NAME\"",
		"          break",
		"        fi",
		"      fi",
		"    fi",
		"  done < \"$GITHUB_AW_SAFE_OUTPUTS\"",
		"fi",
	}
}

// getInitialCommitScript returns the initial commit SHA handling
func (c *Compiler) getInitialCommitScript() []string {
	return []string{
		"# Get the initial commit SHA from the base branch of the pull request",
		"if [ \"$GITHUB_EVENT_NAME\" = \"pull_request\" ] || [ \"$GITHUB_EVENT_NAME\" = \"pull_request_review_comment\" ]; then",
		"  INITIAL_SHA=\"$GITHUB_BASE_REF\"",
		"else",
		"  INITIAL_SHA=\"$GITHUB_SHA\"",
		"fi",
		"echo \"Base commit SHA: $INITIAL_SHA\"",
		"# Configure git user for GitHub Actions",
		"git config --global user.email \"action@github.com\"",
		"git config --global user.name \"GitHub Action\"",
	}
}

// getBranchPatchScript returns the branch-based patch generation logic
func (c *Compiler) getBranchPatchScript() []string {
	return []string{
		"# If we have a branch name, check if that branch exists and get its diff",
		"if [ -n \"$BRANCH_NAME\" ]; then",
		"  echo \"Looking for branch: $BRANCH_NAME\"",
		"  # Check if the branch exists",
		"  if git show-ref --verify --quiet refs/heads/$BRANCH_NAME; then",
		"    echo \"Branch $BRANCH_NAME exists, generating patch from branch changes\"",
		"    # Generate patch from the base to the branch",
		"    git format-patch \"$INITIAL_SHA\"..\"$BRANCH_NAME\" --stdout > /tmp/aw.patch || echo \"Failed to generate patch from branch\" > /tmp/aw.patch",
		"    echo \"Patch file created from branch: $BRANCH_NAME\"",
		"  else",
		"    echo \"Branch $BRANCH_NAME does not exist, falling back to current HEAD\"",
		"    BRANCH_NAME=\"\"",
		"  fi",
		"fi",
	}
}

// getHeadPatchScript returns the HEAD-based patch generation logic
func (c *Compiler) getHeadPatchScript() []string {
	return []string{
		"# If no branch or branch doesn't exist, use the existing logic",
		"if [ -z \"$BRANCH_NAME\" ]; then",
		"  echo \"Using current HEAD for patch generation\"",
		"  # Stage any unstaged files",
		"  git add -A || true",
		"  # Check if there are staged files to commit",
		"  if ! git diff --cached --quiet; then",
		"    echo \"Staged files found, committing them...\"",
		"    git commit -m \"[agent] staged files\" || true",
		"    echo \"Staged files committed\"",
		"  else",
		"    echo \"No staged files to commit\"",
		"  fi",
		"  # Check updated git status",
		"  echo \"Updated git status after committing staged files:\"",
		"  git status",
		"  # Show compact diff information between initial commit and HEAD (committed changes only)",
		"  echo '## Git diff' >> $GITHUB_STEP_SUMMARY",
		"  echo '' >> $GITHUB_STEP_SUMMARY",
		"  echo '```' >> $GITHUB_STEP_SUMMARY",
		"  git diff --name-only \"$INITIAL_SHA\"..HEAD >> $GITHUB_STEP_SUMMARY || true",
		"  echo '```' >> $GITHUB_STEP_SUMMARY",
		"  echo '' >> $GITHUB_STEP_SUMMARY",
		"  # Check if there are any committed changes since the initial commit",
		"  if git diff --quiet \"$INITIAL_SHA\" HEAD; then",
		"    echo \"No committed changes detected since initial commit\"",
		"    echo \"Skipping patch generation - no committed changes to create patch from\"",
		"  else",
		"    echo \"Committed changes detected, generating patch...\"",
		"    # Generate patch from initial commit to HEAD (committed changes only)",
		"    git format-patch \"$INITIAL_SHA\"..HEAD --stdout > /tmp/aw.patch || echo \"Failed to generate patch\" > /tmp/aw.patch",
		"    echo \"Patch file created at /tmp/aw.patch\"",
		"  fi",
		"fi",
	}
}

// getPatchDisplayScript returns the patch display logic
func (c *Compiler) getPatchDisplayScript() []string {
	return []string{
		"# Show patch info if it exists",
		"if [ -f /tmp/aw.patch ]; then",
		"  ls -la /tmp/aw.patch",
		"  # Show the first 50 lines of the patch for review",
		"  echo '## Git Patch' >> $GITHUB_STEP_SUMMARY",
		"  echo '' >> $GITHUB_STEP_SUMMARY",
		"  echo '```diff' >> $GITHUB_STEP_SUMMARY",
		"  head -50 /tmp/aw.patch >> $GITHUB_STEP_SUMMARY || echo \"Could not display patch contents\" >> $GITHUB_STEP_SUMMARY",
		"  echo '...' >> $GITHUB_STEP_SUMMARY",
		"  echo '```' >> $GITHUB_STEP_SUMMARY",
		"  echo '' >> $GITHUB_STEP_SUMMARY",
		"fi",
	}
}
