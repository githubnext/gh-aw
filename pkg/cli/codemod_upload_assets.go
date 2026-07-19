package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var uploadAssetsCodemodLog = logger.New("cli:codemod_upload_assets")

// getUploadAssetsCodemod creates a codemod for migrating upload-assets to upload-asset (plural to singular)
func getUploadAssetsCodemod() Codemod {
	return Codemod{
		ID:           "upload-assets-to-upload-asset-migration",
		Name:         "Migrate upload-assets to upload-asset",
		Description:  "Replaces deprecated 'safe-outputs.upload-assets' field with 'safe-outputs.upload-asset' (plural to singular)",
		IntroducedIn: "0.3.0",
		Apply:        getUploadAssetsCodemodApply,
	}
}

func getUploadAssetsCodemodApply(content string, frontmatter map[string]any) (string, bool, error) {
	if !getUploadAssetsCodemodNeedsMigration(frontmatter) {
		return content, false, nil
	}

	newContent, applied, err := applyFrontmatterLineTransform(content, getUploadAssetsCodemodTransform)
	if applied {
		uploadAssetsCodemodLog.Print("Applied upload-assets to upload-asset migration")
	}
	return newContent, applied, err
}

func getUploadAssetsCodemodNeedsMigration(frontmatter map[string]any) bool {
	// Check if safe-outputs.upload-assets exists
	safeOutputsValue, hasSafeOutputs := frontmatter["safe-outputs"]
	if !hasSafeOutputs {
		return false
	}

	safeOutputsMap, ok := safeOutputsValue.(map[string]any)
	if !ok {
		return false
	}

	// Check if upload-assets field exists in safe-outputs (plural is deprecated)
	_, hasUploadAssets := safeOutputsMap["upload-assets"]
	return hasUploadAssets
}

func getUploadAssetsCodemodTransform(lines []string) ([]string, bool) {
	var modified bool
	var inSafeOutputsBlock bool
	var safeOutputsIndent string
	result := make([]string, len(lines))
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		inSafeOutputsBlock, safeOutputsIndent = getUploadAssetsCodemodTrackSafeOutputsBlock(line, trimmedLine, inSafeOutputsBlock, safeOutputsIndent)
		result[i], modified = getUploadAssetsCodemodReplaceLine(line, trimmedLine, inSafeOutputsBlock, i, modified)
	}
	return result, modified
}

func getUploadAssetsCodemodTrackSafeOutputsBlock(line, trimmedLine string, inSafeOutputsBlock bool, safeOutputsIndent string) (bool, string) {
	// Track if we're in the safe-outputs block
	if strings.HasPrefix(trimmedLine, "safe-outputs:") {
		return true, getIndentation(line)
	}

	// Check if we've left the safe-outputs block
	if inSafeOutputsBlock && trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") && hasExitedBlock(line, safeOutputsIndent) {
		return false, safeOutputsIndent
	}
	return inSafeOutputsBlock, safeOutputsIndent
}

func getUploadAssetsCodemodReplaceLine(line, trimmedLine string, inSafeOutputsBlock bool, lineIndex int, modified bool) (string, bool) {
	// Replace upload-assets with upload-asset if in safe-outputs block
	if !inSafeOutputsBlock || !strings.HasPrefix(trimmedLine, "upload-assets:") {
		return line, modified
	}
	replacedLine, didReplace := findAndReplaceInLine(line, "upload-assets", "upload-asset")
	if didReplace {
		uploadAssetsCodemodLog.Printf("Replaced safe-outputs.upload-assets with safe-outputs.upload-asset on line %d", lineIndex+1)
		return replacedLine, true
	}
	return line, modified
}
