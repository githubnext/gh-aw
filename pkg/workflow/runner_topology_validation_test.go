package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateArcDindRootless(t *testing.T) {
	t.Run("no error when topology is not arc-dind", func(t *testing.T) {
		wd := &WorkflowData{
			CustomSteps: "      - run: sudo apt-get install -y gcc\n",
		}
		assert.NoError(t, validateArcDindRootless(wd))
	})

	t.Run("no error when no steps use sudo", func(t *testing.T) {
		wd := &WorkflowData{
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
			CustomSteps:  "      - run: echo hello\n",
		}
		assert.NoError(t, validateArcDindRootless(wd))
	})

	t.Run("error when custom steps use sudo", func(t *testing.T) {
		wd := &WorkflowData{
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
			CustomSteps:  "      - run: sudo apt-get install -y gcc\n",
		}
		err := validateArcDindRootless(wd)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "arc-dind")
		assert.Contains(t, err.Error(), "sudo")
	})

	t.Run("error when pre-steps use apt-get install", func(t *testing.T) {
		wd := &WorkflowData{
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
			PreSteps:     "      - run: apt-get install -y build-essential\n",
		}
		err := validateArcDindRootless(wd)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "apt-get install")
		assert.Contains(t, err.Error(), "pre-steps")
	})

	t.Run("error when post-steps use sudo", func(t *testing.T) {
		wd := &WorkflowData{
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
			PostSteps:    "      - run: sudo rm -rf /tmp/cache\n",
		}
		err := validateArcDindRootless(wd)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "post-steps")
	})

	t.Run("no error when steps are empty", func(t *testing.T) {
		wd := &WorkflowData{
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
		}
		assert.NoError(t, validateArcDindRootless(wd))
	})

	t.Run("ignores comments containing sudo", func(t *testing.T) {
		wd := &WorkflowData{
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
			CustomSteps:  "      - run: |\n          # don't use sudo here\n          echo hello\n",
		}
		assert.NoError(t, validateArcDindRootless(wd))
	})

	t.Run("no false positive on step name mentioning sudo", func(t *testing.T) {
		wd := &WorkflowData{
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
			CustomSteps:  "      - name: avoid sudo operations\n        run: echo hello\n",
		}
		assert.NoError(t, validateArcDindRootless(wd))
	})
}

func TestFindRootRequiringPatterns(t *testing.T) {
	t.Run("empty for clean content", func(t *testing.T) {
		assert.Empty(t, findRootRequiringPatterns("echo hello\nls -la\n"))
	})

	t.Run("detects sudo", func(t *testing.T) {
		violations := findRootRequiringPatterns("sudo apt-get update\nsudo apt-get install gcc\n")
		assert.Contains(t, violations, "sudo")
	})

	t.Run("detects apt-get install", func(t *testing.T) {
		violations := findRootRequiringPatterns("apt-get install -y gcc\n")
		assert.Contains(t, violations, "apt-get install")
	})

	t.Run("detects apt install", func(t *testing.T) {
		violations := findRootRequiringPatterns("apt install -y gcc\n")
		assert.Contains(t, violations, "apt-get install")
	})

	t.Run("skips comments", func(t *testing.T) {
		assert.Empty(t, findRootRequiringPatterns("# sudo apt-get install gcc\n"))
	})

	t.Run("deduplicates violations", func(t *testing.T) {
		violations := findRootRequiringPatterns("sudo ls\nsudo rm\n")
		count := 0
		for _, v := range violations {
			if v == "sudo" {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("no false positive on YAML metadata name mentioning sudo", func(t *testing.T) {
		assert.Empty(t, findRootRequiringPatterns("      - name: avoid sudo operations\n        run: echo hello\n"))
	})

	t.Run("no false positive on YAML if-expression mentioning sudo", func(t *testing.T) {
		assert.Empty(t, findRootRequiringPatterns("        if: contains(steps.check.outputs.result, 'sudo')\n"))
	})
}

func TestContainsSudoCommand(t *testing.T) {
	t.Run("detects sudo at start of line", func(t *testing.T) {
		assert.True(t, containsSudoCommand("sudo apt-get install gcc"))
	})

	t.Run("detects sudo in inline run: value", func(t *testing.T) {
		assert.True(t, containsSudoCommand("- run: sudo rm -rf /tmp/cache"))
	})

	t.Run("detects sudo after &&", func(t *testing.T) {
		assert.True(t, containsSudoCommand("echo hi && sudo rm -rf /tmp/cache"))
	})

	t.Run("detects sudo after ||", func(t *testing.T) {
		assert.True(t, containsSudoCommand("false || sudo fallback"))
	})

	t.Run("detects sudo after semicolon", func(t *testing.T) {
		assert.True(t, containsSudoCommand("echo hi; sudo rm -rf /tmp/cache"))
	})

	t.Run("detects sudo after pipe", func(t *testing.T) {
		assert.True(t, containsSudoCommand("cat file | sudo tee /etc/conf"))
	})

	t.Run("no false positive: sudo in yaml name value", func(t *testing.T) {
		assert.False(t, containsSudoCommand("- name: avoid sudo operations"))
	})

	t.Run("no false positive: sudo in yaml if expression", func(t *testing.T) {
		assert.False(t, containsSudoCommand("if: contains(steps.check.outputs.result, 'sudo')"))
	})

	t.Run("no false positive: sudo mentioned in comment", func(t *testing.T) {
		assert.False(t, containsSudoCommand("# sudo apt-get install"))
	})

	t.Run("no false positive: sudo inside a quoted string mid-line", func(t *testing.T) {
		assert.False(t, containsSudoCommand("echo \"do not use sudo here\""))
	})

	t.Run("no false positive: run: with multiline block indicator", func(t *testing.T) {
		assert.False(t, containsSudoCommand("run: |"))
	})
}
