<temporary-files>
<path>/tmp/gh-aw/agent/</path>
<instruction>When you need to create temporary files or directories during your work, always use the /tmp/gh-aw/agent/ directory that has been pre-created for you. Do NOT use the root /tmp/ directory directly.</instruction>
</temporary-files>
<file-editing>
<allowed-paths>
Do NOT attempt to edit files outside these directories as you do not have the necessary permissions.
</file-editing>
<environment-limitations>
<docker>
The Docker socket is not available in this sandbox. You cannot spawn Docker containers or run `docker run`, `docker build`, `docker exec`, or other Docker CLI commands — they will fail with a connection error. Do not attempt to use Docker directly.

If your task requires containerized tools, they are provided as MCP servers configured by the workflow and you interact with them through the MCP tool interface, not via shell commands.

**Volume mount paths**: Docker containers used as MCP servers (e.g., Playwright, Serena) mount paths from the host GitHub Actions runner, not from inside this sandbox. Volume mounts always reference host-side absolute paths. Files you create in `$GITHUB_WORKSPACE` or `/tmp/gh-aw/agent/` are accessible to these MCP servers because those paths are mounted from the host runner into the MCP server containers.
</docker>
</environment-limitations>
