// Package main demonstrates Huh forms for interactive CLI input
//
// This example shows how to create accessible, multi-page forms
// with validation and different input types.
//
// Run: go run examples/console-output/huh-form-example.go
package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/charmbracelet/huh"
	"github.com/githubnext/gh-aw/pkg/console"
)

// isAccessibleMode detects if accessibility mode should be enabled
func isAccessibleMode() bool {
	return os.Getenv("ACCESSIBLE") != "" ||
		os.Getenv("TERM") == "dumb" ||
		os.Getenv("NO_COLOR") != ""
}

// ValidateWorkflowName validates workflow name format
func ValidateWorkflowName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("workflow name cannot be empty")
	}
	if len(name) > 100 {
		return fmt.Errorf("workflow name too long (max 100 characters)")
	}
	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(name) {
		return fmt.Errorf("workflow name must be lowercase alphanumeric with hyphens")
	}
	return nil
}

// ValidateInstructions validates workflow instructions
func ValidateInstructions(instructions string) error {
	if len(instructions) < 20 {
		return fmt.Errorf("instructions must be at least 20 characters")
	}
	return nil
}

func main() {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Interactive Workflow Builder"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("This example demonstrates Huh forms with accessibility support"))
	fmt.Fprintln(os.Stderr, "")

	// Example 1: Simple input form
	fmt.Fprintln(os.Stderr, console.FormatListHeader("Example 1: Simple Input Form"))
	
	var workflowName string
	
	simpleForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("What should we call this workflow?").
				Description("Enter a descriptive name (e.g., 'issue-triage', 'pr-review-helper')").
				Value(&workflowName).
				Validate(ValidateWorkflowName),
		),
	).WithAccessible(isAccessibleMode())

	if err := simpleForm.Run(); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Form error: %v", err)))
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Workflow name: %s", workflowName)))
	fmt.Fprintln(os.Stderr, "")

	// Example 2: Multi-page form with different input types
	fmt.Fprintln(os.Stderr, console.FormatListHeader("Example 2: Multi-Page Configuration Form"))
	
	var (
		engine         string
		tools          []string
		safeOutputs    []string
		networkAccess  string
		instructions   string
	)

	multiPageForm := huh.NewForm(
		// Page 1: Basic Configuration
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which AI engine should process this workflow?").
				Description("The AI engine interprets instructions and executes tasks").
				Options(
					huh.NewOption("copilot - GitHub Copilot CLI", "copilot"),
					huh.NewOption("claude - Anthropic Claude Code", "claude"),
					huh.NewOption("codex - OpenAI Codex engine", "codex"),
					huh.NewOption("custom - Custom engine", "custom"),
				).
				Value(&engine),
		).
			Title("Basic Configuration").
			Description("Choose your AI engine"),

		// Page 2: Tools Selection
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which tools should the AI have access to?").
				Description("Tools enable the AI to interact with code, APIs, and external systems").
				Options(
					huh.NewOption("github - GitHub API tools", "github"),
					huh.NewOption("edit - File editing tools", "edit"),
					huh.NewOption("bash - Shell commands", "bash"),
					huh.NewOption("web-fetch - Web content fetching", "web-fetch"),
					huh.NewOption("web-search - Web search", "web-search"),
					huh.NewOption("playwright - Browser automation", "playwright"),
				).
				Height(8).
				Value(&tools),
		).
			Title("Tool Selection").
			Description("Select available tools for your workflow"),

		// Page 3: Safe Outputs
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("What outputs should the AI be able to create?").
				Description("Safe outputs allow the AI to create GitHub resources after approval").
				Options(
					huh.NewOption("create-issue", "create-issue"),
					huh.NewOption("create-pull-request", "create-pull-request"),
					huh.NewOption("add-comment", "add-comment"),
					huh.NewOption("create-discussion", "create-discussion"),
					huh.NewOption("add-labels", "add-labels"),
				).
				Height(8).
				Value(&safeOutputs),
		).
			Title("Safe Outputs").
			Description("Configure output capabilities"),

		// Page 4: Network Configuration
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What network access does the workflow need?").
				Description("Network access controls which external domains can be reached").
				Options(
					huh.NewOption("defaults - Basic infrastructure", "defaults"),
					huh.NewOption("ecosystem - Development ecosystems (Python, Node, Go)", "ecosystem"),
					huh.NewOption("custom - Custom domains", "custom"),
				).
				Value(&networkAccess),
		).
			Title("Network Access").
			Description("Configure network permissions"),

		// Page 5: Instructions
		huh.NewGroup(
			huh.NewText().
				Title("Describe what this workflow should do:").
				Description("Provide clear, detailed instructions for the AI").
				Value(&instructions).
				Validate(ValidateInstructions),
		).
			Title("Workflow Instructions").
			Description("Define the workflow behavior"),
	).WithAccessible(isAccessibleMode())

	if err := multiPageForm.Run(); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Form cancelled: %v", err)))
		os.Exit(1)
	}

	// Display results
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configuration Complete!"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatListHeader("Configuration Summary:"))
	fmt.Fprintf(os.Stderr, "  Engine: %s\n", engine)
	fmt.Fprintf(os.Stderr, "  Tools: %v\n", tools)
	fmt.Fprintf(os.Stderr, "  Safe Outputs: %v\n", safeOutputs)
	fmt.Fprintf(os.Stderr, "  Network Access: %s\n", networkAccess)
	fmt.Fprintf(os.Stderr, "  Instructions: %s...\n", instructions[:50])
	fmt.Fprintln(os.Stderr, "")

	// Example 3: Confirmation dialog
	fmt.Fprintln(os.Stderr, console.FormatListHeader("Example 3: Confirmation Dialog"))
	
	var overwrite bool
	
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Workflow file already exists. Overwrite?").
				Description("This will replace the existing workflow configuration").
				Affirmative("Yes, overwrite").
				Negative("No, cancel").
				Value(&overwrite),
		),
	).WithAccessible(isAccessibleMode())

	if err := confirmForm.Run(); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Confirmation cancelled"))
		os.Exit(1)
	}

	if overwrite {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("User confirmed overwrite"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("User cancelled operation"))
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Accessibility features:"))
	fmt.Fprintln(os.Stderr, console.FormatListItem("Screen reader support"))
	fmt.Fprintln(os.Stderr, console.FormatListItem("High contrast mode"))
	fmt.Fprintln(os.Stderr, console.FormatListItem("Keyboard-only navigation"))
	fmt.Fprintln(os.Stderr, console.FormatListItem("Clear labels and descriptions"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Enable accessibility with: ACCESSIBLE=1 or NO_COLOR=1"))
}
