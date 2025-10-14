package workflow

import (
	"testing"
)

func TestContainerWithCustomArgs(t *testing.T) {
	// Test that custom args are preserved when using container field
	config := map[string]interface{}{
		"container": "test",
		"version":   "latest",
		"args":      []interface{}{"-v", "/tmp:/tmp:ro", "-w", "/tmp"},
		"env": map[string]interface{}{
			"TEST_VAR": "value",
		},
		"allowed": []interface{}{"*"},
	}

	result, err := getMCPConfig(config, "test-tool")
	if err != nil {
		t.Fatalf("getMCPConfig failed: %v", err)
	}

	// Check that command is docker
	if result.Command != "docker" {
		t.Errorf("Expected command 'docker', got '%s'", result.Command)
	}

	// Check that args contain the expected elements (with version appended to container)
	expectedArgs := []string{"run", "--rm", "-i", "-e", "TEST_VAR", "-v", "/tmp:/tmp:ro", "-w", "/tmp", "test:latest"}
	if len(result.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d: %v", len(expectedArgs), len(result.Args), result.Args)
	}

	// Check specific args
	hasVolume := false
	hasWorkdir := false
	for i, arg := range result.Args {
		if arg == "-v" && i+1 < len(result.Args) && result.Args[i+1] == "/tmp:/tmp:ro" {
			hasVolume = true
		}
		if arg == "-w" && i+1 < len(result.Args) && result.Args[i+1] == "/tmp" {
			hasWorkdir = true
		}
	}

	if !hasVolume {
		t.Error("Expected volume mount '-v /tmp:/tmp:ro' in args")
	}
	if !hasWorkdir {
		t.Error("Expected working directory '-w /tmp' in args")
	}

	// Check that container with version is the last arg
	if result.Args[len(result.Args)-1] != "test:latest" {
		t.Errorf("Expected container 'test:latest' as last arg, got '%s'", result.Args[len(result.Args)-1])
	}
}

func TestContainerWithoutCustomArgs(t *testing.T) {
	// Test that container works without custom args (existing behavior)
	config := map[string]interface{}{
		"container": "test:latest",
		"env": map[string]interface{}{
			"TEST_VAR": "value",
		},
		"allowed": []interface{}{"*"},
	}

	result, err := getMCPConfig(config, "test-tool")
	if err != nil {
		t.Fatalf("getMCPConfig failed: %v", err)
	}

	// Check that command is docker
	if result.Command != "docker" {
		t.Errorf("Expected command 'docker', got '%s'", result.Command)
	}

	// Check that args contain the expected elements (no custom args)
	expectedArgs := []string{"run", "--rm", "-i", "-e", "TEST_VAR", "test:latest"}
	if len(result.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d: %v", len(expectedArgs), len(result.Args), result.Args)
	}

	// Check that container is the last arg (backward compatibility - container with :tag in it)
	if result.Args[len(result.Args)-1] != "test:latest" {
		t.Errorf("Expected container 'test:latest' as last arg, got '%s'", result.Args[len(result.Args)-1])
	}
}

func TestContainerWithVersionField(t *testing.T) {
	// Test that version field properly appends to container
	config := map[string]interface{}{
		"container": "ghcr.io/test/image",
		"version":   "v1.2.3",
		"env": map[string]interface{}{
			"TEST_VAR": "value",
		},
		"allowed": []interface{}{"*"},
	}

	result, err := getMCPConfig(config, "test-tool")
	if err != nil {
		t.Fatalf("getMCPConfig failed: %v", err)
	}

	// Check that command is docker
	if result.Command != "docker" {
		t.Errorf("Expected command 'docker', got '%s'", result.Command)
	}

	// Check that container with version is the last arg
	expectedContainer := "ghcr.io/test/image:v1.2.3"
	if result.Args[len(result.Args)-1] != expectedContainer {
		t.Errorf("Expected container '%s' as last arg, got '%s'", expectedContainer, result.Args[len(result.Args)-1])
	}
}

