---
engine: ralph
on:
  workflow_dispatch:
permissions: {}

Write a simple "Hello World" function in Python and create a test for it. Follow these user stories:

## User Stories

```json
{
  "branchName": "ralph-test",
  "userStories": [
    {
      "id": "story-1",
      "title": "Create Hello World function",
      "description": "Create a simple Python function that returns 'Hello, World!'",
      "acceptanceCriteria": [
        "Function is in a file called hello.py",
        "Function is named get_greeting()",
        "Function returns the string 'Hello, World!'"
      ],
      "passes": false
    },
    {
      "id": "story-2", 
      "title": "Create test for Hello World function",
      "description": "Create a test that verifies the hello world function works correctly",
      "acceptanceCriteria": [
        "Test is in a file called test_hello.py",
        "Test verifies the function returns correct string",
        "Test can be run with pytest"
      ],
      "passes": false
    }
  ]
}
```

Please save this JSON to `prd.json` in the repository root and then work through each story.
