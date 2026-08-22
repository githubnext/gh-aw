//go:build !integration

package workflow

import (
	"reflect"
	"testing"
)

func TestGitHubMCPOptionsEmbedCommonOptions(t *testing.T) {
	tests := []struct {
		name       string
		optionType reflect.Type
	}{
		{name: "docker", optionType: reflect.TypeFor[GitHubMCPDockerOptions]()},
		{name: "remote", optionType: reflect.TypeFor[GitHubMCPRemoteOptions]()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, ok := tt.optionType.FieldByName("GitHubMCPCommonOptions")
			if !ok {
				t.Fatalf("expected %s to embed GitHubMCPCommonOptions", tt.optionType.Name())
			}
			if !field.Anonymous {
				t.Fatalf("expected %s.GitHubMCPCommonOptions to be embedded", tt.optionType.Name())
			}
		})
	}
}
