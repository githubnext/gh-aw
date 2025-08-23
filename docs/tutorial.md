# 🚀 Tutorial — Get a workflow running in minutes

This short, hands-on tutorial will get you from zero to a running agentic workflow. You'll install the extension, add a sample workflow, set up the required secrets, compile the workflow to a lock file, and run it. Ready? Let's go! 🎉

### What you'll do

- Install the `gh-aw` extension
- Add the sample `weekly-research` workflow to your repository
- Add the required AI API key secret
- Compile and run the workflow
- Inspect logs and iterate

### Prerequisites

- GitHub CLI (`gh`) installed and authenticated. Check with:

```bash
gh auth status
gh --version
```

- A repository you can push to (or a fork) and permission to add Actions secrets.

### Step 1 — Install the extension

```bash
gh extension install githubnext/gh-aw
```

Verify that `gh aw` is available:

```bash
gh aw --help
gh aw version
```

### Step 2 — Add a sample workflow

The easiest way to get started is to add a sample from the Agentics collection. From your repository root run:

```bash
gh aw add weekly-research -r githubnext/agentics --pr
```

This creates a pull request that adds `.github/workflows/weekly-research.md` and the compiled `.lock.yml`. Review and merge the PR into your repo.

### Step 3 — Add an AI secret

Agentic workflows use an AI engine. For Claude add this repository secret:

```bash
gh secret set ANTHROPIC_API_KEY -a actions --body "<your-anthropic-api-key>"
```

For Codex (experimental), add:

```bash
gh secret set OPENAI_API_KEY -a actions --body "<your-openai-api-key>"
```

These secrets are used by Actions at runtime.

### Step 4 — Compile and preview

Generate the compiled workflow file (`.lock.yml`) from your markdown source:

```bash
gh aw compile
```

To watch for edits while authoring (useful during development):

```bash
gh aw compile --watch
```

### Step 5 — Run the workflow

Trigger the workflow immediately (local execution or dispatch, depending on your setup):

```bash
gh aw run weekly-research
```

Download and inspect execution logs:

```bash
gh aw logs weekly-research
```

### Troubleshooting & diagnostics

- `gh aw status` — Check workflow installation status
- `gh aw logs` — Download recent run logs and cost/usage analysis
- `gh aw mcp-inspect` — Inspect MCP servers and tools
- If compilation fails, run `gh aw compile --verbose` for more details and follow the error output.

### What's next?

- Tweak the frontmatter and content to fit your use case 👩‍💻
- Explore other samples in [The Agentics](https://github.com/githubnext/agentics?tab=readme-ov-file#-the-agentics)
- Read more: [Workflow Structure](workflow-structure.md), [Frontmatter Options](frontmatter.md), [MCPs](mcps.md), and [Authoring in VS Code](vscode.md)

You did it — you're ready to start automating with agentic workflows! ✨

