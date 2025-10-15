---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  create-pull-request:
    title-prefix: "[test] "
    labels: [test, automation]
---

# Test Multiple Pull Requests

This is a test workflow to verify that an agent can create multiple pull requests in a single run.

Please create two separate pull requests:

## First Pull Request
1. Create a new branch called "poem-nature"
2. Create a file called `poems/nature.md` with a short poem about nature (4-6 lines)
3. Commit the file with message "Add nature poem"
4. Create a pull request with title "Add Nature Poem" and a brief description

## Second Pull Request
1. Create a new branch called "poem-technology"
2. Create a file called `poems/technology.md` with a short poem about technology (4-6 lines)
3. Commit the file with message "Add technology poem"
4. Create a pull request with title "Add Technology Poem" and a brief description

Both pull requests should be created in this single workflow run.
