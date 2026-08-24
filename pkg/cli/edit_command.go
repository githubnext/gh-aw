package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

// NewEditCommand creates the experimental command for changing workflow frontmatter.
func NewEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <workflow> [path: value]",
		Short: "Experimental: edit workflow frontmatter and recompile",
		Long: `Experimental: edit schema-validated workflow frontmatter and recompile its generated file.

The workflow-id may be a workflow name, a Markdown filename, or a path. Changes are
validated before writing. Workflows managed by a source: declaration cannot be edited.`,
		Example: `  gh aw edit repo-assist "max-turns: 20"
  gh aw edit repo-assist --schedule "every 6h"
  gh aw edit repo-assist --set model=small --unset engine.model
  gh aw edit repo-assist --add-import shared/common.md
  gh aw edit repo-assist --add-skill shared/review`,
		Args: cobra.RangeArgs(1, 2),
		RunE: runEditCommand,
	}
	cmd.Flags().StringArray("set", nil, "Set a frontmatter path (path=value)")
	cmd.Flags().StringArray("unset", nil, "Remove a frontmatter path")
	cmd.Flags().StringArray("add", nil, "Append a value to a list (path=value)")
	cmd.Flags().StringArray("remove", nil, "Remove a value from a list (path=value)")
	cmd.Flags().StringArray("add-import", nil, "Append a workflow import path")
	cmd.Flags().StringArray("remove-import", nil, "Remove a workflow import path")
	cmd.Flags().StringArray("add-skill", nil, "Append a workflow skill")
	cmd.Flags().StringArray("remove-skill", nil, "Remove a workflow skill")
	cmd.Flags().String("schedule", "", "Set a schedule using a fuzzy schedule or cron expression; use off to remove it")
	cmd.Flags().Bool("dry-run", false, "Validate changes without writing or compiling")
	cmd.Flags().StringP("dir", "d", "", "Workflow directory (default: $GH_AW_WORKFLOWS_DIR or .github/workflows)")
	cmd.ValidArgsFunction = CompleteWorkflowNames
	RegisterDirFlagCompletion(cmd, "dir")
	return cmd
}

func runEditCommand(cmd *cobra.Command, args []string) error {
	workflowPath, err := resolveWorkflowFileInDir(args[0], false, editFlagString(cmd, "dir"))
	if err != nil {
		return err
	}
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("read workflow: %w", err)
	}
	parsed, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return err
	}
	if _, managed := parsed.Frontmatter["source"]; managed {
		return errors.New("cannot edit a source-managed workflow; update its source or pin, then run gh aw update")
	}

	changes, err := editChangesFromCommand(cmd, args)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		return errors.New("provide an assignment or an edit flag")
	}
	for _, change := range changes {
		if err := applyEditChange(parsed.Frontmatter, change); err != nil {
			return err
		}
	}
	if err := parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(parsed.Frontmatter, workflowPath); err != nil {
		return fmt.Errorf("invalid edited workflow: %w", err)
	}
	updated, err := replaceFrontmatter(string(content), parsed.Frontmatter)
	if err != nil {
		return err
	}
	if editFlagBool(cmd, "dry-run") {
		fmt.Fprint(cmd.OutOrStdout(), updated)
		return nil
	}

	return writeAndCompileEditedWorkflow(workflowPath, content, updated)
}

func writeAndCompileEditedWorkflow(workflowPath string, content []byte, updated string) error {
	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	previousLock, lockErr := os.ReadFile(lockPath)
	lockExisted := lockErr == nil
	if lockErr != nil && !os.IsNotExist(lockErr) {
		return fmt.Errorf("read generated lock file: %w", lockErr)
	}
	if err := os.WriteFile(workflowPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write workflow: %w", err)
	}
	if err := compileWorkflow(context.Background(), workflowPath, false, true, ""); err != nil {
		_ = os.WriteFile(workflowPath, content, 0o644)
		if lockExisted {
			_ = os.WriteFile(lockPath, previousLock, 0o644)
		} else {
			_ = os.Remove(lockPath)
		}
		return fmt.Errorf("compile edited workflow: %w", err)
	}
	return nil
}

type editChange struct {
	kind, path string
	value      any
}

