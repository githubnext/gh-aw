{
  echo "### Safe Outputs (JSONL)"
  echo ""
  echo '```json'
  if [ -f ${{ env.GH_AW_SAFE_OUTPUTS }} ]; then
    cat ${{ env.GH_AW_SAFE_OUTPUTS }}
    # Ensure there's a newline after the file content if it doesn't end with one
    if [ -s ${{ env.GH_AW_SAFE_OUTPUTS }} ] && [ "$(tail -c1 ${{ env.GH_AW_SAFE_OUTPUTS }})" != "" ]; then
      echo ""
    fi
  else
    echo "No agent output file found"
  fi
  echo '```'
  echo ""
} >> "$GITHUB_STEP_SUMMARY"
