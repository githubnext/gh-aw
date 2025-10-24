const logContent = `2025-10-24T16:00:00.000Z [INFO] Starting Copilot CLI: 0.0.350
2025-10-24T16:00:01.000Z [DEBUG] response (Request-ID test-1):
2025-10-24T16:00:01.000Z [DEBUG] data:
{
  "id": "chatcmpl-1",
  "model": "claude-sonnet-4",
  "choices": [{
    "message": {
      "role": "assistant",
      "content": "I'll create an issue for you.",
      "tool_calls": [
        {
          "id": "call_create_issue_123",
          "type": "function",
          "function": {
            "name": "github-create_issue",
            "arguments": "{\\"title\\":\\"Test Issue\\",\\"body\\":\\"Test body\\"}"
          }
        }
      ]
    },
    "finish_reason": "tool_calls"
  }],
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 50
  }
}
2025-10-24T16:00:02.000Z [ERROR] Tool execution failed: github-create_issue
2025-10-24T16:00:02.000Z [ERROR] Permission denied: Resource not accessible by integration`;

function scanForToolErrors(logContent) {
  const toolErrors = new Map();
  const lines = logContent.split("\n");
  
  let currentToolCall = null;
  
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    
    // Track tool calls to associate errors with them
    // Look for tool_calls in the JSON data blocks
    if (line.includes('"tool_calls"')) {
      console.log("Found tool_calls at line", i);
      // Next few lines should contain tool call details
      for (let j = i + 1; j < Math.min(i + 20, lines.length); j++) {
        const nextLine = lines[j];
        // Extract tool call ID and function name
        const idMatch = nextLine.match(/"id":\s*"([^"]+)"/);
        const nameMatch = nextLine.match(/"name":\s*"([^"]+)"/);
        
        if (idMatch && nameMatch) {
          currentToolCall = { id: idMatch[1], name: nameMatch[1] };
          console.log("Found tool call:", currentToolCall);
        }
      }
    }
    
    // Look for error messages
    const errorMatch = line.match(/\[ERROR\].*(?:Tool execution failed|Permission denied|Resource not accessible|Error executing tool)/i);
    if (errorMatch) {
      console.log("Found error at line", i, ":", line);
      console.log("Current tool call:", currentToolCall);
      
      // Try to extract tool name from error line
      const toolNameMatch = line.match(/Tool execution failed:\s*([^\s]+)/i);
      const toolIdMatch = line.match(/tool_call_id:\s*([^\s]+)/i);
      
      if (toolNameMatch) {
        console.log("Setting error for tool name:", toolNameMatch[1]);
        toolErrors.set(toolNameMatch[1], true);
      } else if (toolIdMatch) {
        console.log("Setting error for tool ID:", toolIdMatch[1]);
        toolErrors.set(toolIdMatch[1], true);
      } else if (currentToolCall) {
        // Mark the current tool call as failed
        console.log("Setting error for current tool:", currentToolCall);
        toolErrors.set(currentToolCall.id, true);
        toolErrors.set(currentToolCall.name, true);
      }
    }
  }
  
  console.log("Final toolErrors map:", Array.from(toolErrors.entries()));
  return toolErrors;
}

const errors = scanForToolErrors(logContent);
