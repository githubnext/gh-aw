package campaign

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCampaignCommand_HasTriggerSubcommand(t *testing.T) {
	cmd := NewCommand()

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() == "trigger" {
			found = true
			break
		}
	}

	require.True(t, found, "expected campaign command to include 'trigger' subcommand")
}

func TestTriggerCampaignGenerator_BuildsGHArgs(t *testing.T) {
	oldRunGH := runGH
	t.Cleanup(func() { runGH = oldRunGH })

	calls := [][]string{}
	runGH = func(ctx context.Context, args ...string) ([]byte, error) {
		calls = append(calls, args)
		return []byte("ok"), nil
	}

	err := triggerCampaignGenerator(context.Background(), "123", "owner/repo", "create-agentic-campaign")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(calls), 2, "expected gh to be invoked at least twice (--version + issue edit)")
	require.Equal(t, []string{"--version"}, calls[0])
	require.Equal(t, []string{"issue", "edit", "123", "--add-label", "create-agentic-campaign", "--repo", "owner/repo"}, calls[1])
}

func TestTriggerCampaignGenerator_RejectsNonNumericIssue(t *testing.T) {
	oldRunGH := runGH
	t.Cleanup(func() { runGH = oldRunGH })

	// Ensure we don't call gh at all.
	runGH = func(ctx context.Context, args ...string) ([]byte, error) {
		t.Fatalf("runGH should not be called for invalid issue numbers")
		return nil, nil
	}

	err := triggerCampaignGenerator(context.Background(), "abc", "", "create-agentic-campaign")
	require.Error(t, err)
}
