# Print prompt to workflow logs (equivalent to core.info)
echo "Generated Prompt:"
cat "$GH_AW_PROMPT"

# Print prompt to step summary
{
  echo "<details>"
  echo "<summary>Generated Prompt</summary>"
  echo ""
  echo '``````markdown'
  cat "$GH_AW_PROMPT"
  echo '``````'
  echo ""
  echo "</details>"
} >> "$GITHUB_STEP_SUMMARY"
