---
tools:
  claude:
    allowed:
      Edit:
      MultiEdit:
      Write:
      Bash:
      - "echo:*"
---

### Output Report implemented via GitHub Action Job Summary

You will use the Job Summary for GitHub Actions run ${{ github.run_id }} in ${{ github.repository }} to report progess. This means writing to the special file $GITHUB_STEP_SUMMARY. You can write the file using "echo" or the "Write" tool. GITHUB_STEP_SUMMARY is an environment variable set by GitHub Actions which you can use to write the report. You can read this environment variable using the bash command "echo $GITHUB_STEP_SUMMARY".

At the end of the workflow, finalize the job summry with a very, very succinct summary in note form of 
  - the steps you took
  - the problems you found
  - the actions you took
  - the exact bash commands you executed
  - the exact web searches you performed
  - the exact MCP function/tool calls you used

If any step fails, then make this really obvious with emoji. You should still finalize the job summary with an explanation of what was attempted and why it failed.

Include this at the end of the job summary:

  ```
  > AI-generated content by [${{ github.workflow }}](https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }}) may contain mistakes.
  ```
