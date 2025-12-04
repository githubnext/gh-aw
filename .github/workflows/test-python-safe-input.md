---
description: Test Python safe-input tool for data processing
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
safe-inputs:
  analyze-numbers:
    description: "Analyze a list of numbers using Python - calculates sum, average, min, max"
    inputs:
      numbers:
        type: string
        description: "Comma-separated list of numbers to analyze"
        required: true
    py: |
      import json
      
      # Get input from inputs dictionary
      numbers_str = inputs.get('numbers', '')
      
      # Parse and analyze numbers
      try:
          numbers = [float(x.strip()) for x in numbers_str.split(',') if x.strip()]
          
          if not numbers:
              result = {"error": "No valid numbers provided"}
          else:
              result = {
                  "count": len(numbers),
                  "sum": sum(numbers),
                  "average": sum(numbers) / len(numbers),
                  "min": min(numbers),
                  "max": max(numbers)
              }
          
          # Print result as JSON to stdout
          print(json.dumps(result))
                  
      except ValueError as e:
          error_result = {"error": f"Invalid number format: {str(e)}"}
          print(json.dumps(error_result))
  
  process-text:
    description: "Process text with Python - converts to uppercase, counts words and characters"
    inputs:
      text:
        type: string
        description: "Text to process"
        required: true
    py: |
      import json
      
      # Get input from inputs dictionary
      text = inputs.get('text', '')
      
      # Process text
      result = {
          "original": text,
          "uppercase": text.upper(),
          "word_count": len(text.split()),
          "char_count": len(text),
          "char_count_no_spaces": len(text.replace(' ', ''))
      }
      
      # Print result as JSON to stdout
      print(json.dumps(result))

safe-outputs:
  create-issue:
    title-prefix: "Python Safe-Input Test Results"
    labels: ["test", "python"]
timeout-minutes: 5
---

# Python Safe-Input Test Workflow

This workflow tests Python safe-input functionality.

## Task

1. Test the `analyze-numbers` tool with a sample list of numbers: "10, 20, 30, 40, 50"
2. Test the `process-text` tool with a sample text: "Hello World from Python safe-inputs"
3. Report the results from both tools
4. Create an issue with the test results

## Expected Results

The `analyze-numbers` tool should return:
- count: 5
- sum: 150
- average: 30
- min: 10
- max: 50

The `process-text` tool should return:
- uppercase version of the text
- word count: 6
- character count and other statistics

Create an issue with the test results, including both successful outputs.
