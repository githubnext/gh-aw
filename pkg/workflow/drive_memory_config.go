package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
)

const (
	defaultDriveMemoryDir       = "/tmp/gh-aw/drive-memory"
	driveMemoryDirPrefix        = "/tmp/gh-aw/drive-memory-"
	defaultDriveMemoryMountPath = ".gh-aw-drive-memory"
	driveMemoryMountPathPrefix  = ".gh-aw-drive-memory-"
)

// DriveMemoryConfig holds configuration for drive-memory functionality.
type DriveMemoryConfig struct {
	Drives []DriveMemoryEntry `yaml:"drives,omitempty"`
}

// DriveMemoryEntry represents one persistent GitHub Drive.
type DriveMemoryEntry struct {
	ID                string                  `yaml:"id"`
	DriveName         string                  `yaml:"drive-name,omitempty"`
	Description       string                  `yaml:"description,omitempty"`
	DiskSize          string                  `yaml:"disk-size,omitempty"`
	Prefetch          bool                    `yaml:"prefetch,omitempty"`
	RestoreOnly       bool                    `yaml:"restore-only,omitempty"`
	AllowedExtensions []string                `yaml:"allowed-extensions,omitempty"`
	Validation        *MemoryValidationConfig `yaml:"validation,omitempty"`
}

func driveMemoryDirFor(id string) string {
	if id == "" || id == "default" {
		return defaultDriveMemoryDir
	}
	if !isValidCacheID(id) {
		return driveMemoryDirPrefix + memoryValidationStepID("invalid", id)
	}
	return driveMemoryDirPrefix + id
}

func driveMemoryMountPathFor(id string) string {
	if id == "" || id == "default" {
		return defaultDriveMemoryMountPath
	}
	if !isValidCacheID(id) {
		return driveMemoryMountPathPrefix + memoryValidationStepID("invalid", id)
	}
	return driveMemoryMountPathPrefix + id
}

func driveMemoryValidationStepID(id string) string {
	return memoryValidationStepID("validate_drive_memory", id)
}

func driveHasValidationStep(drive DriveMemoryEntry) bool {
	return len(drive.AllowedExtensions) > 0 || drive.Validation != nil
}

func defaultDriveMemoryEntries() []DriveMemoryEntry {
	return []DriveMemoryEntry{{
		ID:                "default",
		DriveName:         "default",
		AllowedExtensions: constants.DefaultAllowedMemoryExtensions,
	}}
}

func parseDriveMemoryEntry(raw map[string]any, defaultID string) (DriveMemoryEntry, error) {
	entry := DriveMemoryEntry{
		ID:        defaultID,
		DriveName: defaultID,
	}
	if id, ok := raw["id"].(string); ok {
		if !isValidCacheID(id) {
			return entry, fmt.Errorf("invalid drive-memory id %q: must contain only letters, digits, underscores, or hyphens (1-64 characters)", id)
		}
		entry.ID = id
		entry.DriveName = id
	}
	if driveName, ok := raw["drive-name"].(string); ok && driveName != "" {
		entry.DriveName = driveName
	}
	if description, ok := raw["description"].(string); ok {
		entry.Description = description
	}
	if diskSize, ok := raw["disk-size"].(string); ok {
		entry.DiskSize = diskSize
	}
	if prefetch, ok := raw["prefetch"].(bool); ok {
		entry.Prefetch = prefetch
	}
	if restoreOnly, ok := raw["restore-only"].(bool); ok {
		entry.RestoreOnly = restoreOnly
	}
	if err := parseDriveMemoryAllowedExtensions(raw, &entry); err != nil {
		return entry, err
	}
	validation, err := parseMemoryValidationConfig(raw, "tools.drive-memory.validation")
	if err != nil {
		return entry, err
	}
	entry.Validation = validation
	if len(entry.AllowedExtensions) == 0 {
		entry.AllowedExtensions = constants.DefaultAllowedMemoryExtensions
	}
	return entry, nil
}

func parseDriveMemoryAllowedExtensions(raw map[string]any, entry *DriveMemoryEntry) error {
	value, exists := raw["allowed-extensions"]
	if !exists {
		return nil
	}
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	entry.AllowedExtensions = make([]string, 0, len(values))
	for _, value := range values {
		extension, ok := value.(string)
		if !ok {
			continue
		}
		if !isValidFileExtension(extension) {
			return fmt.Errorf("invalid allowed-extension %q: must start with '.' followed by alphanumeric characters only (e.g. .json)", extension)
		}
		entry.AllowedExtensions = append(entry.AllowedExtensions, extension)
	}
	return nil
}

func parseDriveMemoryEntries(values []any) ([]DriveMemoryEntry, error) {
	entries := make([]DriveMemoryEntry, 0, len(values))
	ids := make(map[string]struct{}, len(values))
	for _, value := range values {
		raw, ok := value.(map[string]any)
		if !ok {
			continue
		}
		entry, err := parseDriveMemoryEntry(raw, "default")
		if err != nil {
			return nil, err
		}
		if _, exists := ids[entry.ID]; exists {
			return nil, fmt.Errorf("duplicate drive-memory id %q: each drive must have a unique id", entry.ID)
		}
		ids[entry.ID] = struct{}{}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (c *Compiler) extractDriveMemoryConfig(toolsConfig *ToolsConfig) (*DriveMemoryConfig, error) {
	if toolsConfig == nil || toolsConfig.DriveMemory == nil {
		return nil, nil
	}
	config := &DriveMemoryConfig{}
	value := toolsConfig.DriveMemory.Raw
	if value == nil {
		config.Drives = defaultDriveMemoryEntries()
		return config, nil
	}
	if enabled, ok := value.(bool); ok {
		if enabled {
			config.Drives = defaultDriveMemoryEntries()
		}
		return config, nil
	}
	if values, ok := value.([]any); ok {
		entries, err := parseDriveMemoryEntries(values)
		if err != nil {
			return nil, err
		}
		config.Drives = entries
		return config, nil
	}
	if raw, ok := value.(map[string]any); ok {
		entry, err := parseDriveMemoryEntry(raw, "default")
		if err != nil {
			return nil, err
		}
		config.Drives = []DriveMemoryEntry{entry}
		return config, nil
	}
	return nil, nil
}
