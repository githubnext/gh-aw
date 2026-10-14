"""Exercise the Node image publisher's actual decision script."""

import os
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


WORKFLOW = (
    Path(__file__).resolve().parent.parent
    / ".github/workflows/publish-safe-outputs-node.yml"
)


class PublishSafeOutputsNodeTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.source = WORKFLOW.read_text()
        cls.decision = cls.source.split(
            "      - name: Decide whether to publish\n", 1
        )[1].split("\n      - name:", 1)[0]
        cls.script = textwrap.dedent(cls.decision.split("        run: |\n", 1)[1])

    def decide(self, event="push", **overrides):
        env = {
            **os.environ,
            "COOLDOWN_DAYS": "1",
            "CURRENT_BASE_DIGEST": "sha256:base",
            "CURRENT_DOCKERFILE_HASH": "source",
            "EXISTING_BASE_DIGEST": "sha256:base",
            "EXISTING_DOCKERFILE_HASH": "source",
            "IMAGE_EXISTS": "true",
            "IN_COOLDOWN": "false",
            "TRIGGER_EVENT": event,
            **overrides,
        }
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "output"
            env["GITHUB_OUTPUT"] = str(output)
            result = subprocess.run(
                ["bash", "-c", self.script],
                env=env,
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            return dict(line.split("=", 1) for line in output.read_text().splitlines())

    def test_event_is_wired_from_github_context(self):
        self.assertIn("TRIGGER_EVENT: ${{ github.event_name }}", self.decision)

    def test_unchanged_push_skips_publication(self):
        self.assertEqual(self.decide()["should_push"], "false")

    def test_scheduled_and_manual_runs_refresh_unchanged_packages(self):
        for event in ("schedule", "workflow_dispatch"):
            with self.subTest(event=event):
                result = self.decide(event)
                self.assertEqual(result["should_push"], "true")
                self.assertIn("package", result["reason"])

    def test_changed_base_or_source_and_missing_image_still_publish(self):
        for overrides in (
            {"EXISTING_BASE_DIGEST": "sha256:old"},
            {"EXISTING_DOCKERFILE_HASH": "old"},
            {"IMAGE_EXISTS": "false", "EXISTING_BASE_DIGEST": ""},
        ):
            with self.subTest(overrides=overrides):
                self.assertEqual(self.decide(**overrides)["should_push"], "true")

    def test_cooldown_blocks_new_base_on_all_events(self):
        for event in ("push", "schedule", "workflow_dispatch"):
            for exists in ("true", "false"):
                with self.subTest(event=event, exists=exists):
                    result = self.decide(
                        event,
                        IN_COOLDOWN="true",
                        IMAGE_EXISTS=exists,
                        EXISTING_BASE_DIGEST="sha256:old",
                    )
                    self.assertEqual(result["should_push"], "false")
                    self.assertIn("cooldown", result["reason"])

    def test_cooldown_allows_refresh_of_already_published_base(self):
        self.assertEqual(
            self.decide("schedule", IN_COOLDOWN="true")["should_push"], "true"
        )

    def test_build_refreshes_layers_and_preserves_both_platforms(self):
        build = self.source.split("      - name: Build and push gh-aw-node\n", 1)[
            1
        ].split("\n      - name:", 1)[0]
        self.assertRegex(build, r"(?m)^\s+no-cache: true$")
        self.assertRegex(build, r"(?m)^\s+pull: true$")
        self.assertIn("platforms: linux/amd64,linux/arm64", build)
        self.assertIn("if: steps.decision.outputs.should_push == 'true'", build)


if __name__ == "__main__":
    unittest.main()
