package cli

import "math"

func (s *OutcomeSummary) addAIC(aic float64) {
	if aic < 0 || math.IsNaN(aic) || math.IsInf(aic, 0) {
		outcomeEvalLog.Printf("Cannot compute outcome efficiency from invalid AIC: %v", aic)
		return
	}
	if s.Accepted == 0 || aic == 0 {
		return
	}
	s.CostPerAcceptedOutcome = aic / float64(s.Accepted)
}

func loadOutcomeAIC(runDir string, summary *RunSummary) (float64, error) {
	if summary != nil && summary.TokenUsage != nil && summary.TokenUsage.TotalAIC > 0 {
		return summary.TokenUsage.TotalAIC, nil
	}
	usage, err := analyzeTokenUsageAICOnly(runDir, false)
	if err != nil || usage == nil {
		return 0, err
	}
	return usage.TotalAIC, nil
}
