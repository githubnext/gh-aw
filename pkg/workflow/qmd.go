// Package workflow provides qmd documentation search tool integration.
//
// # QMD Tool
//
// This file handles the qmd (https://github.com/tobi/qmd) builtin tool integration.
// qmd provides local vector search over documentation files using the @tobilu/qmd npm package.
//
// The integration has two phases:
//
//  1. Activation job: builds the search index from configured checkouts and/or GitHub searches
//     and uploads it as the "qmd-index" artifact. This step runs in the activation job which
//     already has contents:read permission, so the agent job does NOT need contents:read.
//
//  2. Agent job: downloads the "qmd-index" artifact and mounts the qmd MCP server pointing
//     at the pre-built index. The MCP server exposes a search tool that the agent can use
//     to find relevant documentation files.
//
// # Configuration
//
// Two sources can populate the index:
//
//   - checkouts: glob-based collections from checked-out repositories (each optionally with
//     its own checkout config to target a different repo)
//   - searches: GitHub search queries whose results are downloaded and added to the index
//
// Example frontmatter:
//
//	tools:
//	  qmd:
//	    checkouts:
//	      - name: docs
//	        docs:
//	          - docs/**/*.md
//	    searches:
//	      - query: "repo:owner/repo language:Markdown path:docs/"
//	        min: 1
//	        max: 30
//	        github-token: ${{ secrets.GITHUB_TOKEN }}
//
// # Artifact lifecycle
//
// The index is built once per activation job run and shared with the agent job
// via the "qmd-index" artifact.  Retention is 1 day (same as the activation artifact).
//
// Related files:
//   - tools_types.go: QmdToolConfig, QmdDocCollection, QmdSearchEntry types
//   - tools_parser.go: parseQmdTool / parseQmdDocCollection / parseQmdSearchEntry
//   - mcp_renderer_builtin.go: RenderQmdMCP method
//   - compiler_activation_job.go: activation job qmd index steps
//   - compiler_yaml_main_job.go: agent job qmd artifact download

package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var qmdLog = logger.New("workflow:qmd")

