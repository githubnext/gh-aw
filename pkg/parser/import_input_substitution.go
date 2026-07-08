// Package parser provides functions for parsing and processing workflow markdown files.
// import_input_substitution.go implements text-level substitution of import-inputs
// expressions (${{ github.aw.import-inputs.* }}) in raw workflow file content.
package parser

import (
	"regexp"

	"github.com/github/gh-aw/pkg/importinpututil"
)

// importInputsExprRegex matches ${{ github.aw.import-inputs.<key> }} and
// ${{ github.aw.import-inputs.<key>.<subkey> }} expressions in raw content.
var importInputsExprRegex = regexp.MustCompile(`\$\{\{\s*github\.aw\.import-inputs\.([a-zA-Z0-9_-]+(?:\.[a-zA-Z0-9_-]+)?)\s*\}\}`)

// legacyInputsExprRegex matches ${{ github.aw.inputs.<key> }} (legacy form) in raw content.
var legacyInputsExprRegex = regexp.MustCompile(`\$\{\{\s*github\.aw\.inputs\.([a-zA-Z0-9_-]+)\s*\}\}`)

// substituteImportInputsInContent performs text-level substitution of
// ${{ github.aw.import-inputs.* }} and ${{ github.aw.inputs.* }} expressions
// in raw file content (including YAML frontmatter). This is called before YAML
// parsing so that array/object values serialised as JSON produce valid YAML.
func substituteImportInputsInContent(content string, inputs map[string]any) string {
	if len(inputs) == 0 {
		return content
	}

	importLog.Printf("Substituting import-inputs expressions: inputs=%d, contentBytes=%d", len(inputs), len(content))

	result := legacyInputsExprRegex.ReplaceAllStringFunc(content, buildImportInputReplaceFunc(legacyInputsExprRegex, inputs))
	result = importInputsExprRegex.ReplaceAllStringFunc(result, buildImportInputReplaceFunc(importInputsExprRegex, inputs))
	return result
}

func buildImportInputReplaceFunc(regex *regexp.Regexp, inputs map[string]any) func(string) string {
	return func(match string) string {
		m := regex.FindStringSubmatch(match)
		if len(m) < 2 {
			return match
		}
		strVal, found := resolveImportInputPath(inputs, m[1])
		if found {
			return strVal
		}
		return match
	}
}

func resolveImportInputPath(inputs map[string]any, inputPath string) (string, bool) {
	value, ok := resolveImportInputValue(inputs, inputPath)
	if !ok {
		return "", false
	}
	return importinpututil.FormatResolvedValue(value)
}

func resolveImportInputValue(inputs map[string]any, inputPath string) (any, bool) {
	return importinpututil.ResolvePathValue(inputs, inputPath)
}
