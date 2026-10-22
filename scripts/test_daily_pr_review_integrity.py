"""Regression checks for the daily PR review's integrity and coverage contract."""

import re
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
SOURCE = ROOT / ".github/workflows/daily-pr-review-cursor.md"
LOCK = ROOT / ".github/workflows/daily-pr-review-cursor.lock.yml"


class DailyPRReviewIntegrityTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.source = SOURCE.read_text()
        cls.frontmatter = cls.source.split("---", 2)[1]
        cls.prompt = cls.source.split("---", 2)[2]
        cls.text = re.sub(r"\s+", " ", cls.prompt)

    def test_approved_integrity_remains_explicit_without_trust_promotion(self):
        self.assertIn("    min-integrity: approved\n", self.frontmatter)
        for field in ("trusted-users:", "approval-labels:", "lockdown:", "integrity-proxy:"):
            self.assertNotIn(field, self.frontmatter)
        self.assertIn("strict: true\n", self.frontmatter)
        self.assertIn("    id: awf\n", self.frontmatter)

    def test_candidate_selection_is_bounded_and_metadata_is_not_authorization(self):
        for setting in ("state: open", "sort: created", "direction: desc", "perPage: 5"):
            self.assertIn(setting, self.prompt)
        self.assertIn("Select at most these five returned candidates", self.text)
        self.assertIn("not proof that diff content passes the guard", self.text)
        self.assertIn("do not expand the search to work around a denial", self.text)

    def test_integrity_denial_stops_repeated_probes_without_fabricating_denials(self):
        self.assertIn("making at most one", self.text)
        self.assertIn("`method: get_diff`", self.prompt)
        self.assertIn("Stop content reads on the first integrity-filtering denial", self.text)
        self.assertIn("Do not attempt the remaining candidates after that denial", self.text)
        self.assertIn("mark them **Not attempted**", self.text)
        self.assertIn("not **Blocked** as though each had returned a denial", self.text)
        self.assertIn("gh-aw-mcpg v0.4.37", self.text)
        self.assertIn("meaningful diff review remains unavailable", self.text)

    def test_filtered_content_has_no_alternate_transport_or_trust_workaround(self):
        self.assertIn("On integrity filtering, do not retry", self.text)
        for transport in (
            "`get_files`",
            "other GitHub tools",
            "shell commands",
            "local checkouts",
            "raw URLs",
            "pre-fetched patches",
        ):
            self.assertIn(transport, self.text)
        self.assertIn("do not prescribe a label or membership change as a fix", self.text)
        self.assertIn("not an authentication or model failure", self.text)

    def test_reporting_distinguishes_all_four_coverage_outcomes(self):
        coverage = self.prompt.split("Assign exactly one coverage status", 1)[1].split(
            "On integrity filtering", 1
        )[0]
        self.assertEqual(
            re.findall(r"^- \*\*([^*]+)\*\*:", coverage, re.MULTILINE),
            ["Reviewed", "Blocked", "Unavailable", "Not attempted"],
        )
        self.assertIn("complete diff was returned and inspected", self.text)
        self.assertIn("incomplete, truncated, or missing content", self.text)
        self.assertIn("never describe this PR as blocked or reviewed", self.text)
        self.assertIn("unattempted candidates remain not attempted", self.text)

    def test_unread_content_cannot_receive_a_clean_quality_signal(self):
        self.assertIn("For **Reviewed** PRs only", self.text)
        self.assertIn("must be `N/A`, never zero or green", self.text)
        self.assertIn('"Code quality not assessed"', self.text)
        self.assertIn('"No issues" only for a completely inspected, clean diff', self.text)
        self.assertIn("Do not infer findings from PR titles or metadata", self.text)
        self.assertIn("do not execute PR code, tests, or builds", self.text)

    def test_partial_and_all_blocked_reviews_publish_coverage_and_remain_incomplete(self):
        self.assertIn("even when all diff reads were blocked", self.text)
        self.assertIn(
            "After the report is queued, call `report_incomplete` if any selected PR is not **Reviewed**",
            self.text,
        )
        self.assertIn("exact PR numbers grouped by coverage status", self.text)
        self.assertIn("If report creation fails, call `report_incomplete`", self.text)

    def test_noop_requires_successful_empty_discovery(self):
        self.assertIn("If the list call fails or is denied, call `report_incomplete`", self.text)
        self.assertIn("Only when discovery succeeded and returned no candidates", self.text)
        self.assertIn("No review candidates returned under the current integrity policy", self.text)
        self.assertNotIn("No open PRs to review today", self.prompt)
        self.assertIn("Do not call `noop` when candidates exist", self.text)

    def test_generated_lock_enforces_approved_integrity(self):
        lock = LOCK.read_text()
        self.assertRegex(lock, r'"min-integrity":\s*"approved"')
        self.assertNotIn('"min-integrity": "none"', lock)
        self.assertIn('"guard-policies":', lock)
        self.assertIn('"ghcr.io/github/gh-aw-mcpg:', lock)
        self.assertIn(".github/workflows/daily-pr-review-cursor.md", lock)


if __name__ == "__main__":
    unittest.main()
