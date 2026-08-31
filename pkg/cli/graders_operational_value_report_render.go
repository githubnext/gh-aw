package cli

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"
)

type operationalValueReportArtifactPaths struct {
	JSON     string `json:"json"`
	SVG      string `json:"svg"`
	Markdown string `json:"markdown"`
}

func writeOperationalValueReportArtifacts(report operationalValueReport, outputDir string) (operationalValueReportArtifactPaths, error) {
	if err := os.MkdirAll(outputDir, constants.DirPermPublic); err != nil {
		return operationalValueReportArtifactPaths{}, fmt.Errorf("cannot create operational-value report directory: %w", err)
	}
	base := report.WorkflowID + "-operational-value"
	paths := operationalValueReportArtifactPaths{
		JSON: filepath.Join(outputDir, base+".json"), SVG: filepath.Join(outputDir, base+".svg"), Markdown: filepath.Join(outputDir, base+".md"),
	}
	jsonData, err := marshalIndentJSONOrWrap(report, "operational-value report")
	if err != nil {
		return operationalValueReportArtifactPaths{}, err
	}
	svgData := renderOperationalValueReportSVG(report)
	markdownData := renderOperationalValueReportMarkdown(report, filepath.Base(paths.JSON), filepath.Base(paths.SVG))
	for path, data := range map[string][]byte{paths.JSON: jsonData, paths.SVG: svgData, paths.Markdown: markdownData} {
		if err := writeFileAtomically(path, data); err != nil {
			return operationalValueReportArtifactPaths{}, fmt.Errorf("cannot write operational-value report %s: %w", path, err)
		}
	}
	return paths, nil
}

