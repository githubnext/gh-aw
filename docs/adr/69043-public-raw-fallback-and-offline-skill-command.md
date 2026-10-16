# ADR-69043: Fall back to unauthenticated public raw downloads on HTTP 403 and ship an offline `gh aw skill` command

**Date**: 2026-10-16
**Status**: Draft
**Deciders**: Unknown (inferred from PR #69043)

---

### Context

`gh aw` fetches instruction and skill content from the public `github/gh-aw` repository using the authenticated GitHub REST API (`pkg/parser/remote_download_file.go`, `pkg/cli/copilot_agents.go`). When a user's token is SAML-authorized for a different organization, those requests return HTTP 403 even though the requested content is public, so `gh aw init` and remote include resolution fail for content that any anonymous client could read. The existing fallback path was only triggered for authentication errors (`errorutil.IsAuthError`), not for authorization/403 responses. Separately, users who want the `/agentic-workflows` skill in an agent session had no way to obtain it without running `gh aw init`, which writes files into a repository and requires both a Git repository and network access. Any fix must not weaken enterprise-host isolation: private and GitHub Enterprise content must never be redirected to public github.com endpoints, and no token may be sent to anonymous raw URLs.

### Decision

We will treat HTTP 403 as a retryable-without-credentials condition for public `github/gh-aw` content and add a fully offline command for the bundled skill. Concretely: `downloadFileFromGitHubWithDepth` now also matches `errorutil.IsForbiddenError` before using the existing unauthenticated fallback; `fetchAgenticWorkflowsMarkdownFiles` falls back from the contents API to a public `raw.githubusercontent.com` JSON file list and then to the CLI's embedded list with a warning; and `gh aw skill` (`pkg/cli/skill_command.go`) renders the embedded skill content to stdout with no filesystem writes, no network calls, no authentication, and no Git repository requirement. The same raw-URL fallback rule is documented in the generated and checked-in agent/skill guidance so agents hitting 403 behave consistently, and host checks keep the fallback limited to public github.com content. `gh aw skill` deliberately uses the file list bundled with the installed CLI rather than fetching the latest list, trading freshness for guaranteed offline determinism.

### Alternatives Considered

#### Alternative 1: Surface a clearer 403 error and require the user to fix token authorization

The CLI could detect the SAML/403 case and tell the user to re-authorize their token for the organization, leaving the download path unchanged. This is the most "correct" answer for private content and avoids adding a second transport. It was rejected because the blocked content is public and readable anonymously, so failing closed imposes an org-admin round trip for data that needs no credentials at all, and it leaves `gh aw init` broken for a common enterprise token configuration.

#### Alternative 2: Always fetch public instruction/skill content anonymously

The CLI could skip the authenticated API entirely for `github/gh-aw` paths and always use `raw.githubusercontent.com`. This would be simpler (one code path, no error-class matching) and immune to token problems. It was rejected because it gives up API-level behaviour that the authenticated path provides (rate limits tied to the user, symlink/ref handling via the contents API, consistent behaviour for non-public or enterprise-hosted sources) and would change behaviour for every user to fix a minority failure mode. Keeping the anonymous path as a fallback confines the change to the failing case.

#### Alternative 3: Extend `gh aw init` with a `--print`/`--dry-run` flag instead of a new `skill` command

Rather than a new top-level command, the skill content could be emitted by `gh aw init --print`. This reuses existing wiring and avoids growing the command surface. It was rejected because `init` is defined by repository side effects (`.gitattributes`, skill/agent files, MCP config) and requires a Git repository; a flag that disables all of that is a different command in disguise. A separate `skill` command states the no-side-effect, no-network contract directly in its own help, args validation, and tests.

### Consequences

#### Positive
- `gh aw init` and remote instruction downloads keep working with tokens that are SAML-authorized for another organization, removing an org-admin dependency for public content.
- Agents can load the agentic-workflows skill with a single `gh aw skill` invocation, with no repository files created and no network or auth requirement.
- The 403 fallback behaviour is covered by regression tests asserting no `Authorization` header is sent to raw URLs, that repeated `init` runs stay idempotent, and that the embedded list is used when both public endpoints fail.
- The same fallback rule is encoded in agent-facing guidance and enforced by guidance contract tests, so human, CLI, and agent behaviour stay aligned.

#### Negative
- There are now two transports (authenticated API and anonymous raw) for the same content, so failures can occur in either and error reporting must keep both in view.
- `gh aw skill` output is only as fresh as the installed CLI version; users on an older release will print a stale `.github/aw` file list without any signal that newer files exist.
- A hard-coded raw URL to `pkg/cli/data/agentic_workflows_fallback_aw_files.json` on `main` couples runtime behaviour to a specific file path in the repository layout; moving or renaming that file silently breaks the middle fallback tier.
- Broadening the fallback trigger from auth errors to 403 widens the class of responses that cause an unauthenticated retry, so enterprise/private isolation now depends on host checks remaining correct rather than on the narrower error class.

#### Neutral
- The fallback chain for the file list is three-tiered (API → raw → embedded) and degrades with a warning rather than an error, so successful `init` runs may be based on bundled data.
- Skill-content rendering was factored out into `renderAgenticWorkflowsSkillContent`, shared by `init` and `skill`; future changes to skill rendering now affect both commands.
- `skill` joins the `utilities` command group and has no MCP tool equivalent, documented in `.github/aw/cli-commands.md` and `docs/src/content/docs/setup/cli.md`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