func editChangesFromCommand(cmd *cobra.Command, args []string) ([]editChange, error) {
	var changes []editChange
	if len(args) == 2 {
		change, err := parseEditAssignment(args[1], ":")
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	for _, name := range []string{"set", "add", "remove"} {
		for _, assignment := range editFlagStrings(cmd, name) {
			change, err := parseEditAssignment(assignment, "=")
			if err != nil {
				return nil, err
			}
			change.kind = name
			changes = append(changes, change)
		}
	}
	for _, path := range editFlagStrings(cmd, "unset") {
		changes = append(changes, editChange{kind: "unset", path: path})
	}
	for _, importPath := range editFlagStrings(cmd, "add-import") {
		changes = append(changes, editChange{kind: "add", path: "imports", value: importPath})
	}
	for _, importPath := range editFlagStrings(cmd, "remove-import") {
		changes = append(changes, editChange{kind: "remove", path: "imports", value: importPath})
	}
	for _, skill := range editFlagStrings(cmd, "add-skill") {
		changes = append(changes, editChange{kind: "add", path: "skills", value: skill})
	}
	for _, skill := range editFlagStrings(cmd, "remove-skill") {
		changes = append(changes, editChange{kind: "remove", path: "skills", value: skill})
	}
	if schedule := editFlagString(cmd, "schedule"); schedule != "" {
		change, err := scheduleChange(schedule)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func scheduleChange(schedule string) (editChange, error) {
	schedule = strings.TrimSpace(schedule)
	if strings.EqualFold(schedule, "off") {
		return editChange{kind: "unset", path: "on.schedule"}, nil
	}
	_, _, err := parser.ParseSchedule(schedule)
	if err != nil {
		return editChange{}, fmt.Errorf("invalid schedule: %w", err)
	}
	return editChange{kind: "set", path: "on.schedule", value: schedule}, nil
}

func parseEditAssignment(assignment, separator string) (editChange, error) {
	path, rawValue, ok := strings.Cut(assignment, separator)
	path, rawValue = strings.TrimSpace(path), strings.TrimSpace(rawValue)
	if !ok || path == "" || rawValue == "" {
		return editChange{}, fmt.Errorf("invalid assignment %q; expected path%svalue", assignment, separator)
	}
	var value map[string]any
	if err := yaml.Unmarshal([]byte("value: "+rawValue), &value); err != nil {
		return editChange{}, fmt.Errorf("parse value for %q: %w", path, err)
	}
	return editChange{kind: "set", path: path, value: value["value"]}, nil
}

func applyEditChange(frontmatter map[string]any, change editChange) error {
	change, err := normalizeEditChange(frontmatter, change)
	if err != nil {
		return err
	}
	parent, key, err := editChangeParent(frontmatter, change.path)
	if err != nil {
		return err
	}
	return applyEditChangeToParent(parent, key, change)
}

func normalizeEditChange(frontmatter map[string]any, change editChange) (editChange, error) {
	if change.kind == "set" && change.path == "on.schedule" {
		if schedule, ok := change.value.(string); ok {
			var err error
			change, err = scheduleChange(schedule)
			if err != nil {
				return editChange{}, err
			}
		}
	}
	if change.path == "imports" && (change.kind == "add" || change.kind == "remove") {
		if _, objectImports := frontmatter["imports"].(map[string]any); objectImports {
			change.path = "imports.aw"
		}
	}
	return change, nil
}

func editChangeParent(frontmatter map[string]any, changePath string) (map[string]any, string, error) {
	path := strings.Split(changePath, ".")
	if slices.Contains(path, "") {
		return nil, "", fmt.Errorf("invalid frontmatter path %q", changePath)
	}
	parent := frontmatter
	for _, part := range path[:len(path)-1] {
		child, ok := parent[part].(map[string]any)
		if !ok {
			if part == "on" {
				switch triggers := parent[part].(type) {
				case string:
					child = map[string]any{triggers: nil}
				case []any:
					child = make(map[string]any, len(triggers))
					for _, trigger := range triggers {
						name, ok := trigger.(string)
						if !ok {
							return nil, "", fmt.Errorf("cannot edit %q because on contains a non-string trigger", changePath)
						}
						child[name] = nil
					}
				}
				if child != nil {
					parent[part] = child
					parent = child
					continue
				}
			}
			if parent[part] != nil {
				return nil, "", fmt.Errorf("cannot edit %q because %q is not an object", changePath, part)
			}
			child = map[string]any{}
			parent[part] = child
		}
		parent = child
	}
	return parent, path[len(path)-1], nil
}

func applyEditChangeToParent(parent map[string]any, key string, change editChange) error {
	switch change.kind {
	case "set":
		parent[key] = change.value
	case "unset":
		delete(parent, key)
	case "add":
		values, ok := parent[key].([]any)
		if !ok && parent[key] != nil {
			return fmt.Errorf("cannot add to %q because it is not a list", change.path)
		}
		if !slices.ContainsFunc(values, func(value any) bool { return fmt.Sprint(value) == fmt.Sprint(change.value) }) {
			parent[key] = append(values, change.value)
		}
	case "remove":
		values, ok := parent[key].([]any)
		if !ok {
			return fmt.Errorf("cannot remove from %q because it is not a list", change.path)
		}
		parent[key] = slices.DeleteFunc(values, func(value any) bool { return fmt.Sprint(value) == fmt.Sprint(change.value) })
	default:
		return fmt.Errorf("unsupported edit operation %q", change.kind)
	}
	return nil
}

func replaceFrontmatter(content string, frontmatter map[string]any) (string, error) {
	encoded, err := yaml.MarshalWithOptions(frontmatter, yaml.Indent(2), yaml.IndentSequence(true))
	if err != nil {
		return "", fmt.Errorf("encode frontmatter: %w", err)
	}
	firstLineEnd := strings.IndexByte(content, '\n')
	if firstLineEnd < 0 || strings.TrimSpace(content[:firstLineEnd]) != "---" {
		return "", errors.New("workflow must begin with YAML frontmatter")
	}
	bodyStart := firstLineEnd + 1
	for lineStart := bodyStart; lineStart <= len(content); {
		lineEnd := strings.IndexByte(content[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(content)
		} else {
			lineEnd += lineStart
		}
		if strings.TrimSpace(content[lineStart:lineEnd]) == "---" {
			if lineEnd < len(content) {
				lineEnd++
			}
			return "---\n" + string(encoded) + "---\n" + content[lineEnd:], nil
		}
		if lineEnd == len(content) {
			break
		}
		lineStart = lineEnd + 1
	}
	return "", errors.New("frontmatter not properly closed")
}

func editFlagString(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return value
}

func editFlagStrings(cmd *cobra.Command, name string) []string {
	value, _ := cmd.Flags().GetStringArray(name)
	return value
}

func editFlagBool(cmd *cobra.Command, name string) bool {
	value, _ := cmd.Flags().GetBool(name)
	return value
}
