package workflow

import (
	"fmt"
	"strings"
)

// YAMLBuilder provides utilities for building YAML content with proper indentation
type YAMLBuilder struct {
	builder strings.Builder
}

// NewYAMLBuilder creates a new YAML builder
func NewYAMLBuilder() *YAMLBuilder {
	return &YAMLBuilder{}
}

// String returns the built YAML content
func (y *YAMLBuilder) String() string {
	return y.builder.String()
}

// WriteString writes a string to the YAML builder
func (y *YAMLBuilder) WriteString(s string) {
	y.builder.WriteString(s)
}

// WriteComment writes a comment line
func (y *YAMLBuilder) WriteComment(comment string) {
	y.builder.WriteString(fmt.Sprintf("# %s\n", comment))
}

// WriteKeyValue writes a key-value pair with proper formatting
func (y *YAMLBuilder) WriteKeyValue(key, value string) {
	y.builder.WriteString(fmt.Sprintf("%s: %s\n", key, value))
}

// WriteKeyValueIndented writes an indented key-value pair
func (y *YAMLBuilder) WriteKeyValueIndented(indent, key, value string) {
	y.builder.WriteString(fmt.Sprintf("%s%s: %s\n", indent, key, value))
}

// WriteSection writes a section header
func (y *YAMLBuilder) WriteSection(title string) {
	y.builder.WriteString(fmt.Sprintf("## %s\n", title))
}

// WriteInstructionBlock writes an instruction block with proper markdown formatting
func (y *YAMLBuilder) WriteInstructionBlock(title, content string) {
	y.builder.WriteString(fmt.Sprintf("          **%s**\n", title))
	y.builder.WriteString("          \n")

	// Write content lines with proper indentation
	for _, line := range strings.Split(content, "\n") {
		y.builder.WriteString(fmt.Sprintf("          %s\n", line))
	}
	y.builder.WriteString("          \n")
}

// WriteJSONExample writes a JSON example with proper formatting
func (y *YAMLBuilder) WriteJSONExample(jsonContent string) {
	y.builder.WriteString("          ```json\n")
	y.builder.WriteString(fmt.Sprintf("          %s\n", jsonContent))
	y.builder.WriteString("          ```\n")
}

// WriteMarkdownContent writes markdown content with proper indentation for YAML
func (y *YAMLBuilder) WriteMarkdownContent(content string) {
	for _, line := range strings.Split(content, "\n") {
		y.builder.WriteString("          " + line + "\n")
	}
}

// SafeOutputType represents different types of safe outputs
type SafeOutputType struct {
	Type        string
	Title       string
	Description string
	JSONExample string
}

// WriteShellScript writes a shell script block with proper indentation
func (y *YAMLBuilder) WriteShellScript(lines []string) {
	for _, line := range lines {
		y.builder.WriteString(fmt.Sprintf("          %s\n", line))
	}
}

// WriteStepHeader writes a GitHub Actions step header
func (y *YAMLBuilder) WriteStepHeader(name, ifCondition string, env map[string]string) {
	y.builder.WriteString(fmt.Sprintf("      - name: %s\n", name))
	if ifCondition != "" {
		y.builder.WriteString(fmt.Sprintf("        if: %s\n", ifCondition))
	}
	if len(env) > 0 {
		y.builder.WriteString("        env:\n")
		for key, value := range env {
			y.builder.WriteString(fmt.Sprintf("          %s: %s\n", key, value))
		}
	}
	y.builder.WriteString("        run: |\n")
}

// GetSafeOutputTypes returns predefined safe output types
func GetSafeOutputTypes() map[string]SafeOutputType {
	return map[string]SafeOutputType{
		"add-issue-comment": {
			Type:        "add-issue-comment",
			Title:       "Adding a Comment to an Issue or Pull Request",
			Description: "To add a comment to an issue or pull request:\n1. Write an entry to \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\":\n{example}\n2. After you write to that file, read it as JSONL and check it is valid. If it isn't, make any necessary corrections to it to fix it up",
			JSONExample: `{"type": "add-issue-comment", "body": "Your comment content in markdown"}`,
		},
		"create-issue": {
			Type:        "create-issue",
			Title:       "Creating an Issue",
			Description: "To create an issue:\n1. Write an entry to \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\":\n{example}\n2. After you write to that file, read it as JSONL and check it is valid. If it isn't, make any necessary corrections to it to fix it up",
			JSONExample: `{"type": "create-issue", "title": "Issue title", "body": "Issue body in markdown", "labels": ["optional", "labels"]}`,
		},
		"create-pull-request": {
			Type:        "create-pull-request",
			Title:       "Creating a Pull Request",
			Description: "To create a pull request:\n1. Make any file changes directly in the working directory\n2. If you haven't done so already, create a local branch using an appropriate unique name\n3. Add and commit your changes to the branch. Be careful to add exactly the files you intend, and check there are no extra files left un-added. Check you haven't deleted or changed any files you didn't intend to.\n4. Do not push your changes. That will be done later. Instead append the PR specification to the file \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\":\n{example}\n5. After you write to that file, read it as JSONL and check it is valid. If it isn't, make any necessary corrections to it to fix it up",
			JSONExample: `{"type": "create-pull-request", "branch": "branch-name", "title": "PR title", "body": "PR body in markdown", "labels": ["optional", "labels"]}`,
		},
		"add-issue-label": {
			Type:        "add-issue-label",
			Title:       "Adding Labels to Issues or Pull Requests",
			Description: "To add labels to a pull request:\n1. Write an entry to \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\":\n{example}\n2. After you write to that file, read it as JSONL and check it is valid. If it isn't, make any necessary corrections to it to fix it up",
			JSONExample: `{"type": "add-issue-label", "labels": ["label1", "label2", "label3"]}`,
		},
		"update-issue": {
			Type:        "update-issue",
			Title:       "Updating an Issue",
			Description: "To udpate an issue:\n{example}",
			JSONExample: `{"type": "update-issue", "title": "New issue title", "body": "Updated issue body in markdown", "status": "open"}`,
		},
		"push-to-branch": {
			Type:        "push-to-branch",
			Title:       "Pushing Changes to Branch",
			Description: "To push changes to a branch, for example to add code to a pull request:\n1. Make any file changes directly in the working directory\n2. Add and commit your changes to the branch. Be careful to add exactly the files you intend, and check there are no extra files left un-added. Check you haven't deleted or changed any files you didn't intend to.\n3. Indicate your intention to push to the branch by writing to the file \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\":\n{example}\n4. After you write to that file, read it as JSONL and check it is valid. If it isn't, make any necessary corrections to it to fix it up",
			JSONExample: `{"type": "push-to-branch", "message": "Commit message describing the changes"}`,
		},
		"missing-tool": {
			Type:        "missing-tool",
			Title:       "Reporting Missing Tools or Functionality",
			Description: "If you need to use a tool or functionality that is not available to complete your task:\n1. Write an entry to \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\":\n{example}\n2. The `tool` field should specify the name or type of missing functionality\n3. The `reason` field should explain why this tool/functionality is required to complete the task\n4. The `alternatives` field is optional but can suggest workarounds or alternative approaches\n5. After you write to that file, read it as JSONL and check it is valid. If it isn't, make any necessary corrections to it to fix it up",
			JSONExample: `{"type": "missing-tool", "tool": "tool-name", "reason": "Why this tool is needed", "alternatives": "Suggested alternatives or workarounds"}`,
		},
	}
}
