package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Fprintf(os.Stderr, "Usage: %s <input-file> [output-file]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nBundles JavaScript files by inlining local requires.\n")
		fmt.Fprintf(os.Stderr, "\nIf output-file is not specified, writes to stdout.\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s input.cjs output.cjs\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s input.cjs\n", os.Args[0])
		if len(os.Args) < 2 {
			os.Exit(1)
		}
		os.Exit(0)
	}

	inputFile := os.Args[1]

	// Bundle the JavaScript file
	bundled, err := workflow.BundleJavaScript(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error bundling %s: %v\n", inputFile, err)
		os.Exit(1)
	}

	// Determine output
	if len(os.Args) >= 3 {
		outputFile := os.Args[2]

		// Create output directory if it doesn't exist
		outputDir := filepath.Dir(outputFile)
		if outputDir != "." && outputDir != "" {
			err = os.MkdirAll(outputDir, 0755)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
				os.Exit(1)
			}
		}

		// Write to output file
		err = os.WriteFile(outputFile, []byte(bundled), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", outputFile, err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "✓ Bundled %s -> %s\n", inputFile, outputFile)
	} else {
		// Write to stdout
		fmt.Print(bundled)
	}
}
