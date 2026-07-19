package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/constants"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/workflow"
)

// writeMCPScriptsFiles writes all mcp-scripts MCP server files to the specified directory
func writeMCPScriptsFiles(dir string, mcpScriptsConfig *workflow.MCPScriptsConfig, verbose bool) error {
	mcpInspectLog.Printf("Writing mcp-scripts files to: %s", dir)

	if err := writeMCPScriptsFilesLogsDir(dir); err != nil {
		return err
	}
	if err := writeMCPScriptsFilesJavaScript(dir, verbose); err != nil {
		return err
	}
	if err := writeMCPScriptsFilesToolsConfig(dir, mcpScriptsConfig, verbose); err != nil {
		return err
	}
	if err := writeMCPScriptsFilesServerScript(dir, mcpScriptsConfig, verbose); err != nil {
		return err
	}
	if err := writeMCPScriptsFilesToolHandlers(dir, mcpScriptsConfig, verbose); err != nil {
		return err
	}

	mcpInspectLog.Printf("Successfully wrote all mcp-scripts files")
	return nil
}

func writeMCPScriptsFilesLogsDir(dir string) error {
	// Create logs directory
	logsDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logsDir, constants.DirPermPublic); err != nil {
		errMsg := fmt.Sprintf("failed to create logs directory: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(errMsg))
		return fmt.Errorf("failed to create logs directory: %w", err)
	}
	return nil
}

func writeMCPScriptsFilesJavaScript(dir string, verbose bool) error {
	// Write JavaScript dependencies that are needed
	jsFiles := []struct {
		name    string
		content string
	}{
		{"read_buffer.cjs", workflow.GetReadBufferScript()},
		{"mcp_http_transport.cjs", workflow.GetMCPHTTPTransportScript()},
		{"mcp_scripts_config_loader.cjs", workflow.GetMCPScriptsConfigLoaderScript()},
		{"mcp_server_core.cjs", workflow.GetMCPServerCoreScript()},
		{"mcp_scripts_validation.cjs", workflow.GetMCPScriptsValidationScript()},
		{"mcp_logger.cjs", workflow.GetMCPLoggerScript()},
		{"mcp_handler_shell.cjs", workflow.GetMCPHandlerShellScript()},
		{"mcp_handler_python.cjs", workflow.GetMCPHandlerPythonScript()},
		{"mcp_scripts_mcp_server_http.cjs", workflow.GetMCPScriptsMCPServerHTTPScript()},
	}

	for _, jsFile := range jsFiles {
		if err := writeMCPScriptsFilesFile(dir, jsFile.name, jsFile.content, constants.FilePermPublic, verbose); err != nil {
			return err
		}
	}
	return nil
}

func writeMCPScriptsFilesToolsConfig(dir string, mcpScriptsConfig *workflow.MCPScriptsConfig, verbose bool) error {
	// Generate and write tools.json
	toolsJSON := workflow.GenerateMCPScriptsToolsConfig(mcpScriptsConfig)
	return writeMCPScriptsFilesFile(dir, "tools.json", toolsJSON, constants.FilePermPublic, verbose)
}

func writeMCPScriptsFilesServerScript(dir string, mcpScriptsConfig *workflow.MCPScriptsConfig, verbose bool) error {
	// Generate and write mcp-server.cjs entry point
	mcpServerScript := workflow.GenerateMCPScriptsMCPServerScript(mcpScriptsConfig)
	return writeMCPScriptsFilesFile(dir, "mcp-server.cjs", mcpServerScript, constants.FilePermExecutable, verbose)
}

func writeMCPScriptsFilesToolHandlers(dir string, mcpScriptsConfig *workflow.MCPScriptsConfig, verbose bool) error {
	// Generate and write tool handler files
	for toolName, toolConfig := range mcpScriptsConfig.Tools {
		var content string
		var extension string

		if toolConfig.Script != "" {
			content = workflow.GenerateMCPScriptJavaScriptToolScript(toolConfig)
			extension = ".cjs"
		} else if toolConfig.Run != "" {
			content = workflow.GenerateMCPScriptShellToolScript(toolConfig)
			extension = ".sh"
		} else if toolConfig.Py != "" {
			content = workflow.GenerateMCPScriptPythonToolScript(toolConfig)
			extension = ".py"
		} else {
			continue
		}

		toolPath := filepath.Join(dir, toolName+extension)
		mode := constants.FilePermPublic
		if extension == ".sh" || extension == ".py" {
			mode = constants.FilePermExecutable
		}
		if err := os.WriteFile(toolPath, []byte(content), mode); err != nil {
			errMsg := fmt.Sprintf("failed to write tool %s: %v", toolName, err)
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(errMsg))
			return fmt.Errorf("failed to write tool %s: %w", toolName, err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Wrote tool handler: %s%s", toolName, extension)))
		}
	}
	return nil
}

func writeMCPScriptsFilesFile(dir string, name string, content string, mode os.FileMode, verbose bool) error {
	filePath := filepath.Join(dir, name)
	if err := os.WriteFile(filePath, []byte(content), mode); err != nil {
		errMsg := fmt.Sprintf("failed to write %s: %v", name, err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(errMsg))
		return fmt.Errorf("failed to write %s: %w", name, err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Wrote "+name))
	}
	return nil
}
