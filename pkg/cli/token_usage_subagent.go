package cli

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"
)

var subagentDispatchPattern = regexp.MustCompile(`([A-Za-z0-9][A-Za-z0-9._-]*)\(([A-Za-z0-9][A-Za-z0-9._:-]*)\)`)

func augmentSubagentModelAttribution(runDir string, summary *TokenUsageSummary) {
	if summary == nil {
		return
	}

	requests := extractSubagentModelRequests(runDir)
	if len(requests) == 0 {
		return
	}
	addTokenUsageWarning(summary, subagentStdioWarning)

	actuals := make([]SubagentModelActual, 0, len(summary.ByModel))
	observedModels := make(map[string]string, len(summary.ByModel))
	for model, usage := range summary.ByModel {
		if usage == nil || model == "" {
			continue
		}
		actuals = append(actuals, SubagentModelActual{
			Model:    model,
			Provider: usage.Provider,
			Requests: usage.Requests,
		})
		observedModels[model] = usage.Provider
	}
	slices.SortStableFunc(actuals, func(a, b SubagentModelActual) int {
		if a.Requests != b.Requests {
			if a.Requests > b.Requests {
				return -1
			}
			return 1
		}
		switch {
		case a.Model < b.Model:
			return -1
		case a.Model > b.Model:
			return 1
		default:
			return 0
		}
	})
	summary.SubagentModelActuals = actuals

	var fallbackEffectiveModel string
	if len(observedModels) == 1 {
		for model := range observedModels {
			fallbackEffectiveModel = model
		}
	}

	requestRows := make([]SubagentModelRequest, 0, len(requests))
	mismatchCount := 0
	for _, row := range requests {
		if _, ok := observedModels[row.RequestedModel]; ok {
			row.EffectiveModel = row.RequestedModel
		} else {
			row.EffectiveModel = fallbackEffectiveModel
			if len(observedModels) == 0 {
				row.ReasonCode = modelMismatchReasonTokenUsageMissing
			} else {
				row.ReasonCode = modelMismatchReasonModelNotObserved
			}
			mismatchCount += row.InvocationCount
		}
		requestRows = append(requestRows, row)
	}
	summary.SubagentModelRequests = requestRows
	summary.MismatchCount = mismatchCount
}

func addTokenUsageWarning(summary *TokenUsageSummary, warning string) {
	if summary == nil || warning == "" {
		return
	}
	if slices.Contains(summary.Warnings, warning) {
		return
	}
	summary.Warnings = append(summary.Warnings, warning)
}

func extractSubagentModelRequests(runDir string) []SubagentModelRequest {
	agentStdioPath := findAgentStdioFile(runDir)
	if agentStdioPath == "" {
		return nil
	}

	file, err := os.Open(agentStdioPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	type key struct {
		agent string
		model string
	}
	counts := make(map[key]int)

	reader := bufio.NewReader(file)
	for {
		line, readErr := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			matches := subagentDispatchPattern.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				if len(m) < 3 {
					continue
				}
				agentName := strings.TrimSpace(m[1])
				requestedModel := strings.TrimSpace(m[2])
				if agentName == "" || requestedModel == "" {
					continue
				}
				counts[key{agent: agentName, model: requestedModel}]++
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil
		}
	}

	rows := make([]SubagentModelRequest, 0, len(counts))
	for k, n := range counts {
		rows = append(rows, SubagentModelRequest{
			AgentName:       k.agent,
			RequestedModel:  k.model,
			InvocationCount: n,
		})
	}
	slices.SortStableFunc(rows, func(a, b SubagentModelRequest) int {
		if a.AgentName != b.AgentName {
			if a.AgentName < b.AgentName {
				return -1
			}
			return 1
		}
		switch {
		case a.RequestedModel < b.RequestedModel:
			return -1
		case a.RequestedModel > b.RequestedModel:
			return 1
		default:
			return 0
		}
	})
	return rows
}
