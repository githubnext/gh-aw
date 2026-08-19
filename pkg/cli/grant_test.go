//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrantDisplayFindings_NilOutput(t *testing.T) {
	t.Parallel()
	count, err := grantDisplayFindings("test-image:latest", nil)
	if err != nil {
		t.Fatalf("Expected no error for nil output, got: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 findings for nil output, got %d", count)
	}
}

func TestGrantDisplayFindings_WithDeniedPackages(t *testing.T) {
	t.Parallel()
	output := &grantOutput{}
	output.Run.Targets = []grantTargetResult{
		{
			Evaluation: grantTargetEvaluation{
				Status: "noncompliant",
				Findings: struct {
					Packages []grantPackageFinding `json:"packages"`
				}{
					Packages: []grantPackageFinding{
						{
							Name:     "openssl",
							Version:  "1.0.0",
							Decision: "deny",
							Licenses: []grantLicenseDetail{{ID: "GPL-3.0-only"}},
						},
						{
							Name:     "nolicense",
							Decision: "deny",
						},
						{
							Name:     "allowed",
							Version:  "1.0.0",
							Decision: "allow",
						},
					},
				},
			},
		},
	}

	count, err := grantDisplayFindings("ubuntu:24.04", output)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 denied packages, got %d", count)
	}
}

func TestRunGrantOnLockFiles_NoLockFiles(t *testing.T) {
	t.Parallel()
	err := runGrantOnLockFiles([]string{}, false, false)
	if err != nil {
		t.Errorf("Expected no error for empty lock file list, got: %v", err)
	}
}

func TestGrantPolicyFile(t *testing.T) {
	t.Parallel()
	policyFile, err := grantPolicyFile()
	if err != nil {
		t.Fatalf("Expected grant policy file, got: %v", err)
	}
	if filepath.Base(policyFile) != grantPolicyFilename {
		t.Fatalf("Expected policy file basename %q, got %q", grantPolicyFilename, filepath.Base(policyFile))
	}
}

func TestGrantRunOnImageRejectsInvalidImageRef(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.yaml")
	require.NoError(t, os.WriteFile(policyFile, []byte("policy: true\n"), 0o644))

	testCases := []struct {
		name     string
		imageRef string
		want     string
	}{
		{name: "whitespace", imageRef: "bad image", want: "invalid whitespace/control characters"},
		{name: "empty", imageRef: "", want: "cannot be empty"},
		{name: "null byte", imageRef: "img\x00ref", want: "invalid whitespace/control characters"},
		{name: "escape character", imageRef: "img\x1bref", want: "invalid whitespace/control characters"},
		{name: "leading dash", imageRef: "-alpine:latest", want: "cannot start with '-'"},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := grantRunOnImage(tt.imageRef, policyFile, false)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestGrantRunOnImageVerboseCommandEscapesImageRef(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "policy file.yaml")
	require.NoError(t, os.WriteFile(policyFile, []byte("policy: true\n"), 0o644))

	volumeMount, err := buildDockerReadonlyFileMount(policyFile, grantContainerPolicyPath)
	require.NoError(t, err)

	command := shellJoinArgs([]string{
		"docker",
		"run",
		"--rm",
		"-v", volumeMount,
		GrantImage,
		"--config", grantContainerPolicyPath,
		"--output", "json",
		"check",
		"alpine;id",
	})

	if !strings.Contains(command, "'alpine;id'") {
		t.Fatalf("Expected image ref to be shell-escaped in verbose command, got: %s", command)
	}
}

func TestGrantIgnoredImagePatterns(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, grantPolicyFilename)
	require.NoError(t, os.WriteFile(policyFile, []byte("allow:\n  - MIT\nignore-images:\n  - \"ghcr.io/oraios/serena:*\"\n"), 0o644))

	patterns, err := grantIgnoredImagePatterns(policyFile)
	require.NoError(t, err)
	require.Equal(t, []string{"ghcr.io/oraios/serena:*"}, patterns)
}

func TestGrantIgnoredImagePatterns_MissingKey(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, grantPolicyFilename)
	require.NoError(t, os.WriteFile(policyFile, []byte("allow:\n  - MIT\n"), 0o644))

	patterns, err := grantIgnoredImagePatterns(policyFile)
	require.NoError(t, err)
	require.Empty(t, patterns)
}

func TestGrantIgnoredImagePatterns_InvalidYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, grantPolicyFilename)
	require.NoError(t, os.WriteFile(policyFile, []byte("ignore-images: [\n"), 0o644))

	_, err := grantIgnoredImagePatterns(policyFile)
	require.ErrorContains(t, err, "failed to parse grant policy file")
}

func TestGrantIsImageIgnored(t *testing.T) {
	t.Parallel()

	patterns := []string{"ghcr.io/oraios/serena:*"}

	testCases := []struct {
		name        string
		image       string
		pinnedImage string
		want        bool
	}{
		{
			name:        "tagged image matches glob",
			image:       "ghcr.io/oraios/serena:latest",
			pinnedImage: "ghcr.io/oraios/serena:latest@sha256:0944b2ffe66dbcddeed531694b6819d7f9efd8125b442b282a1cc863f570a03e",
			want:        true,
		},
		{
			name:        "pinned reference only",
			pinnedImage: "ghcr.io/oraios/serena:1.7.0@sha256:6c9459e4246a39c9deaa4f23fb05a526ac6e237b24c8e84a927a098fa1ab6730",
			want:        true,
		},
		{
			name:        "other image is scanned",
			image:       "ghcr.io/github/github-mcp-server:v1.9.0",
			pinnedImage: "ghcr.io/github/github-mcp-server:v1.9.0@sha256:881b53d6f75f69bdbc1b5b10fc2f1361717c19054143b3a8529fb5c32061a50e",
			want:        false,
		},
		{
			name:  "different registry path is scanned",
			image: "ghcr.io/other/oraios/serena:latest",
			want:  false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, grantIsImageIgnored(patterns, tt.image, tt.pinnedImage))
		})
	}
}

func TestGrantIsImageIgnored_NoPatterns(t *testing.T) {
	t.Parallel()

	require.False(t, grantIsImageIgnored(nil, "ghcr.io/oraios/serena:latest", ""))
	require.False(t, grantIsImageIgnored([]string{""}, "ghcr.io/oraios/serena:latest", ""))
}

func TestGrantIsImageIgnored_InvalidPattern(t *testing.T) {
	t.Parallel()

	require.False(t, grantIsImageIgnored([]string{"[z-a]"}, "ghcr.io/oraios/serena:latest"))
}

func TestGrantIsImageIgnored_GlobDoesNotMatchDeeperNamespace(t *testing.T) {
	t.Parallel()

	require.False(t, grantIsImageIgnored(
		[]string{"ghcr.io/*/serena:*"},
		"ghcr.io/oraios/development/serena:latest",
	))
}

func TestGrantRepositoryPolicyIgnoresSerenaImage(t *testing.T) {
	t.Parallel()

	policyFile, err := grantPolicyFile()
	require.NoError(t, err)

	patterns, err := grantIgnoredImagePatterns(policyFile)
	require.NoError(t, err)
	require.True(t, grantIsImageIgnored(patterns, "ghcr.io/oraios/serena:latest", ""),
		"the Serena MCP image should be excluded from license scanning")
}
