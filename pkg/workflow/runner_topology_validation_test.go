package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "arc-dind")
		assert.Contains(t, err.Error(), "sudo")
	})

	t.Run("error when pre-steps use apt-get install", func(t *testing.T) {
		wd := &WorkflowData{
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
			PreSteps:     "      - run: apt-get install -y build-essential\n",
		}
		err := validateArcDindRootless(wd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "apt-get install")
		assert.Contains(t, err.Error(), "pre-steps")
	})

	t.Run("error when post-steps use sudo", func(t *testing.T) {
		wd := &WorkflowData{
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
			PostSteps:    "      - run: sudo rm -rf /tmp/cache\n",
		}
		err := validateArcDindRootless(wd)
		assert.Error(t, err)
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
}
