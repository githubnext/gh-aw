package workflow

import "testing"

func TestFormatSafeJobRunsOn(t *testing.T) {
	tests := []struct {
		name          string
		runsOn        RunsOnValue
		runsOnArray   bool
		defaultRunsOn string
		want          string
	}{
		{
			name:          "nil value defaults",
			runsOn:        nil,
			runsOnArray:   false,
			defaultRunsOn: "ubuntu-latest",
			want:          "runs-on: ubuntu-latest",
		},
		{
			name:          "empty array-shaped value defaults",
			runsOn:        RunsOnValue{},
			runsOnArray:   true,
			defaultRunsOn: "ubuntu-latest",
			want:          "runs-on: ubuntu-latest",
		},
		{
			name:          "single empty-string element treated as unset",
			runsOn:        RunsOnValue{""},
			runsOnArray:   true,
			defaultRunsOn: "ubuntu-latest",
			want:          "runs-on: ubuntu-latest",
		},
		{
			name:          "scalar value rendered inline",
			runsOn:        RunsOnValue{"self-hosted"},
			runsOnArray:   false,
			defaultRunsOn: "ubuntu-latest",
			want:          "runs-on: self-hosted",
		},
		{
			name:          "single-element array shape renders as YAML sequence",
			runsOn:        RunsOnValue{"self-hosted"},
			runsOnArray:   true,
			defaultRunsOn: "ubuntu-latest",
			want:          "runs-on:\n  - self-hosted",
		},
		{
			name:          "multi-element array shape renders YAML sequence",
			runsOn:        RunsOnValue{"self-hosted", "linux"},
			runsOnArray:   true,
			defaultRunsOn: "ubuntu-latest",
			want:          "runs-on:\n  - self-hosted\n  - linux",
		},
		{
			name:          "multi-element value not marked as array still falls back to FormatRunsOn",
			runsOn:        RunsOnValue{"self-hosted", "linux"},
			runsOnArray:   false,
			defaultRunsOn: "ubuntu-latest",
			want:          `runs-on: ["self-hosted","linux"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSafeJobRunsOn(tt.runsOn, tt.runsOnArray, tt.defaultRunsOn)
			if got != tt.want {
				t.Errorf("formatSafeJobRunsOn(%#v, %v, %q) = %q, want %q", tt.runsOn, tt.runsOnArray, tt.defaultRunsOn, got, tt.want)
			}
		})
	}
}
