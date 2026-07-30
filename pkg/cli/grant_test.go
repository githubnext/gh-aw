//go:build !integration

package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

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

func TestGrantRunOnImageRejectsInvalidImageRef(t *testing.T) {
	_, err := grantRunOnImage("bad image", "/tmp/policy.yaml", false)
	if err == nil {
		t.Fatal("Expected grantRunOnImage to reject whitespace in image ref")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid whitespace/control characters") {
		t.Fatalf("Expected whitespace validation error, got: %v", err)
	}
}
