package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run debug_json.go <file>")
		return
	}

	filePath := os.Args[1]
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	fmt.Printf("File size: %d bytes\n", len(content))

	// Try to parse as JSON array
	var logEntries []map[string]interface{}
	if err := json.Unmarshal(content, &logEntries); err != nil {
		fmt.Printf("Failed to parse as JSON array: %v\n", err)

		// Show first 200 characters
		first200 := content
		if len(first200) > 200 {
			first200 = first200[:200]
		}
		fmt.Printf("First 200 chars: %q\n", string(first200))

		// Show last 200 characters
		last200 := content
		if len(last200) > 200 {
			last200 = last200[len(last200)-200:]
		}
		fmt.Printf("Last 200 chars: %q\n", string(last200))
		return
	}

	fmt.Printf("Successfully parsed %d log entries\n", len(logEntries))

	// Look at the structure of entries
	assistantCount := 0
	resultCount := 0
	toolCallCount := 0

	for i, entry := range logEntries {
		if entryType, exists := entry["type"]; exists {
			if typeStr, ok := entryType.(string); ok {
				if typeStr == "result" {
					resultCount++
					fmt.Printf("Entry %d: type=result\n", i)
				} else if typeStr == "assistant" {
					assistantCount++
					fmt.Printf("Entry %d: type=assistant", i)
					if message, exists := entry["message"]; exists {
						if messageMap, ok := message.(map[string]interface{}); ok {
							if content, exists := messageMap["content"]; exists {
								if contentArray, ok := content.([]interface{}); ok {
									fmt.Printf(" (content array length: %d)", len(contentArray))
									for _, contentItem := range contentArray {
										if contentMap, ok := contentItem.(map[string]interface{}); ok {
											if contentType, exists := contentMap["type"]; exists {
												if typeStr, ok := contentType.(string); ok {
													if typeStr == "tool_use" {
														toolCallCount++
														if name, exists := contentMap["name"]; exists {
															fmt.Printf(" [tool_use: %v]", name)
														}
													} else {
														fmt.Printf(" [%s]", typeStr)
													}
												}
											}
										}
									}
								}
							}
						}
					}
					fmt.Printf("\n")
				} else {
					fmt.Printf("Entry %d: type=%s\n", i, typeStr)
				}
			}
		}
	}
	fmt.Printf("\nSummary: %d assistant entries, %d result entries, %d tool calls\n",
		assistantCount, resultCount, toolCallCount)
}
