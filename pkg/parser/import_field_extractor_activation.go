// Package parser provides functions for parsing and processing workflow markdown files.
// import_field_extractor_activation.go implements extraction of activation and
// authentication-related fields (bots, skip-roles, skip-bots, skip-if-match,
// skip-if-no-match, ambient-folders, github-token, github-app, checkout) from
// imported frontmatter.
package parser

import (
	"encoding/json"
)

// extractActivationFields extracts activation and authentication-related fields from
// the frontmatter map: bots, skip-roles, skip-bots, skip-if-match, skip-if-no-match,
// top-level ambient-folders, on.github-token, on.github-app, top-level github-app, and checkout.
//
// Side effects: acc.bots, acc.botsSet, acc.skipRoles, acc.skipRolesSet, acc.skipBots,
// acc.skipBotsSet, acc.skipIfMatch, acc.skipIfNoMatch, acc.activationGitHubToken,
// acc.activationGitHubApp, acc.topLevelGitHubApp, acc.checkouts.
func (acc *importAccumulator) extractActivationFields(fm map[string]any, item importQueueItem) {
	acc.mergeBots(fm)
	acc.mergeSkipRoles(fm)
	acc.mergeSkipBots(fm)
	acc.mergeAmbientFolders(fm)
	acc.extractActivationSkipMatchFields(fm, item.fullPath)
	acc.extractActivationGitHubToken(fm, item.fullPath)
	acc.extractActivationGitHubAppFields(fm, item.fullPath)
	acc.extractCheckoutField(fm, item.fullPath)
}

func (acc *importAccumulator) mergeBots(fm map[string]any) {
	mergeJSONStringListField(fm, "bots", "[]", acc.botsSet, &acc.bots, func(m map[string]any, field string) (string, error) {
		return extractFieldJSONFromMap(m, field, "[]")
	})
}

func (acc *importAccumulator) mergeSkipRoles(fm map[string]any) {
	mergeJSONStringListField(fm, "skip-roles", "[]", acc.skipRolesSet, &acc.skipRoles, extractOnSectionFieldFromMap)
}

func (acc *importAccumulator) mergeSkipBots(fm map[string]any) {
	mergeJSONStringListField(fm, "skip-bots", "[]", acc.skipBotsSet, &acc.skipBots, extractOnSectionFieldFromMap)
}

func (acc *importAccumulator) mergeAmbientFolders(fm map[string]any) {
	mergeJSONStringListField(fm, "ambient-folders", "[]", acc.ambientFoldersSet, &acc.ambientFolders, func(m map[string]any, field string) (string, error) {
		return extractFieldJSONFromMap(m, field, "[]")
	})
}

func (acc *importAccumulator) extractActivationSkipMatchFields(fm map[string]any, fullPath string) {
	if acc.skipIfMatch == "" {
		if skipJSON, skipErr := extractOnSectionAnyFieldFromMap(fm, "skip-if-match"); skipErr == nil && skipJSON != "" && skipJSON != "null" {
			acc.skipIfMatch = skipJSON
			parserLog.Printf("Extracted on.skip-if-match from import: %s", fullPath)
		}
	}
	if acc.skipIfNoMatch == "" {
		if skipJSON, skipErr := extractOnSectionAnyFieldFromMap(fm, "skip-if-no-match"); skipErr == nil && skipJSON != "" && skipJSON != "null" {
			acc.skipIfNoMatch = skipJSON
			parserLog.Printf("Extracted on.skip-if-no-match from import: %s", fullPath)
		}
	}
}

func (acc *importAccumulator) extractActivationGitHubToken(fm map[string]any, fullPath string) {
	if acc.activationGitHubToken != "" {
		return
	}
	tokenJSON, tokenErr := extractOnSectionAnyFieldFromMap(fm, "github-token")
	if tokenErr != nil || tokenJSON == "" || tokenJSON == "null" {
		return
	}
	var token string
	if jsonErr := json.Unmarshal([]byte(tokenJSON), &token); jsonErr == nil && token != "" {
		acc.activationGitHubToken = token
		parserLog.Printf("Extracted on.github-token from import: %s", fullPath)
	}
}

func (acc *importAccumulator) extractActivationGitHubAppFields(fm map[string]any, fullPath string) {
	if acc.activationGitHubApp == "" {
		if appJSON, appErr := extractOnSectionAnyFieldFromMap(fm, "github-app"); appErr == nil {
			if validated := validateGitHubAppJSON(appJSON); validated != "" {
				acc.activationGitHubApp = validated
				parserLog.Printf("Extracted on.github-app from import: %s", fullPath)
			}
		}
	}
	if acc.topLevelGitHubApp == "" {
		if appJSON, appErr := extractFieldJSONFromMap(fm, "github-app", ""); appErr == nil {
			if validated := validateGitHubAppJSON(appJSON); validated != "" {
				acc.topLevelGitHubApp = validated
				parserLog.Printf("Extracted top-level github-app from import: %s", fullPath)
			}
		}
	}
}

func (acc *importAccumulator) extractCheckoutField(fm map[string]any, fullPath string) {
	checkoutJSON, checkoutErr := extractFieldJSONFromMap(fm, "checkout", "")
	if checkoutErr != nil || checkoutJSON == "" || checkoutJSON == "null" || checkoutJSON == "false" {
		return
	}
	acc.checkouts = append(acc.checkouts, checkoutJSON)
	parserLog.Printf("Extracted checkout from import: %s", fullPath)
}
