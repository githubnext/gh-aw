//go:build !integration

package workflow

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/actions/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runArtifactProtocolScript(t *testing.T, script string, env ...string) (map[string]string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	outputPath := filepath.Join(t.TempDir(), "output")
	cmd := exec.CommandContext(ctx, "bash", "-c", script)
	cmd.Env = append(os.Environ(), append(env, "GITHUB_OUTPUT="+outputPath)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("artifact protocol failed: %w: %s", err, output)
	}
	result := make(map[string]string)
	content, err := os.ReadFile(outputPath)
	if os.IsNotExist(err) {
		return result, nil
	}
	require.NoError(t, err)
	for line := range strings.SplitSeq(string(content), "\n") {
		if key, value, ok := strings.Cut(line, "="); ok {
			result[key] = value
		}
	}
	return result, nil
}

func TestArtifactProducerAttemptRetainedArtifacts(t *testing.T) {
	for _, tc := range []struct {
		job             string
		producerAttempt string
		consumerAttempt string
	}{
		{job: "activation", producerAttempt: "1", consumerAttempt: "2"},
		{job: "agent", producerAttempt: "2", consumerAttempt: "3"},
	} {
		t.Run(tc.job, func(t *testing.T) {
			inputs := `{"invocation":"platform-check"}`
			producer, err := runArtifactProtocolScript(t, setup.ArtifactPrefixScript,
				"INPUTS_JSON="+inputs, "GITHUB_RUN_ATTEMPT="+tc.producerAttempt)
			require.NoError(t, err)
			current, err := runArtifactProtocolScript(t, setup.ArtifactPrefixScript,
				"INPUTS_JSON="+inputs, "GITHUB_RUN_ATTEMPT="+tc.consumerAttempt)
			require.NoError(t, err)
			assert.NotEqual(t, producer["prefix"], current["prefix"])
			assert.Equal(t, producer["identity_prefix"], current["identity_prefix"])
			metadata := t.TempDir()
			_, err = runArtifactProtocolScript(t, setup.ArtifactProducerScript,
				"METADATA_DIRECTORY="+metadata, "PRODUCER_JOB="+tc.job,
				"PRODUCER_ATTEMPT="+tc.producerAttempt, "INVOCATION_ID="+producer["identity_prefix"])
			require.NoError(t, err)
			resolved, err := runArtifactProtocolScript(t, setup.ArtifactProducerResolverScript+"\n"+setup.ArtifactPrefixScript,
				"METADATA_DIRECTORY="+metadata, "PRODUCER_JOB="+tc.job, "PRODUCER_RESULT=success",
				"INVOCATION_ID="+current["identity_prefix"], "INPUTS_JSON="+inputs,
				"GITHUB_RUN_ATTEMPT="+tc.consumerAttempt)
			require.NoError(t, err)
			store := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(store, producer["prefix"]+tc.job), []byte("retained output"), 0600))
			payload, err := os.ReadFile(filepath.Join(store, resolved["prefix"]+tc.job))
			require.NoError(t, err, "consumer must fetch the retained producer's actual artifact, not its own attempt")
			assert.Equal(t, "retained output", string(payload))
		})
	}
}