func renderOperationalValueReportSVG(report operationalValueReport) []byte {
	const left, right, top, bottom = 120.0, 1160.0, 130.0, 500.0
	start, _ := time.Parse(time.RFC3339, report.Window.StartAt)
	end, _ := time.Parse(time.RFC3339, report.Window.EndAt)
	span := end.Sub(start).Seconds()
	if span <= 0 {
		span = 1
	}
	xFor := func(value time.Time) float64 {
		position := value.Sub(start).Seconds() / span
		return left + max(0, min(1, position))*(right-left)
	}
	yFor := func(value float64) float64 { return bottom - value*(bottom-top) }
	var svg strings.Builder
	svg.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1280 720" role="img" aria-labelledby="title description">` + "\n")
	fmt.Fprintf(&svg, "<title id=\"title\">%s operational value over time</title>\n", html.EscapeString(report.WorkflowName))
	fmt.Fprintf(&svg, "<desc id=\"description\">Per-run operational attainment and deduplicated weekly averages under evaluator %s.</desc>\n", html.EscapeString(shortOperationalValueDigest(report.Evaluator.SHA256)))
	svg.WriteString(`<style>:root{--fg:#24292f;--muted:#57606a;--bg:#fff;--subtle:#f6f8fa;--border:#d0d7de;--point:#0969da;--weekly:#1a7f37;--baseline:#8250df;--error:#cf222e}@media(prefers-color-scheme:dark){:root{--fg:#f0f6fc;--muted:#8c959f;--bg:#0d1117;--subtle:#161b22;--border:#30363d;--point:#58a6ff;--weekly:#3fb950;--baseline:#a371f7;--error:#f85149}}text{font-family:ui-sans-serif,system-ui,sans-serif;fill:var(--fg);letter-spacing:0}.title{font-size:30px;font-weight:700}.subtitle{font-size:17px;fill:var(--muted)}.axis{font-size:15px;fill:var(--muted)}.grid{stroke:var(--border);stroke-width:1}.weekly{fill:none;stroke:var(--weekly);stroke-width:4;stroke-linejoin:round;stroke-linecap:round}.point{fill:var(--point);stroke:var(--bg);stroke-width:2}.baseline{stroke:var(--baseline);stroke-width:3;stroke-dasharray:7 6}.error{stroke:var(--error);stroke-width:3}</style>` + "\n")
	svg.WriteString(`<rect width="1280" height="720" fill="var(--bg)"/>` + "\n")
	fmt.Fprintf(&svg, "<text x=\"120\" y=\"52\" class=\"title\">%s operational value</text>\n", html.EscapeString(report.WorkflowName))
	fmt.Fprintf(&svg, "<text x=\"120\" y=\"82\" class=\"subtitle\">%d runs, %d numeric values, evaluator %s</text>\n", report.Coverage.RunCount, report.Coverage.NumericCount, html.EscapeString(shortOperationalValueDigest(report.Evaluator.SHA256)))
	for index := 0; index <= 4; index++ {
		value := float64(index) / 4
		y := yFor(value)
		fmt.Fprintf(&svg, "<line class=\"grid\" x1=\"%.1f\" y1=\"%.1f\" x2=\"%.1f\" y2=\"%.1f\"/><text x=\"102\" y=\"%.1f\" text-anchor=\"end\" class=\"axis\">%.2g</text>\n", left, y, right, y, y+5, value)
	}
	for index := 0; index <= 6; index++ {
		at := start.Add(time.Duration(float64(end.Sub(start)) * float64(index) / 6))
		x := xFor(at)
		fmt.Fprintf(&svg, "<line class=\"grid\" x1=\"%.1f\" y1=\"%.1f\" x2=\"%.1f\" y2=\"%.1f\"/><text x=\"%.1f\" y=\"530\" text-anchor=\"middle\" class=\"axis\">%s</text>\n", x, top, x, bottom, x, at.Format("Jan 02"))
	}
	if report.Baseline.Value != nil {
		y := yFor(*report.Baseline.Value)
		fmt.Fprintf(&svg, "<line class=\"baseline\" x1=\"%.1f\" y1=\"%.1f\" x2=\"%.1f\" y2=\"%.1f\"/><text x=\"1152\" y=\"%.1f\" text-anchor=\"end\" class=\"axis\">Baseline %.3f</text>\n", left, y, right, y, y-9, *report.Baseline.Value)
	}
	weeklyPoints := make([]string, 0, len(report.Weekly))
	for _, week := range report.Weekly {
		if week.Mean == nil {
			continue
		}
		weekStart, _ := time.Parse(time.RFC3339, week.WeekStart)
		weekEnd, _ := time.Parse(time.RFC3339, week.WeekEnd)
		x := xFor(weekStart.Add(weekEnd.Sub(weekStart) / 2))
		weeklyPoints = append(weeklyPoints, fmt.Sprintf("%.1f,%.1f", x, yFor(*week.Mean)))
	}
	if len(weeklyPoints) > 1 {
		fmt.Fprintf(&svg, "<polyline class=\"weekly\" points=\"%s\"/>\n", strings.Join(weeklyPoints, " "))
	}
	for _, observation := range report.Observations {
		x := xFor(observation.Run.CreatedAt)
		if observation.Value == nil {
			fmt.Fprintf(&svg, "<line class=\"error\" x1=\"%.1f\" y1=\"506\" x2=\"%.1f\" y2=\"518\"/>\n", x, x)
			continue
		}
		fmt.Fprintf(&svg, "<circle class=\"point\" cx=\"%.1f\" cy=\"%.1f\" r=\"5\"><title>Run %s: %.4f</title></circle>\n", x, yFor(*observation.Value), html.EscapeString(observation.Run.ID), *observation.Value)
	}
	svg.WriteString(`<circle class="point" cx="138" cy="585" r="5"/><text x="154" y="591" class="axis">Per-run value</text><line class="weekly" x1="318" y1="585" x2="354" y2="585"/><text x="366" y="591" class="axis">Weekly mean, one latest value per opportunity</text><line class="error" x1="760" y1="578" x2="760" y2="592"/><text x="774" y="591" class="axis">Missing or error</text>` + "\n")
	fmt.Fprintf(&svg, "<text x=\"120\" y=\"650\" class=\"subtitle\">Higher is better. Repeated opportunity keys are shown as points but deduplicated within weekly means.</text>\n")
	svg.WriteString("</svg>\n")
	return []byte(svg.String())
}

