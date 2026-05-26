package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var labelDispatchRemoteWorkflowLog = logger.New("workflow:label_dispatch_remote_workflow")

// GenerateLabelDispatchRemoteWorkflows generates .remote.yml bridge workflows for workflows
// that declare on.label_dispatch. The generated files are meant to be copied to allowed repos.
func GenerateLabelDispatchRemoteWorkflows(workflowDataList []*WorkflowData, workflowDir string, centralRepo string) error {
	for _, wd := range workflowDataList {
		if wd == nil || strings.TrimSpace(wd.WorkflowID) == "" {
			continue
		}
		path := filepath.Join(workflowDir, wd.WorkflowID+".remote.yml")
		if wd.LabelDispatch == nil || strings.TrimSpace(wd.LabelDispatch.Label) == "" {
			if err := removeIfExists(path); err != nil {
				return fmt.Errorf("failed to delete label-dispatch remote workflow: %w", err)
			}
			continue
		}
		content := buildLabelDispatchRemoteWorkflowYAML(wd, centralRepo)
		if err := os.WriteFile(path, []byte(content), constants.FilePermPublic); err != nil {
			return fmt.Errorf("failed to write label-dispatch remote workflow: %w", err)
		}
		labelDispatchRemoteWorkflowLog.Printf("Wrote label-dispatch remote workflow: %s", path)
	}
	return nil
}

func buildLabelDispatchRemoteWorkflowYAML(wd *WorkflowData, centralRepo string) string {
	label := strings.TrimSpace(wd.LabelDispatch.Label)
	eventType := buildLabelDispatchEventType(wd.WorkflowID)
	jobIf := "github.event.label.name == " + githubActionsStringLiteral(label)
	if repoExpr := buildAllowedRepoExpression("github.repository", wd.LabelDispatch.AllowedRepos); repoExpr != "" {
		jobIf += " && (" + repoExpr + ")"
	}

	var b strings.Builder
	b.WriteString("name: \"" + wd.Name + " Label Dispatch Bridge\"\n")
	b.WriteString("on:\n")
	b.WriteString("  pull_request:\n")
	b.WriteString("    types: [labeled]\n")
	b.WriteString("permissions:\n")
	b.WriteString("  contents: write\n")
	b.WriteString("jobs:\n")
	b.WriteString("  dispatch:\n")
	b.WriteString("    if: ${{ " + jobIf + " }}\n")
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    steps:\n")
	b.WriteString("      - name: Dispatch central workflow\n")
	b.WriteString("        uses: actions/github-script@v7\n")
	b.WriteString("        with:\n")
	b.WriteString("          github-token: ${{ secrets.COPILOT_GITHUB_TOKEN }}\n")
	b.WriteString("          script: |\n")
	b.WriteString("            const [owner, repo] = " + githubActionsStringLiteral(strings.TrimSpace(centralRepo)) + ".split('/')\n")
	b.WriteString("            await github.rest.repos.createDispatchEvent({\n")
	b.WriteString("              owner,\n")
	b.WriteString("              repo,\n")
	b.WriteString("              event_type: " + githubActionsStringLiteral(eventType) + ",\n")
	b.WriteString("              client_payload: {\n")
	b.WriteString("                target_repo: context.repo.owner + '/' + context.repo.repo,\n")
	b.WriteString("                pr_number: context.payload.pull_request.number,\n")
	b.WriteString("                head_sha: context.payload.pull_request.head.sha,\n")
	b.WriteString("                base_sha: context.payload.pull_request.base.sha,\n")
	b.WriteString("                trigger_label: context.payload.label.name,\n")
	b.WriteString("                delivery_id: process.env.GITHUB_RUN_ID,\n")
	b.WriteString("                actor: context.actor\n")
	b.WriteString("              }\n")
	b.WriteString("            });\n")
	return b.String()
}

func buildAllowedRepoExpression(variable string, patterns []string) string {
	var checks []string
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if pattern == "*" {
			return "true"
		}
		if strings.HasSuffix(pattern, "/*") {
			prefix := strings.TrimSuffix(pattern, "/*") + "/"
			checks = append(checks, "startsWith("+variable+", "+githubActionsStringLiteral(prefix)+")")
			continue
		}
		checks = append(checks, variable+" == "+githubActionsStringLiteral(pattern))
	}
	return strings.Join(checks, " || ")
}
