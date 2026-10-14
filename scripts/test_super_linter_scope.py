"""Regression checks for the Super Linter GO_MODULES directory selection."""

import re
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
SOURCE = ROOT / ".github/workflows/super-linter.md"
LOCK = ROOT / ".github/workflows/super-linter.lock.yml"


class SuperLinterScopeTests(unittest.TestCase):
    def setUp(self):
        self.source = SOURCE.read_text()
        match = re.search(r'FILTER_REGEX_EXCLUDE: "([^"\n]+)"', self.source)
        self.assertIsNotNone(match)
        self.pattern = match.group(1)

    def excluded(self, path):
        # Pinned super-linter filters full file paths before queuing their
        # containing directories in file-array-GO_MODULES.
        return re.search(self.pattern, path) is not None

    def test_deliberate_fixtures_and_empty_manifest_are_excluded(self):
        paths = [
            ".github/workflows/go.mod",
            "pkg/linters/appendbytestring/testdata/src/appendbytestring/appendbytestring.go",
            "pkg/linters/bufioscannererunchecked/testdata/src/basic/bad.go",
            "super-linter-output/report.go",
        ]
        for path in paths:
            for prefix in ("", "/github/workspace/"):
                with self.subTest(path=prefix + path):
                    self.assertTrue(self.excluded(prefix + path))
        fixtures = list((ROOT / "pkg/linters").glob("*/testdata/**/*.go"))
        self.assertGreater(len(fixtures), 0)
        for fixture in fixtures:
            self.assertTrue(self.excluded(str(fixture.relative_to(ROOT))), fixture)
        self.assertEqual(list((ROOT / ".github/workflows").rglob("*.go")), [])

    def test_real_code_tests_and_module_remain_included(self):
        paths = [
            "go.mod",
            "cmd/gh-aw/main.go",
            "pkg/logger/logger.go",
            "pkg/workflow/checkout_config_parser.go",
            "pkg/linters/appendbytestring/appendbytestring.go",
            "pkg/linters/appendbytestring/appendbytestring_test.go",
            ".github/scripts/purelock/purity_scan.go",
            "scripts/lint_error_messages_test.go",
            "pkg/linters/example/testdata_helpers.go",
            ".github/workflows/go.mod.backup",
        ]
        for path in paths:
            for prefix in ("", "/github/workspace/"):
                with self.subTest(path=prefix + path):
                    self.assertFalse(self.excluded(prefix + path))

    def test_generated_lock_preserves_scope_and_reports_failures(self):
        lock = LOCK.read_text()
        self.assertIn("FILTER_REGEX_EXCLUDE: " + self.pattern, lock)
        for text in (self.source, lock):
            self.assertIn('VALIDATE_GO_MODULES: "true"', text)
            self.assertIn("continue-on-error: true", text)
            self.assertIn("${{ steps.super-linter.outcome }}", text)
            self.assertIn("Super-linter did not produce a nonempty log", text)
            self.assertIn("/tmp/super-linter-result.txt", text)
            self.assertIn("if-no-files-found: error", text)
        self.assertNotIn("GO_CONFIG_FILE:", self.source)
        self.assertNotIn("LINTER_RULES_PATH:", self.source)
        self.assertIn("twenty examples total", self.source)
        self.assertIn("does not establish an exploitable vulnerability", self.source)
        self.assertNotIn("[Include the full linter output here]", self.source)


if __name__ == "__main__":
    unittest.main()
