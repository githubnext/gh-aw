package workflow

import (
	"path/filepath"
	"strings"
)

func injectCurrentCheckoutPatchWorkspacePath(handlerName string, handlerCfg map[string]any, data *WorkflowData) {
	if handlerCfg == nil || data == nil {
		return
	}
	if handlerName != "create_pull_request" && handlerName != "push_to_pull_request_branch" {
		return
	}

	checkoutManager := NewCheckoutManager(data.CheckoutConfigs)
	currentPath := normalizeCurrentCheckoutPatchPath(checkoutManager.GetCurrentCheckoutPath())
	if currentPath == "" {
		return
	}
	currentRepo := strings.TrimSpace(checkoutManager.GetCurrentRepository())

	targetRepo := ""
	if value, ok := handlerCfg["target-repo"].(string); ok {
		targetRepo = strings.TrimSpace(value)
	}
	// Skip for wildcard and explicitly different repositories.
	if targetRepo == "*" {
		return
	}
	// If handler targets an explicit repository but current checkout resolved to
	// workflow repo (empty repository slug), do not inject a workspace override.
	if targetRepo != "" && currentRepo == "" {
		return
	}
	if targetRepo != "" && currentRepo != "" && targetRepo != currentRepo {
		return
	}

	handlerCfg["patch_workspace_path"] = currentPath
	if currentRepo != "" {
		handlerCfg["current_checkout_repo"] = currentRepo
	}
}

func normalizeCurrentCheckoutPatchPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "." {
		return ""
	}
	path = strings.TrimPrefix(path, "./")
	path = filepath.Clean(path)
	if path == "." || path == "" || filepath.IsAbs(path) {
		return ""
	}
	if path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return ""
	}
	return filepath.ToSlash(path)
}