func TestArtifactProducerMetadataRejected(t *testing.T) {
	for _, tc := range []struct {
		name     string
		document string
		result   string
	}{
		{name: "missing", result: "success"},
		{name: "malformed", document: `{`, result: "success"},
		{name: "foreign producer", document: `{"attempt":1,"identity":"12345678-","producer":"other"}`, result: "success"},
		{name: "foreign invocation", document: `{"attempt":1,"identity":"87654321-","producer":"agent"}`, result: "success"},
		{name: "future attempt", document: `{"attempt":3,"identity":"12345678-","producer":"agent"}`, result: "success"},
		{name: "zero attempt", document: `{"attempt":0,"identity":"12345678-","producer":"agent"}`, result: "success"},
		{name: "fractional attempt", document: `{"attempt":1.5,"identity":"12345678-","producer":"agent"}`, result: "success"},
		{name: "string attempt", document: `{"attempt":"1","identity":"12345678-","producer":"agent"}`, result: "success"},
		{name: "unknown field", document: `{"attempt":1,"identity":"12345678-","producer":"agent","prefix":"injected"}`, result: "success"},
		{name: "failed producer rerun", document: `{"attempt":1,"identity":"12345678-","producer":"agent"}`, result: "failure"},
		{name: "skipped producer", document: `{"attempt":1,"identity":"12345678-","producer":"agent"}`, result: "skipped"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metadata := t.TempDir()
			if tc.document != "" {
				require.NoError(t, os.WriteFile(filepath.Join(metadata, "producer.json"), []byte(tc.document), 0600))
			}

			_, err := runArtifactProtocolScript(t, setup.ArtifactProducerResolverScript+"\n"+setup.ArtifactPrefixScript,
				"METADATA_DIRECTORY="+metadata, "PRODUCER_JOB=agent", "PRODUCER_RESULT="+tc.result,
				"INVOCATION_ID=12345678-", "INPUTS_JSON={}", "GITHUB_RUN_ATTEMPT=2")
			require.Error(t, err)
		})
	}
}

func TestArtifactProducerMetadataReplacedOnRerun(t *testing.T) {
	metadata := t.TempDir()
	producer := map[string]string{}
	for _, attempt := range []string{"1", "2"} {
		var err error
		producer, err = runArtifactProtocolScript(t, setup.ArtifactPrefixScript,
			"INPUTS_JSON={}", "GITHUB_RUN_ATTEMPT="+attempt)
		require.NoError(t, err)
		_, err = runArtifactProtocolScript(t, setup.ArtifactProducerScript,
			"METADATA_DIRECTORY="+metadata, "PRODUCER_JOB=agent",
			"PRODUCER_ATTEMPT="+attempt, "INVOCATION_ID="+producer["identity_prefix"])
		require.NoError(t, err)
	}
	resolved, err := runArtifactProtocolScript(t, setup.ArtifactProducerResolverScript+"\n"+setup.ArtifactPrefixScript,
		"METADATA_DIRECTORY="+metadata, "PRODUCER_JOB=agent", "PRODUCER_RESULT=failure",
		"INVOCATION_ID="+producer["identity_prefix"], "INPUTS_JSON={}", "GITHUB_RUN_ATTEMPT=2")
	require.NoError(t, err, "a failed job may still produce valid diagnostic artifacts in its current attempt")
	assert.Equal(t, producer["prefix"], resolved["prefix"])
}

func TestArtifactProducerUnboundChannelOptionality(t *testing.T) {
	for _, continueOnError := range []string{"true", "false", "${{ inputs.tolerate }}"} {
		t.Run(continueOnError, func(t *testing.T) {
			c := NewCompiler()
			job := &Job{Name: "conclusion", Steps: append(generateArtifactPrefixStep(),
				"      - name: Download manifest\n",
				"        uses: "+getActionPin("actions/download-artifact")+"\n",
				"        continue-on-error: "+continueOnError+"\n",
				"        with:\n",
				"          name: "+localArtifactPrefix+"optional-manifest\n",
			)}
			require.NoError(t, c.jobManager.AddJob(job))
			err := c.bindArtifactProducerAttempts(&WorkflowData{On: "workflow_call"})
			if continueOnError != "true" {
				require.ErrorContains(t, err, "has no bound producer", "only literal true proves an unbound channel is optional")
				return
			}
			require.NoError(t, err)
			steps := strings.Join(job.Steps, "")
			assert.Contains(t, steps, "if: ${{ false }}")
			assert.NotContains(t, steps, "artifact-source-")
		})
	}
}

func TestArtifactProducerUsesRepositoryRetention(t *testing.T) {
	steps := strings.Join(NewCompiler().artifactProducerPublishSteps("agent"), "")
	assert.Contains(t, steps, "retention-days: 0\n", "metadata must not impose a one-day limit on retained producer artifacts")
}
