"""Documentation contract regressions, not an LLM or live workflow evaluation."""

import json
from pathlib import Path
import re
import unittest


SKILL = Path(__file__).resolve().parents[1]
ROOT = SKILL.parents[2]
DEBUG_GUIDE = ROOT / ".github/aw/debug-agentic-workflow.md"
GUIDES = (ROOT / ".github/aw/debug-model-engine-misconfiguration.md",)
BOOTSTRAP = ROOT / "debug.md"
FIXTURES = json.loads((SKILL / "tests/fixtures.json").read_text())
ANCHOR = "#model-and-engine-misconfiguration"


def checklist(text):
    if "## Model and engine misconfiguration\n" not in text:
        return text
    return text.split("## Model and engine misconfiguration\n", 1)[1].split("\n## ", 1)[0]


class GuidanceTests(unittest.TestCase):
    def test_debug_bootstrap_loads_shared_strategy_before_setup(self):
        guide = BOOTSTRAP.read_text()
        load = guide.index("## 1. Load the Debugging Strategy First")
        setup = guide.index("## 2. Check the CLI and Run Identity")
        audit = guide.index("## 3. Audit the Supplied Run")
        self.assertLess(load, setup)
        self.assertLess(setup, audit)
        for required in (
            "ROOT/.github/aw/debug-agentic-workflow.md",
            "same ref",
            "against the file containing the link",
            "raw-content equivalent",
            "HTTP 403",
            "without an Authorization header or token",
            "report the error and stop",
            "target repository's `AGENTS.md`",
            "apply its repository overlay",
        ):
            with self.subTest(bootstrap=required):
                self.assertIn(required, guide[load:setup])

    def test_debug_bootstrap_scopes_audit_and_defers_execution_policy(self):
        guide = BOOTSTRAP.read_text()
        self.assertIn(
            "gh aw audit 'https://github.com/OWNER/REPO/actions/runs/RUN_ID'"
            " --no-baseline --timeout 2m --output AUDIT_DIR --json",
            guide,
        )
        for required in (
            "--repo HOST/OWNER/REPO",
            "key_findings[].code",
            "playbook_url",
            "report is incomplete",
            "untrusted evidence",
            "never dispatch a run to obtain evidence",
            "gh aw compile WORKFLOW --dry-run",
            "does not execute the workflow",
            "explicit authorization",
            "exact",
            "reviewed published source/lock revision",
            "explicit no-dispatch rules",
        ):
            with self.subTest(contract=required):
                self.assertIn(required, guide)
        for obsolete in ("curl -sL", "| bash", "git push", "## Step 3: Apply Fixes"):
            self.assertNotIn(obsolete, guide)
        # Detailed model guidance belongs to the linked canonical guide.
        self.assertNotIn("## Model and engine misconfiguration\n", guide)
        self.assertLessEqual(len(guide.split()), 1600)

    def test_canonical_audits_preserve_bootstrap_run_identity(self):
        guide = DEBUG_GUIDE.read_text()
        evidence = guide.split("## Collect Existing Evidence\n", 1)[1].split("\n## ", 1)[0]
        self.assertIn("do not reduce it to a bare ID", evidence)
        self.assertIn("including", evidence)
        self.assertIn("step fragment", evidence)
        audit_commands = re.findall(r"gh aw audit ([^\n`]+)", evidence)
        self.assertGreaterEqual(len(audit_commands), 4)
        for command in audit_commands:
            with self.subTest(command=command):
                self.assertTrue('"$RUN_URL"' in command or "--repo HOST/OWNER/REPO" in command)
        self.assertIn("gh run view RUN_ID --repo HOST/OWNER/REPO", evidence)

    def test_debug_bootstrap_links_resolve(self):
        guide = BOOTSTRAP.read_text()
        links = re.findall(r"\]\(([^)]+)\)", guide)
        self.assertTrue(links)
        for link in links:
            with self.subTest(link=link):
                path, _, anchor = link.partition("#")
                target = BOOTSTRAP.parent / path
                self.assertTrue(target.is_file())
                if anchor:
                    headings = re.findall(r"^#+ (.+)$", target.read_text(), re.M)
                    self.assertIn(anchor, [heading.lower().replace(" ", "-") for heading in headings])
        self.assertIn(".github/aw/debug-playbook.md", links)
        self.assertIn(".github/aw/analyzing-runs.md", links)
        self.assertIn(f".github/aw/debug-agentic-workflow.md{ANCHOR}", links)

    def test_debug_inventory_preserves_all_debug_named_markdown(self):
        inventory = BOOTSTRAP.read_text().split("## Debugging Entry Point Inventory\n", 1)[1]
        links = re.findall(r"\]\(([^)#]+)(?:#[^)]*)?\)", inventory)
        documented = {(ROOT / link).resolve() for link in links}
        candidates = [ROOT / "debug.md"]
        for directory in (".github/aw", ".github/workflows/shared", "docs", "scratchpad", ".changeset"):
            candidates.extend((ROOT / directory).rglob("*debug*.md"))
        for path in candidates:
            with self.subTest(path=path.relative_to(ROOT)):
                self.assertIn(path.resolve(), documented)
        self.assertIn("not a session bootstrap", inventory)
        self.assertIn("Historical design/release records", inventory)

    def test_debug_entrypoints_delegate_without_dropping_existing_paths(self):
        for case in FIXTURES["bootstrap_routing"]:
            with self.subTest(case=case["id"]):
                entry = ROOT / case["entrypoint"]
                self.assertTrue(entry.is_file())
                text = entry.read_text()
                for route in case["route_mentions"]:
                    self.assertIn(route, text)
        for path in (
            SKILL / "SKILL.md",
            ROOT / "pkg/cli/data/agentic_workflows_skill.md",
            ROOT / ".github/agents/agentic-workflows.md",
            ROOT / "pkg/cli/data/agentic_workflows_agent.md",
        ):
            with self.subTest(router=path):
                self.assertIn("https://raw.githubusercontent.com/github/gh-aw/main/debug.md", path.read_text())
        self.assertEqual(
            (ROOT / ".github/agents/agentic-workflows.md").read_text(),
            (ROOT / "pkg/cli/data/agentic_workflows_agent.md").read_text(),
        )
        template = (ROOT / "pkg/cli/data/agentic_workflows_skill.md").read_text()
        installed = (SKILL / "SKILL.md").read_text()
        before, after = template.split("{{AW_FILE_LIST}}")
        self.assertTrue(installed.startswith(before))
        self.assertTrue(installed.endswith(after))

    def test_debugging_skill_delegates_execution_policy(self):
        guide = (ROOT / ".github/skills/debugging-workflows/SKILL.md").read_text()
        preamble = guide.split("## Table of Contents", 1)[0]
        for route in (
            "../../../debug.md",
            "../../aw/debug-agentic-workflow.md",
            "#test-workflow-changes-from-a-branch",
            "#session-authorization",
            "#dry-run-dangerous-features",
            "#explain-blocked-or-unavailable-debugging",
        ):
            with self.subTest(route=route):
                self.assertIn(route, preamble)
        self.assertLessEqual(len(preamble.split()), 250)

    def test_audit_search_distinguishes_evidence_and_failure_clusters(self):
        guide = (ROOT / ".github/aw/analyzing-runs.md").read_text()
        search = guide.split("### Retrieve and cluster session evidence\n", 1)[1].split("\n## ", 1)[0]
        for required in (
            "--search-limit 5 --json",
            "usage/aw_session.jsonl",
            "not synthesized `ProcessedRun`",
            "bm25_score",
            "anomaly_score",
            "Grouping is automatic; do not configure Drain3",
            "never templates or anomaly scores",
            "successes and failures remain separate",
            "Ungrouped results omit",
            "fleet `failure_clusters`",
            "`audit --group`",
            "Never join Drain3 cluster IDs across runs",
            "Zero search",
            "do not suppress findings",
        ):
            with self.subTest(search_contract=required):
                self.assertIn(required, search)

    def test_docs_merged_mode_scopes_base_checkout_and_recursion(self):
        guide = (ROOT / ".github/aw/documentation-workflows.md").read_text()
        example = re.search(r"```aw\n(.*?)\n```", guide, re.S).group(1)
        condition = next(line for line in example.splitlines() if line.startswith("if: "))
        for required in (
            "github.event.pull_request.merged == true",
            "github.event.pull_request.base.ref == github.event.repository.default_branch",
            "!startsWith(github.event.pull_request.head.ref, 'automation/docs-pages/')",
            "!startsWith(github.event.pull_request.head.ref, 'automation/docs-indexes')",
        ):
            with self.subTest(merged_condition=required):
                self.assertIn(required, condition)
        self.assertIn("branch-prefix: automation/docs-pages/", example)
        self.assertIn("checkout:\n  ref: ${{ github.event.repository.default_branch }}\n  pull-request: false\n", example)
        self.assertNotIn("branch-prefix: automation/docs-\n", example)

    def test_docs_on_demand_narrows_fallback_and_documents_app_scopes(self):
        guide = (ROOT / ".github/aw/documentation-workflows.md").read_text()
        variant = guide.split("### On-demand rewrite\n", 1)[1].split("\n## ", 1)[0]
        for required in (
            "merged-mode `checkout` override (retain current-PR checkout)",
            "`push-to-pull-request-branch.fallback-as-pull-request: false`",
            "`add-comment.issues: false`",
            "`report-incomplete.create-issue: false`",
            "Keep branch-protection preflight enabled",
        ):
            with self.subTest(on_demand_config=required):
                self.assertIn(required, variant)
        for required in (
            "Administration read for default push branch-protection preflight",
            "Issues read for diagnostic handlers",
            "no Issues write with `add-comment.issues: false`",
            "Default `add-comment` otherwise requests Issues write even for PR events",
            "Issues write for issue output/default create-PR fallback",
        ):
            with self.subTest(app_permissions=required):
                self.assertIn(required, guide)

    def test_docs_on_demand_completion_requires_final_head_and_human_event(self):
        guide = (ROOT / ".github/aw/documentation-workflows.md").read_text()
        variant = guide.split("### On-demand rewrite\n", 1)[1].split("\n## ", 1)[0]
        scenarios = {
            "safe_output_and_empty_commit_invalidate_prior_resolution": (
                "After safe-output push and any CI-triggering empty commit finish",
                "read live PR head",
                "exact final SHA + explanation",
                "A later commit requires renewed review/resolution",
                "never reuse the pre-push SHA",
            ),
            "agent_cannot_complete_its_own_page_edits": (
                "Completion owner: maintainer, not agent",
                "review pages, apply `docs:complete`",
                "Require maintainer follow-up in the agent summary",
                "agent must not mark itself complete or write the completion resolution",
            ),
            "completion_requires_supported_event_and_current_head_check": (
                "Human PR-body edit emits subscribed `edited` for the final head",
                "verify the required gate ran and passed on that head",
                "`GITHUB_TOKEN` metadata alone cannot refresh it",
                "an old-event rerun or earlier green check is not completion",
            ),
        }
        for scenario, requirements in scenarios.items():
            with self.subTest(scenario=scenario):
                for requirement in requirements:
                    self.assertIn(requirement, variant)

    def test_docs_index_builder_publishes_validated_final_head_through_app_event(self):
        guide = (ROOT / ".github/aw/documentation-workflows.md").read_text()
        indexes = guide.split("## Indexes: deterministic, post-merge\n", 1)[1].split("\n## ", 1)[0]
        scenarios = {
            "trusted_builder_owns_output_validation_not_page_review": (
                "completion owner: trusted deterministic builder, not docs agent",
                "ordinary Actions metadata step",
                "scoped builder App token",
                "require successful deterministic generation/validation",
                "read live `automation/docs-indexes` PR/head",
                "verify base is default branch",
                "diff contains only allowed generated outputs from the trusted generation",
                "not agent page review or human approval",
            ),
            "final_head_receives_one_decision_and_resolution": (
                "After the PR writer finishes all pushes/empty commits",
                "set `docs:required` + `docs:complete`, remove `docs:not-needed`",
                "replace exactly one `Docs-resolution:` body line",
                "preserve other body text",
                "exact final head SHA + source-content digest + generation/validation explanation",
            ),
            "no_op_failure_or_head_race_cannot_complete_new_head": (
                "No generated diff and no existing PR: no completion writes",
                "unchanged existing PR: reconcile metadata without a new commit",
                "If validation fails or head changes before publication",
                "stop without marking the new head complete",
                "regenerate/revalidate before renewal",
            ),
            "app_event_refreshes_normal_gate_without_bypass": (
                "Write resolution last via App PR-body edit",
                "subscribed `edited` reevaluates the normal gate against that head",
                "Verify required check ran/passed for the same live SHA",
                "stale checks or `GITHUB_TOKEN`-only metadata are not completion",
                "same decision/current-head gate, no index-PR exception",
            ),
        }
        for scenario, requirements in scenarios.items():
            with self.subTest(scenario=scenario):
                for requirement in requirements:
                    self.assertIn(requirement, indexes)
        self.assertLess(indexes.index("After the PR writer finishes"), indexes.index("After verification"))
        self.assertLess(indexes.index("After verification"), indexes.index("Write resolution last"))

    def test_docs_not_needed_stops_before_page_writes(self):
        guide = (ROOT / ".github/aw/documentation-workflows.md").read_text()
        prompt = re.search(r"```aw\n(.*?)\n```", guide, re.S).group(1)
        self.assertRegex(prompt, r"(?m)^docs:not-needed: no changes; stop\.$")
        self.assertIn("Only docs:required:\n", prompt)
        self.assertLess(prompt.index("docs:not-needed:"), prompt.index("Only docs:required:"))
        required = prompt.split("Only docs:required:\n", 1)[1]
        for action in ("Update required pages", "Open docs-only PR"):
            with self.subTest(required_action=action):
                self.assertNotIn(action, prompt.split("Only docs:required:\n", 1)[0])
                self.assertIn(action, required)

    def test_docs_on_demand_rewrites_prompt_and_outputs(self):
        guide = (ROOT / ".github/aw/documentation-workflows.md").read_text()
        self.assertIn("### On-demand rewrite\n", guide)
        variant = guide.split("### On-demand rewrite\n", 1)[1].split("\n## ", 1)[0]
        for required in (
            "remove merged condition, `create-pull-request`, `create-issue`",
            "`push-to-pull-request-branch` (scope: `allowed-files`) + `add-comment`",
            "read current triggering PR, not merged PR",
            "docs:not-needed: no changes; stop",
            "Missing/conflicting/unsafe: comment; stop",
            "Only docs:required: edit allowed docs; push to current PR branch",
            "Remove merged-PR reads, new-docs-PR creation, issue escalation",
            "Never request undeclared outputs",
            "extra outputs only if explicitly configured",
        ):
            with self.subTest(on_demand_contract=required):
                self.assertIn(required, variant)

    def test_live_test_branch_notification(self):
        for path in (
            DEBUG_GUIDE,
            ROOT / ".github/aw/debug-security-review.md",
        ):
            with self.subTest(path=path):
                text = path.read_text()
                self.assertRegex(text, r"[Nn]otify the user before")
                self.assertRegex(text, r"published feature branch")
                self.assertRegex(text, r"required to live test")
                self.assertIn("exact", text)
                self.assertIn("commit", text)
                self.assertIn("A draft PR is recommended, not required,", text)
                self.assertIn("and a merge is not required.", text)

    def test_debug_playbook_agent_consumption_budget(self):
        guide = (ROOT / ".github/aw/debug-playbook.md").read_text()
        self.assertLessEqual(len(guide.split()), 2800)
        self.assertLessEqual(len(guide.encode()), 26000)
        self.assertIn("Read only the linked", guide)
        for variable in ("RUN_URL", "REPO", "RUN_ID", "MODEL", "PROVIDER", "BASE_RUN_ID"):
            with self.subTest(variable=variable):
                self.assertIn(f"${{{variable}:?", guide)
        self.assertNotRegex(guide, r"\b(?:RUN_ID|BASE_RUN_ID)=\d+")
        self.assertNotRegex(guide, r"(?m)^MODEL=[a-z0-9]")
        start = guide.split("## Start here\n", 1)[1].split("\n## ", 1)[0]
        self.assertIn('gh aw audit "$RUN_URL" --no-baseline --timeout 2m', start)
        self.assertIn("HOST/OWNER/REPO", start)
        self.assertIn("including its fragment", start)
        self.assertNotIn('gh aw audit "$RUN_ID"', start)

    def test_diagnostic_routing_mini_eval_contract(self):
        guide = (ROOT / ".github/aw/debug-playbook.md").read_text()
        for case in FIXTURES["diagnostic_routing"]:
            with self.subTest(case=case["id"]):
                section = guide.split(f"## {case['section']}\n", 1)[1].split("\n## ", 1)[0]
                row = next(line for line in section.splitlines() if f"`{case['code']}`" in line)
                if "next_guide" in case:
                    self.assertIn(case["next_guide"], section)
                    self.assertTrue((ROOT / ".github/aw" / case["next_guide"]).is_file())
                if "next_section" in case:
                    anchor = case["next_section"].lower().replace(" ", "-")
                    self.assertIn(f"(#{anchor})", row)
                    self.assertIn(f"## {case['next_section']}\n", guide)
                if "required" in case:
                    self.assertIn(case["required"], row)

    def test_debug_playbook_covers_diagnostic_codes(self):
        guide = (ROOT / ".github/aw/debug-playbook.md").read_text()
        self.assertNotRegex(guide, r"(?m)^Sources?:")
        self.assertNotRegex(guide, r"\.\./\.\./(?:pkg|actions|specs)/")
        self.assertNotRegex(guide, r"\.(?:go|cjs|mjs|js|ts|tsx|jsx|py)\b")
        rows = {}
        for line in guide.splitlines():
            if line.startswith("| ") and line.count("|") == 4:
                codes, evidence, remedy = line.strip("|").split("|")
                if evidence.strip() and remedy.strip():
                    for code in re.findall(r"`([^`]+)`", codes):
                        rows[code] = line
        sources = [
            *sorted((ROOT / "actions/setup/js").glob("*.cjs")),
            *sorted((ROOT / "pkg/workqueue").glob("*.go")),
        ]
        code_patterns = (
            r'\b(?:code|reason_code|failure_code|error_code|reasonCode|errorCode)\s*:\s*["\']([A-Za-z][A-Za-z0-9_-]+)["\']',
            r'\bqueueError\(\s*["\']([a-z][a-z0-9_-]+)["\']',
            r'\bE\d{3}\b',
        )
        for path in sources:
            if ".test." in path.name or path.name.endswith(("_test.go", "_checks.cjs")) or "fixture" in path.name:
                continue
            for pattern in code_patterns:
                for code in set(re.findall(pattern, path.read_text())):
                    with self.subTest(path=path.name, code=code):
                        self.assertTrue(code in rows, f"{code} needs an evidence/remedy row")
        registries = (
            ("pkg/cli/audit_finding_codes.go", r'AuditFindingCode = "([^"]+)"'),
            ("actions/setup/js/error_codes.cjs", r'const [A-Z_0-9]+ = "([^"]+)"'),
            ("actions/setup/js/agent_error_patterns.cjs", r'`([a-z_]+)=\$\{'),
            ("actions/setup/js/agent_execution.cjs", r'categorySet.add\("([^"]+)"\)'),
            ("actions/setup/js/harness_error_patterns.cjs", r'guard: "([^"]+)"'),
        )
        for name, pattern in registries:
            for code in re.findall(pattern, (ROOT / name).read_text()):
                if code != "missing_model_pricing_model_name":
                    with self.subTest(registry=name, code=code):
                        self.assertTrue(code in rows, f"{code} needs an evidence/remedy row")
        causes = (ROOT / "actions/setup/js/empty_output_outcome.cjs").read_text()
        for block in re.findall(r"const EMPTY_OUTPUT_(?:FAILURE_)?CAUSES = Object.freeze\(\{(.*?)\}\)", causes, re.S):
            for code in re.findall(r"^\s+([a-z_]+):", block, re.M):
                self.assertTrue(code in rows, f"{code} needs an evidence/remedy row")
        self.assertIn("debug-playbook.md", DEBUG_GUIDE.read_text())
        for required in ("gh aw audit", "--no-baseline", "gh aw models", "--refresh-observed=false",
                         "Do not dispatch workflows", "first failing boundary", "Unmapped codes"):
            self.assertIn(required, guide)

    def test_model_availability_playbook(self):
        name = "debug-playbook.md"
        guide = (ROOT / ".github/aw" / name).read_text()
        for path in (DEBUG_GUIDE, GUIDES[0]):
            with self.subTest(path=path):
                self.assertIn(f"{name}#model-availability", path.read_text())
        for required in (
            "models_fetch_complete", "routing_models",
            "supported_endpoints", "model_aliases", "awf-reflect.json",
            "token-usage.jsonl", "gh run download", "gh run view",
            "gh aw models", "--refresh-observed=false", "--logs-dir",
            "gh aw audit", "--no-baseline", "--timeout 2m",
            'gh aw audit "$BASE_RUN_ID" "$RUN_ID"', "incomplete JSON report",
            "model_unavailable", "model_availability_unknown",
            "Availability is unknown", "empty file", "missing file",
            "same AWF version", "credential source (names only)",
            "not proof", "Do not dispatch workflows",
        ):
            with self.subTest(required=required):
                self.assertIn(required, guide)
        routing = DEBUG_GUIDE.read_text()
        for required in (
            "key_findings[].code", "playbook_url",
            'gh aw audit "$RUN_URL" --no-baseline --timeout 2m --output AUDIT_DIR --json',
            "model_unavailable", "model_availability_unknown",
            "cached `gh aw models`", "consolidated", "partial JSON",
        ):
            with self.subTest(routing=required):
                self.assertIn(required, routing)
        self.assertNotIn("debug-model-not-available.md", routing)

    def test_public_instruction_download_fallback(self):
        for path in (
            SKILL / "SKILL.md",
            ROOT / "pkg/cli/data/agentic_workflows_skill.md",
            ROOT / ".github/agents/agentic-workflows.md",
            ROOT / "pkg/cli/data/agentic_workflows_agent.md",
        ):
            with self.subTest(path=path):
                text = path.read_text()
                for required in (
                    "HTTP 403",
                    "SAML-authorized for a different organization",
                    "do not retry with the same token",
                    "https://raw.githubusercontent.com/github/gh-aw/main/<path>",
                    "curl --fail --silent --show-error --location",
                    "referenced instructions and `SKILL.md` files",
                    "never send an Authorization header or token",
                    "report that error",
                    "not private or enterprise repositories",
                ):
                    self.assertIn(required, text)

    def test_analyzing_runs_routes_and_evidence_contract(self):
        guide = (ROOT / ".github/aw/analyzing-runs.md").read_text()
        router = (SKILL / "SKILL.md").read_text()
        overlay = (ROOT / ".github/aw/instructions.md").read_text()
        agent = (ROOT / ".github/agents/agentic-workflows.md").read_text()
        embedded_agent = (ROOT / "pkg/cli/data/agentic_workflows_agent.md").read_text()
        for text in (router, overlay, agent, embedded_agent):
            self.assertIn("analyzing-runs.md", text)
            self.assertIn("failure clustering", text)
        for required in (
            "gh aw logs", "gh aw audit", "audit-diff",
            "failure_investigation.selected_runs", "failure_clusters",
            "failure_boundary", "aftermath", "signal.job", "signal.step",
            "representative_runs", "audit_path", "logs_path", "metadata_only",
            "Intentional failures", "unique run IDs", "common root cause",
            "end_date", "before_run_id", "continuations", "recompute clusters",
            "prune cache records", "Missing evidence", "successful",
            "debug-agentic-workflow.md", "optimize-agentic-workflow.md",
            "Do not dispatch workflows",
        ):
            with self.subTest(required=required):
                self.assertIn(required, guide)

    def test_no_network_guidance_for_read_only_workflows(self):
        constraints = (ROOT / ".github/aw/workflow-constraints.md").read_text()
        creator = (ROOT / ".github/aw/create-agentic-workflow.md").read_text()
        for text in (constraints, creator):
            with self.subTest(guide=text[:80]):
                self.assertIn("network.allowed: []", text)
                self.assertIn("gh-proxy", text)
        self.assertIn("network: defaults", constraints)
        self.assertIn("policy/compliance", creator)
        self.assertIn("structurally", creator)

    def test_persona_rotation_finishes_remaining_personas(self):
        explorer = (ROOT / ".github/workflows/agent-persona-explorer.md").read_text()
        self.assertIn("If 1 or 2 remain, select those before resetting", explorer)
        self.assertIn("If the updated list now contains all 9 personas, reset to an empty list", explorer)
        self.assertNotIn("If fewer than 3 remain, reset", explorer)

    def test_security_reporting_omits_empty_findings_tables(self):
        for path in (
            ROOT / ".github/aw/debug-security-review.md",
            ROOT / ".github/skills/developer-security/SKILL.md",
        ):
            with self.subTest(path=path):
                text = path.read_text()
                self.assertIn(
                    "Show a findings table only when confirmed findings exist.", text
                )
                self.assertIn("No confirmed vulnerabilities found", text)
                self.assertIn("placeholder row", text)
                self.assertIn("final replies and task-completion summaries", text)
                self.assertIn("findings table when empty, including its header", text)
                self.assertIn("sentence in a table row", text)
        gate = (ROOT / ".github/aw/debug-security-review.md").read_text()
        self.assertIn("without labeling them as vulnerabilities", gate)
        self.assertIn("rendering an empty overview table", gate)

    def test_single_security_reviewer_and_markdown_overview(self):
        gate = (ROOT / ".github/aw/debug-security-review.md").read_text()
        self.assertIn("exactly one read-only security-review agent", gate)
        self.assertIn("same reviewer", gate)
        self.assertIn("every question has an evidence-backed YES", gate)
        self.assertIn("## Single Reviewer", gate)
        self.assertIn("### Annotated Markdown Change Overview", gate)
        self.assertIn("Ignore generated `.lock.yml` files in", gate)
        self.assertIn("Generated locks remain execution-review evidence", gate)
        self.assertIn("Show it in the user response", gate)
        self.assertIn("| Markdown source | Change | Security impact |", gate)
        for obsolete in (
            "Launch two independent", "third independent cheap judge",
            "two evidence-backed YES votes", "two-affirmative-vote requirement",
        ):
            self.assertNotIn(obsolete, gate)
        for skill in ("debugging-workflows", "review-agentic-workflows"):
            with self.subTest(skill=skill):
                text = (ROOT / f".github/skills/{skill}/SKILL.md").read_text()
                self.assertIn("debug-security-review.md#single-reviewer", text)
                self.assertIn(
                    "debug-security-review.md#annotated-markdown-change-overview", text
                )

    def test_customer_case_remedy_order(self):
        case = FIXTURES["model_mismatch"]
        self.assertIn("v0.89.21", case["lock_header"])
        self.assertIn("model: auto", case["workflow"])
        self.assertIn("gpt-5.6-luna-utility", case["agent_stdio"])
        self.assertIn("Cannot translate Copilot request feature", case["agent_stdio"])
        for path in GUIDES:
            with self.subTest(path=path):
                text = " ".join(checklist(path.read_text()).split())
                positions = [text.index(fix) for fix in case["expected_fixes"]]
                self.assertEqual(positions, sorted(positions))
                self.assertLess(positions[-1], text.index(case["last_resort"]))
                self.assertLess(text.index(case["last_resort"]), text.index("model: gpt-4.1"))
                for required in (
                    "v0.89.21", "64177", "compiler_version", "cli_version",
                    "including prereleases", "latest non-prerelease",
                    "gh extension install github/gh-aw --force --pin",
                    "underlying misconfiguration in place",
                ):
                    self.assertIn(required, " ".join(text.split()))

    def test_wire_api_and_artifact_contract(self):
        for path in GUIDES:
            with self.subTest(path=path):
                text = " ".join(checklist(path.read_text()).split())
                precedence = text[text.index("precedence"):] if "precedence" in text else text[text.index("order:"):]
                positions = [precedence.index(term) for term in (
                    "override", "wire_api", "-utility", "gpt-5+", "CLI default",
                )]
                self.assertEqual(positions, sorted(positions))
                for required in (
                    "/responses", "/chat/completions", "whole session",
                    "sub-agents", "67460", "5103", "/reflect",
                    "supported_endpoints", "Model endpoint mismatch:",
                    "engine.copilot-sdk: true",
                    "Cannot translate Copilot request feature",
                    "Unsupported Responses custom tool", "model_policy_violation",
                    "not accessible via the … endpoint",
                    "agent-stdio.log", "[copilot-harness]",
                    "sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl",
                    "requested_model", "awf_version", "aw_info.json", "gh aw audit RUN_ID",
                ):
                    self.assertIn(required, text)
                silent_case = FIXTURES["silent_subagent_fallback"]
                for required in (
                    silent_case["awf_version"], silent_case["error"],
                    silent_case["event"], silent_case["event_file"],
                    silent_case["audit_finding"], silent_case["request_status"],
                ):
                    self.assertIn(required, text)
                self.assertLess(
                    text.index("supported_endpoints"),
                    text.index("For the normal Copilot harness path")
                    if "For the normal Copilot harness path" in text
                    else text.index("normal CLI wire-API precedence"),
                )

    def test_misplaced_version_fields(self):
        case = FIXTURES["misplaced_versions"]
        for path in GUIDES:
            with self.subTest(path=path):
                text = checklist(path.read_text())
                for failure in case["failures"]:
                    field_start = text.index(f"`{failure['field']}`")
                    boundary = text.find("\n", field_start)
                    self.assertIn(failure["step"], text[field_start:boundary])
                self.assertIn("1.0.x", text)
                self.assertIn("vX.Y.Z", text)
                self.assertIn("gh-aw-firewall", text)
                self.assertIn(f"There is no `{case['invalid_field']}` field", text)
                self.assertIn("remove", text)
                self.assertIn("compiled default", text)

    def test_routing_and_evidence_links(self):
        # SKILL.md is regenerated from the embedded template by `gh aw init`,
        # so both must carry the routing line.
        for path in (SKILL / "SKILL.md", ROOT / "pkg/cli/data/agentic_workflows_skill.md"):
            with self.subTest(path=path):
                lines = [line for line in path.read_text().splitlines() if ANCHOR in line]
                self.assertTrue(lines, f"{path} must route to {ANCHOR}")
                line = lines[0]
                self.assertIn(
                    f"`.github/aw/debug-agentic-workflow.md{ANCHOR}`",
                    line,
                )
                self.assertNotIn("https://github.com/github/gh-aw/blob/main/", line)
                self.assertNotIn("(../../aw/", line)
                for symptom in (
                    "AWF model/endpoint 400s",
                    "silent cross-family sub-agent failures",
                    "`model: auto` failures",
                    "install-step 404s after a version pin",
                    "version fields",
                ):
                    self.assertIn(symptom, line)
        self.assertTrue((SKILL / "../../aw/debug-agentic-workflow.md").resolve().is_file())
        full = DEBUG_GUIDE.read_text()
        self.assertIn(
            "(debug-model-engine-misconfiguration.md)", checklist(full)
        )
        for section in ("Collect Existing Evidence", "Identify the First Failing Boundary"):
            content = full.split(f"## {section}\n", 1)[1].split("\n## ", 1)[0]
            self.assertIn(ANCHOR, content)
        self.assertIn(f".github/aw/debug-agentic-workflow.md{ANCHOR}", BOOTSTRAP.read_text())

    def test_docs_merged_example_labels_search_best_effort(self):
        guide = (ROOT / ".github/aw/documentation-workflows.md").read_text()
        example = re.search(r"```aw\n(.*?)\n```", guide, re.S).group(1)
        self.assertIn("best-effort duplicate avoidance only, not exactly-once", guide)
        self.assertIn("Best-effort: check linked docs PRs", example)
        self.assertIn("including closed PRs", example)
        self.assertIn("Concurrent runs can still create duplicates", example)
        self.assertNotIn("existing PR: stop, no duplicate", example)
        self.assertIn("`branch-prefix` only namespaces output branches", guide)
        self.assertIn("merge-SHA branch is not enforced idempotency", guide)

    def test_docs_strict_dedup_requires_durable_atomic_claim_and_recovery(self):
        guide = (ROOT / ".github/aw/documentation-workflows.md").read_text()
        contract = guide.split("- Strict merged-run deduplication:", 1)[1].split(
            "- Agent cannot set its own completion label", 1
        )[0]
        for required in (
            "repository-owned trusted orchestration, not model instructions",
            "Before inference/outputs, atomically create a durable claim",
            "repository + workflow purpose + source merge SHA",
            "only the winning owner proceeds",
            "Existing claim: stop, whether pending or complete",
            "read-then-write record is not atomic",
            "completion + output PR URL using owner-checked compare-and-set",
            "retain records after PR closure/branch deletion",
            "Crash or uncertain output creation: fail closed",
            "reconcile actual outputs before any explicit recovery",
            "Never expire/reassign a claim blindly or retry an ambiguous creation",
            "Claim + PR creation are not one transaction",
            "not an exactly-once guarantee",
            "The example does not implement that ledger",
        ):
            with self.subTest(dedup_contract=required):
                self.assertIn(required, contract)


if __name__ == "__main__":
    unittest.main()
