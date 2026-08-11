package workflow

import (
	"encoding/base64"
	"fmt"
	"strconv"
)

const defaultMemoryValidationTimeoutSeconds = 30

type MemoryValidationConfig struct {
	Script  string `yaml:"script,omitempty"`
	Timeout int    `yaml:"timeout,omitempty"`
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
		config := &MemoryValidationConfig{}
		if script, ok := value["script"].(string); ok {
			config.Script = script
		}
		if timeout, exists := value["timeout"]; exists {
			parsed, err := parseMemoryValidationTimeout(timeout, fieldPath+".timeout")
			if err != nil {
				return nil, err
			}
			config.Timeout = parsed
		}
		return normalizeMemoryValidationConfig(config, fieldPath)
	default:
		return nil, fmt.Errorf("%s must be an object with script and optional timeout, or a script string", fieldPath)
	}
}

func parseMemoryValidationTimeout(value any, fieldPath string) (int, error) {
	switch v := value.(type) {
	case int:
		return validateMemoryValidationTimeout(v, fieldPath)
	case float64:
		return validateMemoryValidationTimeout(int(v), fieldPath)
	case uint64:
		if v > uint64(^uint(0)>>1) {
			return 0, fmt.Errorf("%s must be between 1 and 300 seconds", fieldPath)
		}
		return validateMemoryValidationTimeout(int(v), fieldPath)
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer number of seconds", fieldPath)
		}
		return validateMemoryValidationTimeout(parsed, fieldPath)
	default:
		return 0, fmt.Errorf("%s must be an integer number of seconds", fieldPath)
	}
}

func validateMemoryValidationTimeout(timeout int, fieldPath string) (int, error) {
	if timeout < 1 || timeout > 300 {
		return 0, fmt.Errorf("%s must be between 1 and 300 seconds", fieldPath)
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
	if config.Timeout == 0 {
		config.Timeout = defaultMemoryValidationTimeoutSeconds
	}
	return config, nil
}

func memoryValidationScriptBase64(config *MemoryValidationConfig) string {
	if config == nil || config.Script == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(config.Script))
}
