{
  echo "<details>"
  echo "<summary>Generated Prompt</summary>"
  echo ""
  echo '```markdown'
  cat "$GH_AW_PROMPT"
  echo '```'
  echo ""
  echo "</details>"
} >> "$GITHUB_STEP_SUMMARY"
