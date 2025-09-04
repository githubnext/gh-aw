package workflow

import (
	"fmt"
)

// buildCreateOutputMissingToolJob creates the missing_tool job
func (c *Compiler) buildCreateOutputMissingToolJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.MissingTool == nil {
		return nil, fmt.Errorf("safe-outputs.missing-tool configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Record Missing Tool\n")
	steps = append(steps, "        id: missing_tool\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))

	// Pass the max configuration if set
	if data.SafeOutputs.MissingTool.Max > 0 {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_MISSING_TOOL_MAX: %d\n", data.SafeOutputs.MissingTool.Max))
	}

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(missingToolScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"tools_reported": "${{ steps.missing_tool.outputs.tools_reported }}",
		"total_count":    "${{ steps.missing_tool.outputs.total_count }}",
	}

	// Create the job
	job := &Job{
		Name:           "missing_tool",
		RunsOn:         "runs-on: ubuntu-latest",
		If:             "if: ${{ always() }}",                // Always run to capture missing tools
		Permissions:    "permissions:\n      contents: read", // Only needs read access for logging
		TimeoutMinutes: 5,                                    // Short timeout since it's just processing output
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// missingToolScript is the JavaScript code that processes missing-tool output
const missingToolScript = `
const fs = require('fs');
const path = require('path');

// Get environment variables
const agentOutput = process.env.GITHUB_AW_AGENT_OUTPUT || '';
const maxReports = process.env.GITHUB_AW_MISSING_TOOL_MAX ? parseInt(process.env.GITHUB_AW_MISSING_TOOL_MAX) : null;

console.log('Processing missing-tool reports...');
console.log('Agent output length:', agentOutput.length);
if (maxReports) {
  console.log('Maximum reports allowed:', maxReports);
}

const missingTools = [];

if (agentOutput.trim()) {
  const lines = agentOutput.split('\n').filter(line => line.trim());
  
  for (const line of lines) {
    try {
      const entry = JSON.parse(line);
      
      if (entry.type === 'missing-tool') {
        // Validate required fields
        if (!entry.tool) {
          console.log('Warning: missing-tool entry missing "tool" field:', line);
          continue;
        }
        if (!entry.reason) {
          console.log('Warning: missing-tool entry missing "reason" field:', line);
          continue;
        }
        
        const missingTool = {
          tool: entry.tool,
          reason: entry.reason,
          alternatives: entry.alternatives || null,
          timestamp: new Date().toISOString()
        };
        
        missingTools.push(missingTool);
        console.log('Recorded missing tool:', missingTool.tool);
        
        // Check max limit
        if (maxReports && missingTools.length >= maxReports) {
          console.log('Reached maximum number of missing tool reports (${maxReports})');
          break;
        }
      }
    } catch (error) {
      console.log('Warning: Failed to parse line as JSON:', line);
      console.log('Parse error:', error.message);
    }
  }
}

console.log('Total missing tools reported:', missingTools.length);

// Output results
core.setOutput('tools_reported', JSON.stringify(missingTools));
core.setOutput('total_count', missingTools.length.toString());

// Log details for debugging
if (missingTools.length > 0) {
  console.log('Missing tools summary:');
  missingTools.forEach((tool, index) => {
    console.log('${index + 1}. Tool: ${tool.tool}');
    console.log('   Reason: ${tool.reason}');
    if (tool.alternatives) {
      console.log('   Alternatives: ${tool.alternatives}');
    }
    console.log('   Reported at: ${tool.timestamp}');
    console.log('');
  });
} else {
  console.log('No missing tools reported in this workflow execution.');
}
`
