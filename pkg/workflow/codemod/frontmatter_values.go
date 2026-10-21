package codemod

import (
	"bytes"

	"github.com/goccy/go-yaml"
)

func equalValues(left, right any) (bool, error) {
	first, err := yaml.MarshalWithOptions(left, yaml.JSON())
	if err != nil {
		return false, err
	}
	second, err := yaml.MarshalWithOptions(right, yaml.JSON())
	if err != nil {
		return false, err
	}
	return bytes.Equal(first, second), nil
}

func containsValue(values []any, value any) (bool, error) {
	for _, existing := range values {
		equal, err := equalValues(existing, value)
		if err != nil || equal {
			return equal, err
		}
	}
	return false, nil
}