// shellSingleQuote wraps s in POSIX single quotes, escaping any embedded single
// quotes via the '"'"' idiom.  The result is safe to interpolate directly into
// a shell command: no shell metacharacters ($, `, \, ;, |, etc.) are
// interpreted inside single-quoted strings.
func shellSingleQuote(s string) string {
	// Replace each ' with '\'' (end-quote, literal-apostrophe, re-open-quote)
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// hasQmdTool checks if the qmd tool is enabled in the tools configuration.
func hasQmdTool(parsedTools *Tools) bool {
	if parsedTools == nil {
		return false
	}
	return parsedTools.Qmd != nil
}

// resolvedQmdCollection is an internal representation of a qmd collection
// with its working directory resolved.
type resolvedQmdCollection struct {
	name    string
	docs    []string
	workdir string // absolute path within the runner (e.g. ${GITHUB_WORKSPACE} or /tmp/gh-aw/qmd-checkout-<name>)
}

// resolveQmdCheckouts converts the checkouts (or legacy docs) portion of a QmdToolConfig
// into a list of resolvedQmdCollections.
func resolveQmdCheckouts(qmdConfig *QmdToolConfig) []resolvedQmdCollection {
	// Structured form: explicit checkouts list
	if len(qmdConfig.Checkouts) > 0 {
		resolved := make([]resolvedQmdCollection, 0, len(qmdConfig.Checkouts))
		for _, col := range qmdConfig.Checkouts {
			name := col.Name
			if name == "" {
				name = "docs"
			}
			workdir := "${GITHUB_WORKSPACE}"
			if col.Checkout != nil {
				if col.Checkout.Path != "" {
					// Checkout path is relative to GITHUB_WORKSPACE; strip leading "./" for cleanliness
					checkoutPath := strings.TrimPrefix(col.Checkout.Path, "./")
					workdir = "${GITHUB_WORKSPACE}/" + checkoutPath
				} else {
					// No explicit path → use an isolated temp directory
					workdir = "/tmp/gh-aw/qmd-checkout-" + name
				}
			}
			resolved = append(resolved, resolvedQmdCollection{
				name:    name,
				docs:    col.Docs,
				workdir: workdir,
			})
		}
		return resolved
	}

	// Legacy form: docs shorthand → single default collection
	docs := qmdConfig.Docs
	if len(docs) == 0 && len(qmdConfig.Searches) == 0 {
		// No explicit docs and no searches → default to all markdown
		docs = []string{"**/*.md"}
	}
	if len(docs) > 0 {
		return []resolvedQmdCollection{{
			name:    "docs",
			docs:    docs,
			workdir: "${GITHUB_WORKSPACE}",
		}}
	}
	return nil
}

// generateQmdCollectionCheckoutStep generates a checkout step YAML string for a qmd
// collection that targets a non-default repository.  Returns an empty string when the
// collection uses the current repository (no checkout needed).
func generateQmdCollectionCheckoutStep(col *QmdDocCollection) string {
	if col.Checkout == nil {
		return ""
	}
	cfg := col.Checkout

	// Determine checkout path used in the runner filesystem
	checkoutPath := cfg.Path
	if checkoutPath == "" {
		checkoutPath = "/tmp/gh-aw/qmd-checkout-" + col.Name
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "      - name: Checkout %s for qmd\n", col.Name)
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/checkout"))
	sb.WriteString("        with:\n")
	sb.WriteString("          persist-credentials: false\n")
	if cfg.Repository != "" {
		fmt.Fprintf(&sb, "          repository: %s\n", cfg.Repository)
	}
	if cfg.Ref != "" {
		fmt.Fprintf(&sb, "          ref: %s\n", cfg.Ref)
	}
	fmt.Fprintf(&sb, "          path: %s\n", checkoutPath)
	if cfg.GitHubToken != "" {
		fmt.Fprintf(&sb, "          token: %s\n", cfg.GitHubToken)
	}
	if cfg.FetchDepth != nil {
		fmt.Fprintf(&sb, "          fetch-depth: %d\n", *cfg.FetchDepth)
	}
	if cfg.SparseCheckout != "" {
		sb.WriteString("          sparse-checkout: |\n")
		for line := range strings.SplitSeq(strings.TrimRight(cfg.SparseCheckout, "\n"), "\n") {
			fmt.Fprintf(&sb, "            %s\n", strings.TrimSpace(line))
		}
	}
	if cfg.Submodules != "" {
		fmt.Fprintf(&sb, "          submodules: %s\n", cfg.Submodules)
	}
	if cfg.LFS {
		sb.WriteString("          lfs: true\n")
	}
	return sb.String()
}

// generateQmdSearchStep generates an activation-job step that runs a GitHub search query,
// downloads the matching files, and adds them as a qmd collection named after the search index.
// The step uses the gh CLI to execute the search.
func generateQmdSearchStep(entry *QmdSearchEntry, index int) string {
	collectionName := fmt.Sprintf("search-%d", index)
	searchDir := fmt.Sprintf("/tmp/gh-aw/qmd-search-%d", index)

	maxResults := entry.Max
	if maxResults <= 0 {
		maxResults = 30
	}

	// Build the GH_TOKEN env override if a custom token is provided
	var tokenEnv string
	if entry.GitHubToken != "" {
		tokenEnv = fmt.Sprintf("GH_TOKEN=%s ", entry.GitHubToken)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "      - name: Search GitHub for qmd collection %q\n", collectionName)
	sb.WriteString("        run: |\n")
	sb.WriteString("          set -e\n")
	fmt.Fprintf(&sb, "          mkdir -p %s\n", searchDir)

	// Execute gh search code, download each result file, then register the collection
	sb.WriteString("          # Download search results and add them to the qmd index\n")
	fmt.Fprintf(&sb, "          %sgh search code %s --limit %d --json path,repository | \\\n",
		tokenEnv,
		shellSingleQuote(entry.Query),
		maxResults,
	)
	// Use jq to extract repo+path pairs and download each file via gh api
	fmt.Fprintf(&sb, "            jq -r '.[] | .repository.fullName + \" \" + .path' | \\\n")
	fmt.Fprintf(&sb, "            while IFS=' ' read -r repo file_path; do\n")
	fmt.Fprintf(&sb, "              dest=%s/\"${repo//\\//-}\"-\"${file_path//\\//-}\"\n", searchDir)
	fmt.Fprintf(&sb, "              %sgh api \"repos/$repo/contents/$file_path\" --jq '.content' | base64 -d > \"$dest\" 2>/dev/null || true\n", tokenEnv)
	fmt.Fprintf(&sb, "            done\n")

	// Enforce minimum count
	if entry.Min > 0 {
		fmt.Fprintf(&sb, "          count=$(find %s -type f | wc -l)\n", searchDir)
		fmt.Fprintf(&sb, "          if [ \"$count\" -lt %d ]; then\n", entry.Min)
		fmt.Fprintf(&sb, "            echo \"qmd search %q returned $count results, minimum is %d\" >&2\n", collectionName, entry.Min)
		sb.WriteString("            exit 1\n")
		sb.WriteString("          fi\n")
	}

	// Add the downloaded files as a qmd collection
	fmt.Fprintf(&sb, "          QMD_CACHE_DIR=/tmp/gh-aw/qmd-index qmd collection add %s --name %s --glob %s\n",
		shellSingleQuote(searchDir),
		shellSingleQuote(collectionName),
		"'**/*'",
	)

	return sb.String()
}

// generateQmdIndexSteps generates the activation job steps that install qmd, register
// collections for each configured checkout and/or search, and build the vector search index.
// The index is stored at /tmp/gh-aw/qmd-index and uploaded as the qmd-index artifact.
func generateQmdIndexSteps(qmdConfig *QmdToolConfig, data *WorkflowData) []string {
	qmdLog.Printf("Generating qmd index steps: docs=%v checkouts=%d searches=%d",
		qmdConfig.Docs, len(qmdConfig.Checkouts), len(qmdConfig.Searches))

	version := string(constants.DefaultQmdVersion)
	var steps []string

	// Setup Node.js (required to run npm/npx)
	steps = append(steps, "      - name: Setup Node.js for qmd\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/setup-node")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          node-version: \"%s\"\n", string(constants.DefaultNodeVersion)))

	// Install qmd globally
	steps = append(steps, "      - name: Install qmd\n")
	steps = append(steps, "        run: |\n")
	steps = append(steps, fmt.Sprintf("          npm install -g @tobilu/qmd@%s\n", version))

	// Emit a checkout step for each checkout-based collection that needs its own repo
	for _, col := range qmdConfig.Checkouts {
		if checkoutStep := generateQmdCollectionCheckoutStep(col); checkoutStep != "" {
			steps = append(steps, checkoutStep)
		}
	}

	// Build the index: create the cache dir and register all collections
	steps = append(steps, "      - name: Build qmd index\n")
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          set -e\n")
	steps = append(steps, "          mkdir -p /tmp/gh-aw/qmd-index\n")

	// Register each checkout-based collection.
	// The workdir is double-quoted to preserve ${GITHUB_WORKSPACE} variable expansion.
	// User-provided names and globs are POSIX single-quoted to prevent shell injection.
	checkouts := resolveQmdCheckouts(qmdConfig)
	for _, col := range checkouts {
		var globArg string
		if len(col.docs) > 0 {
			globArg = shellSingleQuote(strings.Join(col.docs, ","))
		} else {
			globArg = "'**/*.md'"
		}
		steps = append(steps, fmt.Sprintf(
			"          QMD_CACHE_DIR=/tmp/gh-aw/qmd-index qmd collection add \"%s\" --name %s --glob %s\n",
			col.workdir,
			shellSingleQuote(col.name),
			globArg,
		))
	}

	// Emit a step per GitHub search entry
	for i, search := range qmdConfig.Searches {
		steps = append(steps, generateQmdSearchStep(search, i))
	}

	// Upload qmd index as a separate artifact for the agent job
	qmdLog.Print("Adding qmd index artifact upload step")
	qmdArtifactName := artifactPrefixExprForActivationJob(data) + constants.QmdArtifactName
	steps = append(steps, "      - name: Upload qmd index artifact\n")
	steps = append(steps, "        if: success()\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/upload-artifact")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          name: %s\n", qmdArtifactName))
	steps = append(steps, "          path: /tmp/gh-aw/qmd-index/\n")
	steps = append(steps, "          retention-days: 1\n")

	return steps
}

// generateQmdDownloadStep generates the agent job step that downloads the qmd-index artifact.
// Returns the steps as a YAML string slice ready to be appended to the agent job steps.
func generateQmdDownloadStep(data *WorkflowData) string {
	qmdArtifactName := artifactPrefixExprForDownstreamJob(data) + constants.QmdArtifactName
	var sb strings.Builder
	sb.WriteString("      - name: Download qmd index artifact\n")
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/download-artifact"))
	sb.WriteString("        with:\n")
	fmt.Fprintf(&sb, "          name: %s\n", qmdArtifactName)
	sb.WriteString("          path: /tmp/gh-aw/qmd-index/\n")
	return sb.String()
}
