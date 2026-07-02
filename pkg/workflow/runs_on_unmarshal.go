package workflow

import (
	"encoding/json"

	"github.com/github/gh-aw/pkg/logger"
)

var runsOnUnmarshalLog = logger.New("workflow:runs_on_unmarshal")

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
		runsOnUnmarshalLog.Printf("Failed to unmarshal SafeOutputsConfig: %v", err)
		return err
	}

	c.RunsOn = renderRunsOnSnippet(aux.RunsOn)
	runsOnUnmarshalLog.Printf("Unmarshaled SafeOutputsConfig with runs-on snippet (%d bytes)", len(c.RunsOn))
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
		runsOnUnmarshalLog.Printf("Failed to unmarshal ThreatDetectionConfig: %v", err)
		return err
	}

	c.RunsOn = renderRunsOnSnippet(aux.RunsOn)
	runsOnUnmarshalLog.Printf("Unmarshaled ThreatDetectionConfig with runs-on snippet (%d bytes)", len(c.RunsOn))
	return nil
}
