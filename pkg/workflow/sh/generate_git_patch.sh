# Check current git status
echo "Current git status:"
git status

# Show patch info if patches directory exists
if [ -d /tmp/gh-aw/patches ]; then
  echo "Patches directory found"
  ls -la /tmp/gh-aw/patches/
  
  # Count patches
  PATCH_COUNT=$(ls -1 /tmp/gh-aw/patches/*.patch 2>/dev/null | wc -l)
  
  if [ "$PATCH_COUNT" -gt 0 ]; then
    echo "Found $PATCH_COUNT patch file(s)"
    
    # Show summary in step summary
    echo '## Git Patches' >> $GITHUB_STEP_SUMMARY
    echo '' >> $GITHUB_STEP_SUMMARY
    echo "Found $PATCH_COUNT patch file(s)" >> $GITHUB_STEP_SUMMARY
    echo '' >> $GITHUB_STEP_SUMMARY
    
    # Show preview of each patch (truncated)
    for patch_file in /tmp/gh-aw/patches/*.patch; do
      if [ -f "$patch_file" ]; then
        patch_name=$(basename "$patch_file")
        echo "### Patch: $patch_name" >> $GITHUB_STEP_SUMMARY
        echo '' >> $GITHUB_STEP_SUMMARY
        echo '```diff' >> $GITHUB_STEP_SUMMARY
        head -100 "$patch_file" >> $GITHUB_STEP_SUMMARY || echo "Could not display patch contents" >> $GITHUB_STEP_SUMMARY
        
        # Check if file is longer than 100 lines
        LINE_COUNT=$(wc -l < "$patch_file")
        if [ "$LINE_COUNT" -gt 100 ]; then
          echo '...' >> $GITHUB_STEP_SUMMARY
          echo "(Truncated - showing first 100 of $LINE_COUNT lines)" >> $GITHUB_STEP_SUMMARY
        fi
        
        echo '```' >> $GITHUB_STEP_SUMMARY
        echo '' >> $GITHUB_STEP_SUMMARY
      fi
    done
  else
    echo "No patch files found in patches directory"
  fi
else
  echo "Patches directory does not exist"
fi
