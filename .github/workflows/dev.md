---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot/*
      - collect-guards
engine: copilot
tools:
  github:
    allowed:
      - list_pull_requests
      - get_pull_request
safe-outputs:
    staged: true
    create-issue:
---
Generate a summary of all currently opened pull requests in this repository. Include the PR title, number, author, and a brief description of the changes. Post the summary as a new issue titled "Open Pull Requests Summary".