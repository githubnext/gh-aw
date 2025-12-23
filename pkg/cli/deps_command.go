package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/spf13/cobra"
)

// NewDepsCommand creates the deps command with subcommands for dependency management
func NewDepsCommand() *cobra.Command {
	depsCmd := &cobra.Command{
		Use:   "deps",
		Short: "Analyze and manage project dependencies",
		Long: `Analyze and manage project dependencies including health metrics, outdated packages, and security vulnerabilities.

This command provides comprehensive dependency analysis to help maintain a healthy,
secure, and up-to-date dependency tree.

Examples:
  gh aw deps health      # Show dependency health metrics and recommendations
  gh aw deps outdated    # List outdated dependencies with available updates
  gh aw deps security    # Check for security vulnerabilities
  gh aw deps report      # Generate comprehensive dependency report`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default behavior: show health metrics
			return runDepsHealth(cmd, args)
		},
	}

	// Add subcommands
	depsCmd.AddCommand(newDepsHealthCommand())
	depsCmd.AddCommand(newDepsOutdatedCommand())
	depsCmd.AddCommand(newDepsSecurityCommand())
	depsCmd.AddCommand(newDepsReportCommand())

	return depsCmd
}

func newDepsHealthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Show dependency health metrics and analysis",
		Long: `Analyze dependency health including:
- Total dependency count (direct and indirect)
- v0.x (unstable) dependency ratio  
- Dependency breakdown by major version
- Health assessment against target thresholds

This helps track progress toward reducing unstable dependency exposure
from the current baseline to a target of <30% v0.x dependencies.`,
		RunE: runDepsHealth,
	}
}

func newDepsOutdatedCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "List outdated dependencies with available updates",
		Long: `List direct dependencies that have newer versions available.

Queries the Go proxy API to check each dependency for updates and displays:
- Current version
- Latest available version
- Age of the latest version
- Special markers for v0.x (unstable) dependencies

This helps identify opportunities for dependency updates while tracking
the stability profile of available upgrades.`,
		RunE: runDepsOutdated,
	}

	cmd.Flags().BoolP("verbose", "v", false, "Show detailed output including update check progress")
	return cmd
}

func newDepsSecurityCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security",
		Short: "Check for security vulnerabilities in dependencies",
		Long: `Check dependencies against the GitHub Security Advisory database.

Queries the GitHub Advisory API to identify known vulnerabilities in
Go dependencies and displays:
- Severity level (critical, high, medium, low)
- CVE identifier (if available)
- Summary of the vulnerability
- Fixed versions available
- Direct link to advisory details

This provides proactive security monitoring without requiring additional tooling.`,
		RunE: runDepsSecurity,
	}

	cmd.Flags().BoolP("verbose", "v", false, "Show detailed output including API query progress")
	return cmd
}

func newDepsReportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate comprehensive dependency health report",
		Long: `Generate a comprehensive dependency health report including:
- Summary statistics (total, direct, indirect dependencies)
- Outdated dependencies analysis
- Security vulnerability scan
- Dependency maturity breakdown (v0.x, v1.x, v2+)
- Actionable recommendations

Outputs a complete assessment in either human-readable or JSON format.
The JSON format is suitable for CI/CD pipelines and automation.`,
		RunE: runDepsReport,
	}

	cmd.Flags().BoolP("verbose", "v", false, "Show detailed output including progress information")
	cmd.Flags().Bool("json", false, "Output report in JSON format")
	return cmd
}

func runDepsHealth(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("=== Dependency Health Analysis ==="))
	fmt.Fprintln(os.Stderr, "")

	// Generate report
	report, err := GenerateDependencyReport(verbose)
	if err != nil {
		return fmt.Errorf("failed to generate dependency report: %w", err)
	}

	// Display health metrics
	fmt.Fprintf(os.Stderr, "Total dependencies: %d (%d direct, %d indirect)\n",
		report.TotalDeps, report.DirectDeps, report.IndirectDeps)
	fmt.Fprintln(os.Stderr, "")

	// Version breakdown
	fmt.Fprintln(os.Stderr, "Version breakdown:")
	v0Percentage := float64(report.V0Count) / float64(report.TotalDeps) * 100
	v1Percentage := float64(report.V1PlusCount) / float64(report.TotalDeps) * 100
	v2Percentage := float64(report.V2PlusCount) / float64(report.TotalDeps) * 100

	fmt.Fprintf(os.Stderr, "  v0.x (unstable): %d (%.1f%%)\n", report.V0Count, v0Percentage)
	fmt.Fprintf(os.Stderr, "  v1.x (stable):   %d (%.1f%%)\n", report.V1PlusCount, v1Percentage)
	fmt.Fprintf(os.Stderr, "  v2+ (mature):    %d (%.1f%%)\n", report.V2PlusCount, v2Percentage)
	fmt.Fprintln(os.Stderr, "")

	// Health assessment
	fmt.Fprintln(os.Stderr, "Health assessment:")
	if v0Percentage > 50 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("  ⚠️  High unstable dependency ratio (%.1f%% v0.x)", v0Percentage)))
		fmt.Fprintln(os.Stderr, "      Target: < 30% for improved stability")
	} else if v0Percentage > 30 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  ℹ️  Moderate unstable dependency ratio (%.1f%% v0.x)", v0Percentage)))
		fmt.Fprintln(os.Stderr, "      Target: < 30% for improved stability")
	} else {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("  ✓ Good unstable dependency ratio (%.1f%% v0.x)", v0Percentage)))
	}
	fmt.Fprintln(os.Stderr, "")

	// Quick status
	if len(report.Outdated) > 0 {
		outdatedPercentage := float64(len(report.Outdated)) / float64(report.DirectDeps) * 100
		fmt.Fprintf(os.Stderr, "Outdated dependencies: %d (%.0f%% of direct deps)\n", len(report.Outdated), outdatedPercentage)
	}
	if len(report.Advisories) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Security advisories: %d ⚠️", len(report.Advisories))))
	}

	return nil
}

func runDepsOutdated(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	outdated, err := CheckOutdatedDependencies(verbose)
	if err != nil {
		return fmt.Errorf("failed to check outdated dependencies: %w", err)
	}

	// Count total dependencies for percentage calculation
	goModPath, err := findGoMod()
	if err != nil {
		return fmt.Errorf("failed to find go.mod: %w", err)
	}

	deps, err := parseGoMod(goModPath)
	if err != nil {
		return fmt.Errorf("failed to parse go.mod: %w", err)
	}

	DisplayOutdatedDependencies(outdated, len(deps))
	return nil
}

func runDepsSecurity(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking for security vulnerabilities..."))
	fmt.Fprintln(os.Stderr, "")

	advisories, err := CheckSecurityAdvisories(verbose)
	if err != nil {
		return fmt.Errorf("failed to check security advisories: %w", err)
	}

	DisplaySecurityAdvisories(advisories)

	if len(advisories) > 0 {
		return fmt.Errorf("found %d security %s", len(advisories), pluralize("advisory", len(advisories)))
	}

	return nil
}

func runDepsReport(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	report, err := GenerateDependencyReport(verbose)
	if err != nil {
		return fmt.Errorf("failed to generate dependency report: %w", err)
	}

	if jsonOutput {
		return DisplayDependencyReportJSON(report)
	}

	DisplayDependencyReport(report)
	return nil
}
