package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/workflow"
)

func mergeProjectFileWithTracking(resolved *ResolvedWorkflow, tracker *FileTracker, gitRoot string) error {
	destFile := filepath.Join(gitRoot, workflow.RepoConfigFileName)
	existing := []byte("{}")
	fileExists := fileutil.FileExists(destFile)
	if fileExists {
		var err error
		existing, err = os.ReadFile(filepath.Clean(destFile))
		if err != nil {
			return fmt.Errorf("failed to read project file %q: %w", workflow.RepoConfigFileName, err)
		}
	}

	merged, err := mergeProjectJSON(existing, resolved.Content)
	if err != nil {
		return fmt.Errorf("failed to merge package project file %q: %w", resolved.Spec.WorkflowPath, err)
	}
	if err := validateMergedProjectJSON(merged); err != nil {
		return fmt.Errorf("added package project settings are invalid: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(destFile), constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create project file directory: %w", err)
	}

	if tracker != nil {
		if fileExists {
			tracker.TrackModified(destFile)
		} else {
			tracker.TrackCreated(destFile)
		}
	}
	if err := os.WriteFile(destFile, merged, constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write project file %q: %w", workflow.RepoConfigFileName, err)
	}
	return nil
}

func validateMergedProjectJSON(merged []byte) error {
	tempRoot, err := os.MkdirTemp("", "gh-aw-project-config-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary project config directory: %w", err)
	}
	defer os.RemoveAll(tempRoot)
	configPath := filepath.Join(tempRoot, workflow.RepoConfigFileName)
	if err := os.MkdirAll(filepath.Dir(configPath), constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create temporary project config directory: %w", err)
	}
	if err := os.WriteFile(configPath, merged, constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write temporary project config: %w", err)
	}
	_, err = workflow.LoadRepoConfig(tempRoot)
	return err
}

func mergeProjectJSON(existing, added []byte) ([]byte, error) {
	var existingSettings map[string]any
	if err := json.Unmarshal(existing, &existingSettings); err != nil {
		return nil, fmt.Errorf("target project file is not valid JSON: %w", err)
	}
	var addedSettings map[string]any
	if err := json.Unmarshal(added, &addedSettings); err != nil {
		return nil, fmt.Errorf("added project file is not valid JSON: %w", err)
	}
	if existingSettings == nil || addedSettings == nil {
		return nil, errors.New("project files must contain JSON objects")
	}
	mergeProjectSettings(existingSettings, addedSettings)
	merged, err := json.MarshalIndent(existingSettings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode merged project file: %w", err)
	}
	return append(merged, '\n'), nil
}

func mergeProjectSettings(target, added map[string]any) {
	for key, addedValue := range added {
		addedObject, addedIsObject := addedValue.(map[string]any)
		targetObject, targetIsObject := target[key].(map[string]any)
		if addedIsObject && targetIsObject {
			mergeProjectSettings(targetObject, addedObject)
			continue
		}
		target[key] = addedValue
	}
}
