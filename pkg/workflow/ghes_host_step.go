package workflow

import "github.com/github/gh-aw/pkg/logger"

var ghesHostStepLog = logger.New("workflow:ghes_host_step")

// generateGHESHostConfigurationStep generates a lightweight inline step that exports GH_HOST
// to GITHUB_ENV by stripping the protocol prefix from GITHUB_SERVER_URL. This ensures the
// gh CLI targets the correct GitHub instance in all subsequent steps within the job.
//
// On github.com runners GITHUB_SERVER_URL is "https://github.com", so GH_HOST becomes
// "github.com" (the default — a no-op). On GHES/GHEC runners GITHUB_SERVER_URL is e.g.
// "https://myorg.ghe.com", so GH_HOST becomes "myorg.ghe.com".
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
