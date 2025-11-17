package main

import (
	"fmt"
	"os"
	
	"github.com/githubnext/gh-aw/pkg/cli"
)

func main() {
	content, err := os.ReadFile("/tmp/test_workflow.md")
	if err != nil {
		panic(err)
	}
	
	fmt.Println("===== ORIGINAL =====")
	fmt.Println(string(content))
	fmt.Println()
	
	// Test 1: Add source field
	fmt.Println("===== TEST 1: Add source field =====")
	result1, err := cli.UpdateFieldInFrontmatter(string(content), "source", "test/repo@v1.0.0")
	if err != nil {
		panic(err)
	}
	fmt.Println(result1)
	fmt.Println()
	
	// Test 2: Remove stop-after field
	fmt.Println("===== TEST 2: Remove stop-after field =====")
	result2, err := cli.RemoveFieldFromOnTrigger(string(content), "stop-after")
	if err != nil {
		panic(err)
	}
	fmt.Println(result2)
	fmt.Println()
	
	// Test 3: Set stop-after field
	fmt.Println("===== TEST 3: Set stop-after field =====")
	result3, err := cli.SetFieldInOnTrigger(string(content), "stop-after", "+72h")
	if err != nil {
		panic(err)
	}
	fmt.Println(result3)
}
