package workflow

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
)

const (
	defaultMemoryValidationTimeoutMinutes = 1
	maxMemoryValidationTimeoutMinutes     = 5
)

type MemoryValidationConfig struct {
	Script         string `yaml:"script,omitempty"`
	TimeoutMinutes int    `yaml:"timeout-minutes,omitempty"`
}

func parseMemoryValidationConfig(configMap map[string]any, fieldPath string) (*MemoryValidationConfig, error) {
	raw, ok := configMap["validation"]
	if !ok {
		if script, ok := configMap["validation-script"].(string); ok {
			return normalizeMemoryValidationConfig(&MemoryValidationConfig{Script: script}, fieldPath)
		}
		if script, ok := configMap["custom-validation"].(string); ok {
			return normalizeMemoryValidationConfig(&MemoryValidationConfig{Script: script}, fieldPath)
		}
		return nil, nil
	}

	switch value := raw.(type) {
	case string:
		return normalizeMemoryValidationConfig(&MemoryValidationConfig{Script: value}, fieldPath)
	case map[string]any:
		if _, exists := value["timeout"]; exists {
			return nil, fmt.Errorf("%s.timeout has been renamed to %s.timeout-minutes", fieldPath, fieldPath)
		}
		config := &MemoryValidationConfig{}
		if script, ok := value["script"].(string); ok {
			config.Script = script
		}
		if timeout, exists := value["timeout-minutes"]; exists {
			parsed, err := parseMemoryValidationTimeoutMinutes(timeout, fieldPath+".timeout-minutes")
			if err != nil {
				return nil, err
			}
			config.TimeoutMinutes = parsed
		}
		return normalizeMemoryValidationConfig(config, fieldPath)
	default:
		return nil, fmt.Errorf("%s must be an object with script and optional timeout-minutes, or a script string", fieldPath)
	}
}

func parseMemoryValidationTimeoutMinutes(value any, fieldPath string) (int, error) {
	switch v := value.(type) {
	case int:
		return validateMemoryValidationTimeoutMinutes(v, fieldPath)
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) || v != math.Trunc(v) {
			return 0, fmt.Errorf("%s must be an integer number of minutes", fieldPath)
		}
		if v < 1 || v > maxMemoryValidationTimeoutMinutes {
			return 0, fmt.Errorf("%s must be between 1 and %d minutes", fieldPath, maxMemoryValidationTimeoutMinutes)
		}
		return validateMemoryValidationTimeoutMinutes(int(v), fieldPath)
	case uint64:
		if v > uint64(^uint(0)>>1) {
			return 0, fmt.Errorf("%s must be between 1 and %d minutes", fieldPath, maxMemoryValidationTimeoutMinutes)
		}
		return validateMemoryValidationTimeoutMinutes(int(v), fieldPath)
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer number of minutes", fieldPath)
		}
		return validateMemoryValidationTimeoutMinutes(parsed, fieldPath)
	default:
		return 0, fmt.Errorf("%s must be an integer number of minutes", fieldPath)
	}
}

func validateMemoryValidationTimeoutMinutes(timeout int, fieldPath string) (int, error) {
	if timeout < 1 || timeout > maxMemoryValidationTimeoutMinutes {
		return 0, fmt.Errorf("%s must be between 1 and %d minutes", fieldPath, maxMemoryValidationTimeoutMinutes)
	}
	return timeout, nil
}

func normalizeMemoryValidationConfig(config *MemoryValidationConfig, fieldPath string) (*MemoryValidationConfig, error) {
	if config == nil {
		return nil, nil
	}
	if config.Script == "" {
		return nil, fmt.Errorf("%s.script must not be empty", fieldPath)
	}
	if config.TimeoutMinutes == 0 {
		config.TimeoutMinutes = defaultMemoryValidationTimeoutMinutes
	}
	return config, nil
}

func memoryValidationTimeoutSeconds(config *MemoryValidationConfig) int {
	return config.TimeoutMinutes * 60
}

func memoryValidationScriptBase64(config *MemoryValidationConfig) string {
	if config == nil || config.Script == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(config.Script))
}

func memoryValidationStepID(prefix, memoryID string) string {
	return fmt.Sprintf("%s_%x", prefix, memoryID)
}