func renderOperationalValueReportMarkdown(report operationalValueReport, jsonName, svgName string) []byte {
	var markdown strings.Builder
	fmt.Fprintf(&markdown, "# %s operational value\n\n", sanitizeOperationalValueMarkdown(report.WorkflowName))
	fmt.Fprintf(&markdown, "![%s operational value timeline](%s)\n\n", sanitizeOperationalValueMarkdown(report.WorkflowName), svgName)
	markdown.WriteString("## Summary\n\n")
	fmt.Fprintf(&markdown, "- **Operational value:** %s\n", sanitizeOperationalValueMarkdown(report.OperationalValue))
	fmt.Fprintf(&markdown, "- **History:** %s through %s\n", report.Window.StartAt, report.Window.EndAt)
	fmt.Fprintf(&markdown, "- **Coverage:** %d of %d runs produced numeric values; %d unavailable; %d errors\n", report.Coverage.NumericCount, report.Coverage.RunCount, report.Coverage.UnavailableCount, report.Coverage.ErrorCount)
	fmt.Fprintf(&markdown, "- **Current evaluator:** `%s`\n", report.Evaluator.SHA256)
	fmt.Fprintf(&markdown, "- **Weekly cache:** %d hits; %d runs evaluated in this invocation\n", report.Coverage.WeeklyCacheHits, report.Coverage.EvaluatedCount)
	if report.Summary.Latest != nil {
		fmt.Fprintf(&markdown, "- **Latest value:** %s", formatOperationalValue(*report.Summary.Latest))
		if report.Summary.Change != nil {
			fmt.Fprintf(&markdown, " (%s from the first numeric observation)", formatSignedOperationalValue(*report.Summary.Change))
		}
		markdown.WriteString("\n")
	}
	if report.Baseline.Value != nil {
		fmt.Fprintf(&markdown, "- **Frozen baseline:** %s", formatOperationalValue(*report.Baseline.Value))
		if report.Summary.LatestDeltaFromBaseline != nil {
			fmt.Fprintf(&markdown, " (latest delta %s)", formatSignedOperationalValue(*report.Summary.LatestDeltaFromBaseline))
		}
		markdown.WriteString("\n")
	}
	fmt.Fprintf(&markdown, "- **Structured report:** [%s](%s)\n\n", jsonName, jsonName)

	markdown.WriteString("## Weekly History\n\n")
	markdown.WriteString("| Week | Runs | Distinct opportunities | Mean | Range |\n|---|---:|---:|---:|---:|\n")
	for _, week := range report.Weekly {
		mean, valueRange := "missing", "missing"
		if week.Mean != nil {
			mean = formatOperationalValue(*week.Mean)
			valueRange = formatOperationalValue(*week.Minimum) + "-" + formatOperationalValue(*week.Maximum)
		}
		fmt.Fprintf(&markdown, "| %s | %d | %d | %s | %s |\n", strings.TrimSuffix(week.WeekStart, "T00:00:00Z"), week.RunCount, week.DistinctOpportunityCount, mean, valueRange)
	}

	markdown.WriteString("\n## Frozen Contract\n\n")
	fmt.Fprintf(&markdown, "- **Opportunity:** %s\n", sanitizeOperationalValueMarkdown(report.EvaluatorDefinition().Evidence.Opportunity))
	fmt.Fprintf(&markdown, "- **Assignment:** %s\n", sanitizeOperationalValueMarkdown(report.EvaluatorDefinition().Evidence.Assignment))
	fmt.Fprintf(&markdown, "- **Accepted evidence:** %s\n", sanitizeOperationalValueMarkdown(report.EvaluatorDefinition().Evidence.Accepted))
	fmt.Fprintf(&markdown, "- **Collection:** %s\n", sanitizeOperationalValueMarkdown(report.EvaluatorDefinition().Evidence.Collection))
	fmt.Fprintf(&markdown, "- **Maturation:** %s\n", sanitizeOperationalValueMarkdown(report.EvaluatorDefinition().Evidence.Maturation))
	fmt.Fprintf(&markdown, "- **Zero rule:** %s\n", sanitizeOperationalValueMarkdown(report.EvaluatorDefinition().Evidence.ZeroRule))
	fmt.Fprintf(&markdown, "- **Missing rule:** %s\n", sanitizeOperationalValueMarkdown(report.EvaluatorDefinition().Evidence.MissingRule))
	fmt.Fprintf(&markdown, "- **Primary metric:** `%s` = %s\n", report.EvaluatorDefinition().PrimaryMetric.ID, sanitizeOperationalValueMarkdown(report.EvaluatorDefinition().PrimaryMetric.Formula))

	markdown.WriteString("\n## Coverage Notes\n\n")
	fmt.Fprintf(&markdown, "%d distinct opportunity keys were observed; %d additional numeric observations repeated a key. Repeated observations remain visible in the plot but are not treated as independent within a weekly mean.\n\n", report.Coverage.DistinctOpportunityCount, report.Coverage.DuplicateOpportunityCount)
	if report.Coverage.ErrorCount > 0 {
		markdown.WriteString("### Evaluation Errors\n\n")
		for _, observation := range report.Observations {
			if observation.Status == "error" {
				fmt.Fprintf(&markdown, "- Run `%s`: %s\n", observation.Run.ID, sanitizeOperationalValueMarkdown(observation.Message))
			}
		}
		markdown.WriteString("\n")
	}
	markdown.WriteString("## Interpretation\n\n")
	markdown.WriteString(report.Caveat + " Pre-grader runs have no archived case or event payload, so their cases are reconstructed only when the evaluator supports assignment from the run subject. Missing evidence is never treated as zero.\n")
	return []byte(markdown.String())
}

func (report operationalValueReport) EvaluatorDefinition() operationalValueReportDefinition {
	var definition operationalValueReportDefinition
	_ = json.Unmarshal(report.Evaluator.Definition, &definition)
	return definition
}

func sanitizeOperationalValueMarkdown(value string) string {
	return strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ", "|", "\\|").Replace(value))
}

func formatOperationalValue(value float64) string {
	return strconv.FormatFloat(math.Round(value*10000)/10000, 'f', -1, 64)
}

func formatSignedOperationalValue(value float64) string {
	if value > 0 {
		return "+" + formatOperationalValue(value)
	}
	return formatOperationalValue(value)
}

func shortOperationalValueDigest(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}
