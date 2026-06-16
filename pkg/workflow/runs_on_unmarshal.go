package workflow

import "encoding/json"

// UnmarshalJSON supports string/array/object forms for safe-outputs.runs-on while
// storing a normalized runs-on YAML snippet for downstream rendering.
func (c *SafeOutputsConfig) UnmarshalJSON(data []byte) error {
	type alias SafeOutputsConfig
	aux := &struct {
		RunsOn any `json:"runs-on,omitempty"`
		*alias
	}{
		alias: (*alias)(c),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	c.RunsOn = renderRunsOnSnippet(aux.RunsOn)
	return nil
}

// UnmarshalJSON supports string/array/object forms for
// safe-outputs.threat-detection.runs-on while storing a normalized runs-on YAML
// snippet for downstream rendering.
func (c *ThreatDetectionConfig) UnmarshalJSON(data []byte) error {
	type alias ThreatDetectionConfig
	aux := &struct {
		RunsOn any `json:"runs-on,omitempty"`
		*alias
	}{
		alias: (*alias)(c),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	c.RunsOn = renderRunsOnSnippet(aux.RunsOn)
	return nil
}