func TestContainerWithEntrypointArgs(t *testing.T) {
	// Test that entrypointArgs are added after the container image
	config := map[string]interface{}{
		"container":      "test-image",
		"version":        "latest",
		"entrypointArgs": []interface{}{"--config", "/app/config.json", "--verbose"},
		"env": map[string]interface{}{
			"TEST_VAR": "value",
		},
		"allowed": []interface{}{"*"},
	}

	result, err := getMCPConfig(config, "test-tool")
	if err != nil {
		t.Fatalf("getMCPConfig failed: %v", err)
	}

	// Check that command is docker
	if result.Command != "docker" {
		t.Errorf("Expected command 'docker', got '%s'", result.Command)
	}

	// Expected args structure: ["run", "--rm", "-i", "-e", "TEST_VAR", "test-image:latest", "--config", "/app/config.json", "--verbose"]
	expectedArgs := []string{"run", "--rm", "-i", "-e", "TEST_VAR", "test-image:latest", "--config", "/app/config.json", "--verbose"}
	if len(result.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d: %v", len(expectedArgs), len(result.Args), result.Args)
	}

	// Check that entrypoint args come after container image
	containerImageIndex := -1
	for i, arg := range result.Args {
		if arg == "test-image:latest" {
			containerImageIndex = i
			break
		}
	}

	if containerImageIndex == -1 {
		t.Fatal("Container image not found in args")
	}

	// Verify entrypoint args are after container image
	if containerImageIndex+1 >= len(result.Args) {
		t.Error("No args found after container image")
	} else {
		// Check each entrypoint arg
		entrypointArgsStart := containerImageIndex + 1
		expectedEntrypointArgs := []string{"--config", "/app/config.json", "--verbose"}

		for i, expectedArg := range expectedEntrypointArgs {
			actualIndex := entrypointArgsStart + i
			if actualIndex >= len(result.Args) {
				t.Errorf("Missing entrypoint arg at index %d: expected '%s'", i, expectedArg)
				continue
			}
			if result.Args[actualIndex] != expectedArg {
				t.Errorf("Entrypoint arg %d: expected '%s', got '%s'", i, expectedArg, result.Args[actualIndex])
			}
		}
	}
}

func TestContainerWithArgsAndEntrypointArgs(t *testing.T) {
	// Test that both args (before container) and entrypointArgs (after container) work together
	config := map[string]interface{}{
		"container":      "test-image",
		"version":        "v1.0",
		"args":           []interface{}{"-v", "/host:/container"},
		"entrypointArgs": []interface{}{"serve", "--port", "8080"},
		"env": map[string]interface{}{
			"ENV_VAR": "value",
		},
		"allowed": []interface{}{"*"},
	}

	result, err := getMCPConfig(config, "test-tool")
	if err != nil {
		t.Fatalf("getMCPConfig failed: %v", err)
	}

	// Check that command is docker
	if result.Command != "docker" {
		t.Errorf("Expected command 'docker', got '%s'", result.Command)
	}

	// Expected structure: ["run", "--rm", "-i", "-e", "ENV_VAR", "-v", "/host:/container", "test-image:v1.0", "serve", "--port", "8080"]
	expectedArgs := []string{"run", "--rm", "-i", "-e", "ENV_VAR", "-v", "/host:/container", "test-image:v1.0", "serve", "--port", "8080"}
	if len(result.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d: %v", len(expectedArgs), len(result.Args), result.Args)
	}

	// Find container image position
	containerImageIndex := -1
	for i, arg := range result.Args {
		if arg == "test-image:v1.0" {
			containerImageIndex = i
			break
		}
	}

	if containerImageIndex == -1 {
		t.Fatal("Container image not found in args")
	}

	// Verify args come before container image (specifically the volume mount)
	hasVolumeBefore := false
	for i := 0; i < containerImageIndex; i++ {
		if result.Args[i] == "-v" && i+1 < containerImageIndex && result.Args[i+1] == "/host:/container" {
			hasVolumeBefore = true
			break
		}
	}
	if !hasVolumeBefore {
		t.Error("Expected volume mount args before container image")
	}

	// Verify entrypoint args come after container image
	expectedEntrypointArgs := []string{"serve", "--port", "8080"}
	entrypointArgsStart := containerImageIndex + 1

	for i, expectedArg := range expectedEntrypointArgs {
		actualIndex := entrypointArgsStart + i
		if actualIndex >= len(result.Args) {
			t.Errorf("Missing entrypoint arg at index %d: expected '%s'", i, expectedArg)
			continue
		}
		if result.Args[actualIndex] != expectedArg {
			t.Errorf("Entrypoint arg %d: expected '%s', got '%s'", i, expectedArg, result.Args[actualIndex])
		}
	}
}
