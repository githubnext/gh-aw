package campaign

import (
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var summaryLog = logger.New("campaign:summary")

// campaignSummary provides a compact, human-friendly view of campaign specs
// for the default `gh aw campaign` table output. Full details remain
// available via `--json`.
type campaignSummary struct {
	ID         string `json:"id" console:"header:ID"`
	Name       string `json:"name" console:"header:Name,maxlen:30"`
	State      string `json:"state" console:"header:State"`
	RiskLevel  string `json:"risk_level,omitempty" console:"header:Risk,omitempty"`
	Workflows  string `json:"workflows,omitempty" console:"header:Workflows,omitempty,maxlen:40"`
	Owners     string `json:"owners,omitempty" console:"header:Owners,omitempty,maxlen:30"`
	ConfigPath string `json:"config_path" console:"header:Config Path,maxlen:60"`
}

// buildCampaignSummaries converts full campaign specs into compact summaries
// suitable for human-friendly table output.
func buildCampaignSummaries(specs []CampaignSpec) []campaignSummary {
	summaryLog.Printf("Building campaign summaries for %d campaigns", len(specs))

	summaries := make([]campaignSummary, 0, len(specs))
	for _, spec := range specs {
		summaries = append(summaries, campaignSummary{
			ID:         spec.ID,
			Name:       spec.Name,
			State:      spec.State,
			RiskLevel:  spec.RiskLevel,
			Workflows:  strings.Join(spec.Workflows, ", "),
			Owners:     strings.Join(spec.Owners, ", "),
			ConfigPath: spec.ConfigPath,
		})
	}

	summaryLog.Printf("Created %d campaign summaries", len(summaries))
	return summaries
}
