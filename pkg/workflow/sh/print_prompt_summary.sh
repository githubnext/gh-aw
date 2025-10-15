echo "<details>" >> $GITHUB_STEP_SUMMARY
echo "<summary>Generated Prompt</summary>" >> $GITHUB_STEP_SUMMARY
echo "" >> $GITHUB_STEP_SUMMARY
echo '```markdown' >> $GITHUB_STEP_SUMMARY
cat $GITHUB_AW_PROMPT >> $GITHUB_STEP_SUMMARY
echo '```' >> $GITHUB_STEP_SUMMARY
echo "" >> $GITHUB_STEP_SUMMARY
echo "</details>" >> $GITHUB_STEP_SUMMARY
