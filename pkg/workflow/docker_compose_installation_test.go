//go:build !integration

package workflow

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDockerComposeInstallerRejectsWritableVerifier(t *testing.T) {
	t.Parallel()
	for _, arch := range []string{"x86_64", "amd64", "aarch64", "arm64"} {
		for _, tampered := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/tampered=%t", arch, tampered), func(t *testing.T) {
				runDockerComposeInstaller(t, arch, tampered)
			})
		}
	}
	t.Run("unsupported", func(t *testing.T) {
		runDockerComposeInstaller(t, "riscv64", true)
	})
}

func runDockerComposeInstaller(t *testing.T, arch string, tampered bool) {
	t.Helper()
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	helperDir := filepath.Join(root, "gh-aw", "actions")
	for _, dir := range []string{bin, helperDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(helperDir, "release_integrity.sh"), `
printf sourced > "$RUNNER_TEMP/helper-sourced"
verify_release_digest() { return 0; }
`)
	payload := "#!/bin/bash\nprintf executed > \"$RUNNER_TEMP/plugin-executed\"\n"
	write(filepath.Join(root, "payload"), payload)
	write(filepath.Join(bin, "uname"), "#!/bin/bash\nprintf '%s\\n' '"+arch+"'\n")
	write(filepath.Join(bin, "mktemp"), "#!/bin/bash\nprintf '%s\\n' \"$RUNNER_TEMP/download\"\n")
	write(filepath.Join(bin, "curl"), `#!/bin/bash
printf '%s\n' "$*" > "$RUNNER_TEMP/curl-args"
[[ "$1" == "--proto" && "$2" == "=https" && "$3" == "--proto-redir" && "$4" == "=https" ]] || exit 1
cp "$RUNNER_TEMP/payload" "${@: -1}"
`)
	install, err := exec.LookPath("install")
	if err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(bin, "install"), "#!/bin/bash\nprintf installed > \"$RUNNER_TEMP/plugin-installed\"\nexec \""+install+"\" \"$@\"\n")
	write(filepath.Join(bin, "docker"), "#!/bin/bash\n\"$DOCKER_CONFIG/cli-plugins/docker-compose\"\n")

	lines := generateDockerComposeInstallStep()[3:]
	var script strings.Builder
	for _, line := range lines {
		script.WriteString(strings.TrimPrefix(line, "          "))
		script.WriteByte('\n')
	}
	run := script.String()
	if !tampered {
		// Replace only the reviewed compiler pins with a trusted fixture pin,
		// leaving the generated verifier and downloaded bytes unchanged.
		fixtureDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
		for _, pin := range []string{
			"9040bd35b2cc0783ce6c5de491de7e52e24d4137dbfc5de8a524f718fc23556c",
			"d1148609319706a57b755ff0f61d604a63a8cf57adb24c17535baa766ff14b4f",
		} {
			run = strings.ReplaceAll(run, pin, fixtureDigest)
		}
	}
	cmd := exec.Command("bash", "-c", run)
	cmd.Env = append(os.Environ(),
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"RUNNER_TEMP="+root,
		"DOCKER_CONFIG="+filepath.Join(root, "docker"),
	)
	output, err := cmd.CombinedOutput()
	if tampered {
		if err == nil {
			t.Fatalf("untrusted Compose bytes must fail: %s", output)
		}
		expected := "Trusted SHA256 verification failed"
		if arch == "riscv64" {
			expected = "Unsupported architecture"
		}
		if !strings.Contains(string(output), expected) {
			t.Fatalf("expected %q, got %v: %s", expected, err, output)
		}
	} else if err != nil {
		t.Fatalf("verified Compose fixture must install: %v: %s", err, output)
	}
	for _, marker := range []string{"helper-sourced", "plugin-installed", "plugin-executed", "docker/cli-plugins/docker-compose"} {
		_, err := os.Stat(filepath.Join(root, marker))
		wantPresent := !tampered && marker != "helper-sourced"
		if wantPresent && err != nil {
			t.Errorf("%s should exist: %v", marker, err)
		} else if !wantPresent && !os.IsNotExist(err) {
			t.Errorf("%s must not exist, got %v", marker, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "download")); !os.IsNotExist(err) {
		t.Errorf("download must be cleaned up: %v", err)
	}
	if arch != "riscv64" {
		args, err := os.ReadFile(filepath.Join(root, "curl-args"))
		if err != nil {
			t.Fatal(err)
		}
		assetArch := "x86_64"
		if arch == "arm64" || arch == "aarch64" {
			assetArch = "aarch64"
		}
		if !strings.Contains(string(args), "https://github.com/docker/compose/releases/download/v2.36.2/docker-compose-linux-"+assetArch+" -o ") {
			t.Errorf("unexpected Compose asset download: %s", args)
		}
	}
}
