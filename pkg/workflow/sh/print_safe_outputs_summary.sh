echo "## Safe Outputs (JSONL)" >> $GITHUB_STEP_SUMMARY
echo "" >> $GITHUB_STEP_SUMMARY
echo '```json' >> $GITHUB_STEP_SUMMARY
if [ -f ${{ env.GH_AW_SAFE_OUTPUTS }} ]; then
  cat ${{ env.GH_AW_SAFE_OUTPUTS }} >> $GITHUB_STEP_SUMMARY
  # Ensure there's a newline after the file content if it doesn't end with one
  if [ -s ${{ env.GH_AW_SAFE_OUTPUTS }} ] && [ "$(tail -c1 ${{ env.GH_AW_SAFE_OUTPUTS }})" != "" ]; then
    echo "" >> $GITHUB_STEP_SUMMARY
  fi
else
  echo "No agent output file found" >> $GITHUB_STEP_SUMMARY
fi
echo '```' >> $GITHUB_STEP_SUMMARY
echo "" >> $GITHUB_STEP_SUMMARY
