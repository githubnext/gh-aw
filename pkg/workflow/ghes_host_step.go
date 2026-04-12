package workflow

import "github.com/github/gh-aw/pkg/logger"

var ghesHostStepLog = logger.New("workflow:ghes_host_step")

// GHESHostStepID is the step ID for the GH_HOST configuration step.
const GHESHostStepID = "ghes-host-config"

// GHESHostOutputExpr is the GitHub Actions expression to reference
// the GH_HOST value from the ghes-host-config step output.
const GHESHostOutputExpr = "${{ steps." + GHESHostStepID + ".outputs.GH_HOST }}"

// generateGHESHostConfigurationStep generates a lightweight inline step that exports GH_HOST
// to GITHUB_OUTPUT by stripping the protocol prefix from GITHUB_SERVER_URL. Subsequent steps
// that need GH_HOST must include it in their step-level env block via GHESHostOutputExpr.
//
// On github.com runners GITHUB_SERVER_URL is "https://github.com", so GH_HOST becomes
// "github.com" (the default — a no-op). On GHES/GHEC runners GITHUB_SERVER_URL is e.g.
// "https://myorg.ghe.com", so GH_HOST becomes "myorg.ghe.com".
//
// The step writes to GITHUB_OUTPUT (not GITHUB_ENV) to avoid the github-env security
// finding flagged by zizmor. Although GITHUB_SERVER_URL is framework-controlled and not
// attacker-influenced, using GITHUB_OUTPUT with step-scoped env propagation is the
// recommended best practice.
//
// This step has zero external dependencies (no setup scripts) and can be safely prepended
// to any job. It is used for custom frontmatter jobs and the safe-outputs job where the
// full configure_gh_for_ghe.sh script is not available.
func generateGHESHostConfigurationStep() string {
	ghesHostStepLog.Print("Generating inline GH_HOST configuration step for GHES compatibility")

	return `      - name: Configure GH_HOST for enterprise compatibility
        id: ghes-host-config
        shell: bash
        run: |
          # Derive GH_HOST from GITHUB_SERVER_URL so the gh CLI targets the correct
          # GitHub instance (GHES/GHEC). On github.com this is a harmless no-op.
          GH_HOST="${GITHUB_SERVER_URL#https://}"
          GH_HOST="${GH_HOST#http://}"
          echo "GH_HOST=${GH_HOST}" >> "$GITHUB_OUTPUT"
`
}

// injectGHESHostEnv adds the GH_HOST environment variable to a WorkflowStep so that
// the gh CLI targets the correct GitHub instance. This is needed because the
// ghes-host-config step writes to GITHUB_OUTPUT (not GITHUB_ENV), so each step
// that may use the gh CLI must explicitly receive GH_HOST via its env block.
func injectGHESHostEnv(step *WorkflowStep) {
	if step.Env == nil {
		step.Env = make(map[string]string)
	}
	step.Env["GH_HOST"] = GHESHostOutputExpr
}
