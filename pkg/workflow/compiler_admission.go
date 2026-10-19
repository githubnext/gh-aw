package workflow

import (
	"fmt"
	"net/url"
	"regexp"
)

// AdmissionConfig selects the service trusted with the ephemeral GitHub Actions token.
type AdmissionConfig struct {
	URL      string `json:"url"`
	OnOutage string `json:"on_outage,omitempty"`
}

var admissionURLPattern = regexp.MustCompile(`^https://[^\s\\?#${}]+$`)

func validateAdmissionConfig(config *AdmissionConfig) error {
	if config == nil {
		return nil
	}
	if config.OnOutage != "" && config.OnOutage != "ignore" && config.OnOutage != "block" {
		return fmt.Errorf("%s admission.on_outage must be ignore or block; omit it to continue execution during service outages", RepoConfigFileName)
	}
	endpoint, err := url.Parse(config.URL)
	if err != nil || !admissionURLPattern.MatchString(config.URL) || endpoint.Hostname() == "" || endpoint.User != nil {
		return fmt.Errorf("%s admission.url must be an absolute HTTPS URL without credentials, query parameters, fragments, or expressions; configure a trusted admission service endpoint", RepoConfigFileName)
	}
	return nil
}

func (c *Compiler) buildAdmissionGateStep(data *WorkflowData) ([]string, error) {
	config, err := c.loadRepoConfig()
	if err != nil {
		return nil, err
	}
	if config == nil || config.Admission == nil {
		return nil, nil
	}
	onOutage := config.Admission.OnOutage
	if onOutage == "" {
		onOutage = "ignore"
	}
	return []string{
		"      - name: Check workflow admission\n",
		"        id: admission\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		fmt.Sprintf("          GH_AW_ADMISSION_URL: %q\n", config.Admission.URL),
		fmt.Sprintf("          GH_AW_ADMISSION_ON_OUTAGE: %q\n", onOutage),
		"          GH_AW_ADMISSION_TOKEN: ${{ github.token }}\n",
		fmt.Sprintf("          GH_AW_WORKFLOW_ID: %q\n", data.WorkflowID),
		"        with:\n",
		"          github-token: ${{ github.token }}\n",
		"          script: |\n",
		generateGitHubScriptWithRequire("check_admission.cjs"),
	}, nil
}
