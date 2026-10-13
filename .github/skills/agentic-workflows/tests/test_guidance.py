"""Documentation contract regressions, not an LLM or live workflow evaluation."""

import json
from pathlib import Path
import unittest


SKILL = Path(__file__).resolve().parents[1]
ROOT = SKILL.parents[2]
DEBUG_GUIDE = ROOT / ".github/aw/debug-agentic-workflow.md"
GUIDES = (ROOT / ".github/aw/debug-model-engine-misconfiguration.md", ROOT / "debug.md")
FIXTURES = json.loads((SKILL / "tests/fixtures.json").read_text())
ANCHOR = "#model-and-engine-misconfiguration"


def checklist(text):
    if "## Model and engine misconfiguration\n" not in text:
        return text
    return text.split("## Model and engine misconfiguration\n", 1)[1].split("\n## ", 1)[0]


class GuidanceTests(unittest.TestCase):
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
                    boundary = text.find("\n", field_start) if path == GUIDES[0] else text.index("point", field_start)
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
                self.assertIn(f"(../../aw/debug-agentic-workflow.md{ANCHOR})", line)
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
        self.assertIn(f".github/aw/debug-agentic-workflow.md{ANCHOR}", GUIDES[1].read_text())


if __name__ == "__main__":
    unittest.main()
