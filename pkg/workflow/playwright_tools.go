package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var playwrightToolsLog = logger.New("workflow:playwright_tools")

// GetPlaywrightTools returns the list of Playwright browser tool names available in the
// copilot agent MCP server configuration.
// This is a shared function used by all engines for consistent Playwright tool configuration.
func GetPlaywrightTools() []any {
	tools := []string{
		"browser_click",
		"browser_close",
		"browser_console_messages",
		"browser_drag",
		"browser_evaluate",
		"browser_file_upload",
		"browser_fill_form",
		"browser_handle_dialog",
		"browser_hover",
		"browser_install",
		"browser_navigate",
		"browser_navigate_back",
		"browser_network_requests",
		"browser_press_key",
		"browser_resize",
		"browser_select_option",
		"browser_snapshot",
		"browser_tabs",
		"browser_take_screenshot",
		"browser_type",
		"browser_wait_for",
	}
	playwrightToolsLog.Printf("Returning %d Playwright tools", len(tools))
	return sliceutil.Map(tools, func(t string) any { return t })
}
