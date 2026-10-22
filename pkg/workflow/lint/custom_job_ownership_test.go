package lint

import (
	"context"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomJobDetectionActionsWithoutAgent(t *testing.T) {
	workflow, err := actions.Parse([]byte("jobs:\n  detection:\n    runs-on: ubuntu-latest\n    steps: [{run: echo detection}]\n  notify:\n    needs: detection\n    if: always()\n    uses: ./.github/workflows/notify.yml\n"))
	require.NoError(t, err)
	diagnostics, err := CustomJobDetectionRule().Check(context.Background(), Input{Workflow: workflow})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)
}

func TestCustomJobDetectionMaintenanceOwnership(t *testing.T) {
	for _, name := range []constants.JobName{
		constants.PushEvalsStateJobName, constants.PushLedgerChangesJobName,
		constants.PushExperimentsStateJobName, constants.PushRepoMemoryJobName,
		constants.UpdateCacheMemoryJobName, constants.UpdateDriveMemoryJobName,
	} {
		t.Run(string(name), func(t *testing.T) {
			jobs := "  " + string(name) + ":\n    needs: agent\n    if: always()\n    uses: ./.github/workflows/persist.yml\n"
			document, err := aw.Parse("---\njobs:\n" + jobs + "---\nPrompt.\n")
			require.NoError(t, err)
			workflow, err := actions.Parse([]byte("jobs:\n  agent:\n    runs-on: ubuntu-latest\n    steps: [{run: echo agent}]\n  detection:\n    needs: agent\n    runs-on: ubuntu-latest\n    steps: [{run: echo detection}]\n" + jobs))
			require.NoError(t, err)
			rule := CustomJobDetectionRule()
			standalone, err := rule.Check(context.Background(), Input{Workflow: workflow})
			require.NoError(t, err)
			assert.Empty(t, standalone, "standalone locks recognize generated maintenance jobs")
			for _, input := range []Input{{Document: document}, {Document: document, Workflow: workflow}} {
				authored, err := rule.Check(context.Background(), input)
				require.NoError(t, err)
				require.Len(t, authored, 1, "authored jobs must not inherit generated ownership by name")
				assert.Equal(t, "jobs."+string(name)+".needs", authored[0].Path)
			}
		})
	}
}

func TestCustomJobDetectionAuthoredReservedName(t *testing.T) {
	document, err := aw.Parse("---\njobs:\n  evals:\n    needs: agent\n    uses: ./.github/workflows/report.yml\n---\n")
	require.NoError(t, err)
	diagnostics, err := CustomJobDetectionRule().Check(context.Background(), Input{Document: document})
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)
	assert.Equal(t, "jobs.evals.needs", diagnostics[0].Path)
}
