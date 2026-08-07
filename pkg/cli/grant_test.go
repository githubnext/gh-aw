//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type grantPolicy struct {
	Allow          []string `yaml:"allow"`
	IgnorePackages []string `yaml:"ignore-packages"`
}

func TestGrantDisplayFindings_NilOutput(t *testing.T) {
	count, err := grantDisplayFindings("test-image:latest", nil)
	if err != nil {
		t.Fatalf("Expected no error for nil output, got: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 findings for nil output, got %d", count)
	}
}

func TestGrantDisplayFindings_WithDeniedPackages(t *testing.T) {
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
	err := runGrantOnLockFiles([]string{}, false, false)
	if err != nil {
		t.Errorf("Expected no error for empty lock file list, got: %v", err)
	}
}

func TestGrantPolicyFile(t *testing.T) {
	policyFile, err := grantPolicyFile()
	if err != nil {
		t.Fatalf("Expected grant policy file, got: %v", err)
	}
	if filepath.Base(policyFile) != grantPolicyFilename {
		t.Fatalf("Expected policy file basename %q, got %q", grantPolicyFilename, filepath.Base(policyFile))
	}
}

func TestGrantPolicy_AllowsReviewedContainerLicenses(t *testing.T) {
	policyFile, err := grantPolicyFile()
	require.NoError(t, err)

	content, err := os.ReadFile(policyFile)
	require.NoError(t, err)

	var policy grantPolicy
	require.NoError(t, yaml.Unmarshal(content, &policy))

	for _, license := range []string{
		"Artistic-2.0",
		"BlueOak-1.0.0",
		"CC-BY-3.0",
		"CC0-1.0",
		"curl",
		"MPL-2.0",
		"X11",
		"Zlib",
	} {
		require.Contains(t, policy.Allow, license)
	}
}

func TestGrantPolicy_IgnoresReviewedBaseAndLocalPackages(t *testing.T) {
	policyFile, err := grantPolicyFile()
	require.NoError(t, err)

	content, err := os.ReadFile(policyFile)
	require.NoError(t, err)

	var policy grantPolicy
	require.NoError(t, yaml.Unmarshal(content, &policy))

	for _, pkg := range []string{
		"alpine-baselayout",
		"alpine-baselayout-data",
		"apk-tools",
		"awf-cli-proxy",
		"bash",
		"busybox",
		"busybox-binsh",
		"libgcc",
		"libapk",
		"libidn2",
		"libncursesw",
		"libstdc++",
		"libunistring",
		"musl-utils",
		"ncurses-terminfo-base",
		"node",
		"qrcode-terminal",
		"readline",
		"scanelf",
		"ssl_client",
		"zstd-libs",
	} {
		require.Contains(t, policy.IgnorePackages, pkg)
	}
}

func TestGrantRunOnImageRejectsInvalidImageRef(t *testing.T) {
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
			_, err := grantRunOnImage(tt.imageRef, policyFile, false)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestGrantRunOnImageVerboseCommandEscapesImageRef(t *testing.T) {
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
