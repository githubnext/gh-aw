package cli

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var domainBreakdownLog = logger.New("cli:domain_breakdown")

// DomainBreakdown provides metrics for a single label/domain from outcomes.
type DomainBreakdown struct {
	Label                  string  `json:"label" console:"header:Domain"`
	Attempted              int     `json:"attempted" console:"header:Attempted"`
	Accepted               int     `json:"accepted" console:"header:Accepted"`
	Rejected               int     `json:"rejected" console:"header:Rejected"`
	Pending                int     `json:"pending" console:"header:Pending"`
	TotalObjectiveValue    int     `json:"total_objective_value" console:"header:Total Value"`
	AcceptedObjectiveValue int     `json:"accepted_objective_value" console:"header:Accepted Value"`
	ObjectiveEfficiency    float64 `json:"objective_efficiency,omitempty" console:"header:Efficiency"`
	AcceptanceRate         float64 `json:"acceptance_rate,omitempty" console:"header:Acceptance Rate"`
}

// ComputeDomainBreakdowns aggregates outcome metrics by label/domain.
// Returns a slice of DomainBreakdown sorted by total_objective_value descending.
func ComputeDomainBreakdowns(reports []OutcomeReport) []DomainBreakdown {
	if len(reports) == 0 {
		return []DomainBreakdown{}
	}

	// Map domain label → metrics
	domains := make(map[string]*DomainBreakdown)

	for _, report := range reports {
		// If outcome has objective labels, aggregate by each label
		for _, label := range report.ObjectiveLabels {
			normalizedLabel := strings.ToLower(strings.TrimSpace(label))
			if _, exists := domains[normalizedLabel]; !exists {
				domains[normalizedLabel] = &DomainBreakdown{
					Label: label,
				}
			}

			domain := domains[normalizedLabel]
			domain.Attempted++
			domain.TotalObjectiveValue += report.ObjectiveValue

			switch report.Result {
			case OutcomeAccepted:
				domain.Accepted++
				domain.AcceptedObjectiveValue += report.ObjectiveValue
			case OutcomeRejected:
				domain.Rejected++
			case OutcomePending:
				domain.Pending++
			}
		}

		// If outcome has NO objective labels, create "unmapped" entry
		if len(report.ObjectiveLabels) == 0 && report.ObjectiveValue == 0 {
			if _, exists := domains["unmapped"]; !exists {
				domains["unmapped"] = &DomainBreakdown{
					Label: "unmapped",
				}
			}
			domain := domains["unmapped"]
			domain.Attempted++

			switch report.Result {
			case OutcomeAccepted:
				domain.Accepted++
			case OutcomeRejected:
				domain.Rejected++
			case OutcomePending:
				domain.Pending++
			}
		}
	}

	// Compute efficiency metrics for each domain
	result := make([]DomainBreakdown, 0, len(domains))
	for _, domain := range domains {
		if domain.Attempted > 0 {
			domain.AcceptanceRate = float64(domain.Accepted) / float64(domain.Attempted)
		}
		if domain.TotalObjectiveValue > 0 {
			domain.ObjectiveEfficiency = float64(domain.AcceptedObjectiveValue) / float64(domain.TotalObjectiveValue)
		}
		result = append(result, *domain)
	}

	// Sort by total_objective_value descending
	slices.SortFunc(result, func(a, b DomainBreakdown) int {
		if a.TotalObjectiveValue != b.TotalObjectiveValue {
			return cmp.Compare(b.TotalObjectiveValue, a.TotalObjectiveValue)
		}
		return strings.Compare(a.Label, b.Label)
	})

	domainBreakdownLog.Printf("Computed domain breakdowns: domains=%d, total_attempted=%d", len(result), countTotalAttempted(result))
	return result
}

func countTotalAttempted(breakdowns []DomainBreakdown) int {
	total := 0
	for _, d := range breakdowns {
		total += d.Attempted
	}
	return total
}

// DomainInsight provides a human-readable assessment of a domain's performance.
type DomainInsight struct {
	Label      string
	Status     string // "excellent", "good", "fair", "poor", "new"
	Efficiency float64
	Message    string
	Suggestion string
}

// AnalyzeDomainPerformance provides strategic insights about domain efficiency.
func AnalyzeDomainPerformance(breakdown DomainBreakdown) DomainInsight {
	insight := DomainInsight{
		Label:      breakdown.Label,
		Efficiency: breakdown.ObjectiveEfficiency,
	}

	if breakdown.Attempted == 0 {
		insight.Status = "new"
		insight.Message = "No outcomes yet"
		return insight
	}

	switch {
	case breakdown.ObjectiveEfficiency >= 0.90:
		insight.Status = "excellent"
		insight.Message = fmt.Sprintf("✅ %s: %.0f%% efficiency | %d/%d outcomes accepted | %d value delivered",
			breakdown.Label,
			breakdown.ObjectiveEfficiency*100,
			breakdown.Accepted,
			breakdown.Attempted,
			breakdown.AcceptedObjectiveValue)
		insight.Suggestion = "Keep current strategy working well"

	case breakdown.ObjectiveEfficiency >= 0.75:
		insight.Status = "good"
		insight.Message = fmt.Sprintf("✅ %s: %.0f%% efficiency | %d/%d outcomes accepted | %d value delivered",
			breakdown.Label,
			breakdown.ObjectiveEfficiency*100,
			breakdown.Accepted,
			breakdown.Attempted,
			breakdown.AcceptedObjectiveValue)
		insight.Suggestion = "Good progress; monitor for regressions"

	case breakdown.ObjectiveEfficiency >= 0.50:
		insight.Status = "fair"
		insight.Message = fmt.Sprintf("⚠️ %s: %.0f%% efficiency | %d/%d outcomes accepted | %d value delivered",
			breakdown.Label,
			breakdown.ObjectiveEfficiency*100,
			breakdown.Accepted,
			breakdown.Attempted,
			breakdown.AcceptedObjectiveValue)
		insight.Suggestion = "Consider reviewing agent strategy or adding human review"

	default:
		insight.Status = "poor"
		insight.Message = fmt.Sprintf("🔴 %s: %.0f%% efficiency | %d/%d outcomes accepted | %d value delivered",
			breakdown.Label,
			breakdown.ObjectiveEfficiency*100,
			breakdown.Accepted,
			breakdown.Attempted,
			breakdown.AcceptedObjectiveValue)
		insight.Suggestion = "Major issues; investigate root cause or pause automation"
	}

	return insight
}
