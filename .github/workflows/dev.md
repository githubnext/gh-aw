---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot/*
engine: copilot
safe-outputs:
    staged: false
    edit-wiki:
      path: ["dev/", "docs/"]
      max: 3
      min: 1
---
Write a poem and add it to the wiki.