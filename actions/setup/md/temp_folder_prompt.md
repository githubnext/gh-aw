<temporary-files>
<path>/tmp/gh-aw/agent/</path>
<instruction>When you need to create temporary files or directories during your work, always use the /tmp/gh-aw/agent/ directory that has been pre-created for you. Do NOT use the root /tmp/ directory directly.</instruction>
</temporary-files>
<file-editing>
<allowed-paths>
  <path name="workspace">$GITHUB_WORKSPACE</path>
  <path name="temporary">/tmp/gh-aw/</path>
</allowed-paths>
<restriction>Do NOT attempt to edit files outside these directories as you do not have the necessary permissions.</restriction>
</file-editing>
<markdown-generation>
<instruction>When generating markdown text, use 6 backticks instead of 3 to avoid creating unbalanced code regions where the text looks broken because the code regions are opening and closing out of sync.</instruction>
</markdown-generation>
