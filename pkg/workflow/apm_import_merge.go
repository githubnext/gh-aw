package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// isUnsubstitutedImportExpression reports whether s looks like an unresolved
// ${{ github.aw.import-inputs.* }} template expression. Such values appear when
// a shared workflow uses import-schema inputs but the importer omits an optional
// parameter, leaving the placeholder unreplaced in the YAML.
// Only import-inputs expressions are matched — other ${{ }} expressions such as
// ${{ secrets.MY_TOKEN }} are valid and must be preserved.
func isUnsubstitutedImportExpression(s string) bool {
	return strings.Contains(s, "${{") && strings.Contains(s, "github.aw.import-inputs.")
}

// mergeImportedAPMPackages parses a list of JSON-serialized apm-packages configurations
// collected from imported shared workflows (e.g. shared/apm.md) and merges them into
// a single APMDependenciesInfo. Each element of apmPackagesConfigs is a JSON value
// that matches the apm-packages field format: either a JSON array of package strings
// or a JSON object with a "packages" key.
//
// Unresolved ${{ github.aw.import-inputs.* }} template expressions (left behind when
// an importer omits an optional parameter) are silently discarded so they do not
// propagate into the compiled workflow as literal token values.
//
// The first non-empty auth configuration (github-token, github-app, isolated) encountered
// across the imported configs is used (first-wins). Packages are deduplicated and merged.
//
// Returns nil if no packages are found across all configs.
func mergeImportedAPMPackages(apmPackagesConfigs []string) (*APMDependenciesInfo, error) {
	if len(apmPackagesConfigs) == 0 {
		return nil, nil
	}

	var allPackages []string
	seenPackages := make(map[string]bool)
	var githubToken string
	var githubApp *GitHubAppConfig
	var isolated bool

	for _, configJSON := range apmPackagesConfigs {
		if configJSON == "" {
			continue
		}
		// Unmarshal JSON to any so we can pass it to extractAPMDependenciesFromValue
		var apmValue any
		if err := json.Unmarshal([]byte(configJSON), &apmValue); err != nil {
			return nil, fmt.Errorf("failed to parse apm-packages JSON from import: %w", err)
		}
		// Strip unresolved import-input expressions from object fields before extraction.
		// This handles optional parameters (e.g. github-token, isolated) that were not
		// supplied by the importer and remain as literal "${{ github.aw.import-inputs.* }}" strings.
		if obj, ok := apmValue.(map[string]any); ok {
			for k, v := range obj {
				if str, isStr := v.(string); isStr && isUnsubstitutedImportExpression(str) {
					delete(obj, k)
				}
			}
		}
		deps, err := extractAPMDependenciesFromValue(apmValue, "imports[apm-packages]")
		if err != nil {
			return nil, err
		}
		if deps == nil {
			continue
		}
		for _, pkg := range deps.Packages {
			if !seenPackages[pkg] {
				seenPackages[pkg] = true
				allPackages = append(allPackages, pkg)
			}
		}
		// First-wins for auth config
		if githubToken == "" && deps.GitHubToken != "" {
			githubToken = deps.GitHubToken
		}
		if githubApp == nil && deps.GitHubApp != nil {
			githubApp = deps.GitHubApp
		}
		if !isolated && deps.Isolated {
			isolated = deps.Isolated
		}
	}

	if len(allPackages) == 0 {
		return nil, nil
	}
	apmDepsLog.Printf("Merged %d APM packages from imports", len(allPackages))
	return &APMDependenciesInfo{
		Packages:    allPackages,
		GitHubToken: githubToken,
		GitHubApp:   githubApp,
		Isolated:    isolated,
	}, nil
}
