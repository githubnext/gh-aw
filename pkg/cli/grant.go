package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
)

var grantLog = logger.New("cli:grant")

const grantPolicyFilename = ".grant.yaml"

type grantOutput struct {
	Tool string `json:"tool"`
	Run  struct {
		Targets []grantTargetResult `json:"targets"`
	} `json:"run"`
}

type grantTargetResult struct {
	Source struct {
		Ref string `json:"ref"`
	} `json:"source"`
	Evaluation grantTargetEvaluation `json:"evaluation"`
}

type grantTargetEvaluation struct {
	Status   string `json:"status"`
	Findings struct {
		Packages []grantPackageFinding `json:"packages"`
	} `json:"findings"`
}

type grantPackageFinding struct {
	Name     string               `json:"name"`
	Version  string               `json:"version"`
	Decision string               `json:"decision"`
	Licenses []grantLicenseDetail `json:"licenses"`
}

type grantLicenseDetail struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// runGrantOnLockFiles extracts container image references from the gh-aw-manifest
// headers in the provided lock files, deduplicates them, and runs the grant
// license scanner on each unique image via Docker.
func runGrantOnLockFiles(lockFiles []string, verbose bool, strict bool) error {
	if len(lockFiles) == 0 {
		return nil
	}

	images := collectContainerImagesFromLockFiles(lockFiles)
	if len(images) == 0 {
		grantLog.Print("No container images found in lock files")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("No container images found in lock files to scan with grant"))
		}
		return nil
	}

	policyFile, err := grantPolicyFile()
	if err != nil {
		return err
	}

	if len(images) == 1 {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Running grant license scanner on 1 container image"))
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(
			fmt.Sprintf("Running grant license scanner on %d container images", len(images))))
	}

	totalFindings := 0
	var scanErrors []string

	for _, img := range images {
		imageRef := img.PinnedImage
		if imageRef == "" {
			imageRef = img.Image
		}
		displayName := img.Image
		if displayName == "" {
			displayName = imageRef
		}

		output, err := grantRunOnImage(imageRef, policyFile, verbose)
		if err != nil {
			grantLog.Printf("Grant scan failed for %s: %v", displayName, err)
			scanErrors = append(scanErrors, fmt.Sprintf("%s: %v", displayName, err))
			continue
		}

		count, err := grantDisplayFindings(displayName, output)
		if err != nil {
			grantLog.Printf("Grant findings invalid for %s: %v", displayName, err)
			scanErrors = append(scanErrors, fmt.Sprintf("%s: %v", displayName, err))
			continue
		}
		totalFindings += count
	}

	if len(scanErrors) > 0 {
		errMsg := fmt.Sprintf("grant scan failed for %d image(s): %s",
			len(scanErrors), strings.Join(scanErrors, "; "))
		if strict {
			return errors.New(errMsg)
		}
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(errMsg))
	}

	if strict && totalFindings > 0 {
		return fmt.Errorf("strict mode: grant found %d license policy finding(s) in container images", totalFindings)
	}

	return nil
}

func grantPolicyFile() (string, error) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return "", fmt.Errorf("grant requires a git repository checkout to locate %s: %w", grantPolicyFilename, err)
	}

	policyFile := filepath.Join(repoRoot, grantPolicyFilename)
	info, err := os.Stat(policyFile)
	if err != nil {
		return "", fmt.Errorf(
			"grant requires %s at the repository root (create it or run compile without --grant): %w",
			grantPolicyFilename,
			err,
		)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("grant policy file is not a regular file: %s", policyFile)
	}

	return policyFile, nil
}

func grantRunOnImage(imageRef, policyFile string, verbose bool) (*grantOutput, error) {
	containerPolicyPath := "/tmp/gh-aw-grant-policy.yaml"

	// #nosec G204 -- imageRef and policyFile are derived from compiled lock files and the
	// current repository checkout. exec.Command passes arguments directly without a shell.
	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-v", policyFile+":"+containerPolicyPath+":ro",
		GrantImage,
		"--config", containerPolicyPath,
		"--output", "json",
		"check",
		imageRef,
	)

	if verbose {
		dockerCmd := fmt.Sprintf("docker run --rm -v %s:%s:ro %s --config %s --output json check %s",
			policyFile, containerPolicyPath, GrantImage, containerPolicyPath, imageRef)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Run grant directly: "+dockerCmd))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	var output grantOutput
	if stdout.Len() > 0 && strings.HasPrefix(strings.TrimSpace(stdout.String()), "{") {
		if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
			return nil, fmt.Errorf("failed to parse grant JSON output for %s: %w", imageRef, err)
		}
	}

	if len(output.Run.Targets) == 0 {
		if runErr != nil {
			stderrStr := strings.TrimSpace(stderr.String())
			if stderrStr != "" {
				return nil, fmt.Errorf("grant failed for %s: %s", imageRef, stderrStr)
			}
			return nil, fmt.Errorf("grant failed for %s: %w", imageRef, runErr)
		}
		return nil, fmt.Errorf("grant produced no scan targets for %s", imageRef)
	}

	for _, target := range output.Run.Targets {
		if target.Evaluation.Status == "error" {
			targetRef := target.Source.Ref
			if targetRef == "" {
				targetRef = imageRef
			}
			return nil, fmt.Errorf("grant reported an evaluation error for %s", targetRef)
		}
	}

	return &output, nil
}

func grantDisplayFindings(imageTag string, output *grantOutput) (int, error) {
	if output == nil || len(output.Run.Targets) == 0 {
		return 0, nil
	}

	total := 0
	for _, target := range output.Run.Targets {
		for _, pkg := range target.Evaluation.Findings.Packages {
			if pkg.Decision != "deny" {
				continue
			}

			licenses := "no licenses found"
			if len(pkg.Licenses) > 0 {
				names := make([]string, 0, len(pkg.Licenses))
				for _, license := range pkg.Licenses {
					name := license.ID
					if name == "" {
						name = license.Name
					}
					if name != "" {
						names = append(names, name)
					}
				}
				if len(names) > 0 {
					licenses = strings.Join(names, ", ")
				}
			}

			message := fmt.Sprintf("license policy violation: %s (%s)", grantPackageRef(pkg), licenses)
			compilerErr := console.CompilerError{
				Position: console.ErrorPosition{
					File:   imageTag,
					Line:   1,
					Column: 1,
				},
				Type:    "error",
				Message: message,
			}
			fmt.Fprint(os.Stderr, console.FormatError(compilerErr))
			total++
		}
	}

	return total, nil
}

func grantPackageRef(pkg grantPackageFinding) string {
	if pkg.Version == "" {
		return pkg.Name
	}
	return pkg.Name + "@" + pkg.Version
}
