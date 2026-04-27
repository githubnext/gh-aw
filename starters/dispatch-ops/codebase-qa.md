---
name: Codebase Q&A
description: Answer natural-language questions about the codebase on demand using repository context.
on:
  workflow_dispatch:
    inputs:
      question:
        description: "What do you want to know about the codebase?"
        required: true
        type: string
permissions:
  contents: read
tools:
  github:
    toolsets: [repos]
safe-outputs:
  create-issue:
    title-prefix: "[qa] "
    labels: [documentation]
    max: 1
---

Answer the following question about the codebase: "${{ inputs.question }}".
Explore relevant source files and README documents to gather context before answering.
Create an issue with the question as the title and your detailed answer as the body so the response is permanently documented.
