import hashlib
import json
import os
from pathlib import Path
import re
import tempfile
import sys
import unittest
from unittest.mock import patch

import check
from check import MUTATIONS, ROOT, check_compiled_workflows, check_result, concretize, normalize_trace


class RuntimePinTest(unittest.TestCase):
    def test_release_pin_matches_workflow_and_documentation(self):
        expected = "0f56d6a813368c48008dfbecb23a73ab43a4a5e1bf09f026782a674aa9d4b3c9"
        self.assertEqual(check.JAR_SHA256, expected)
        workflow = (ROOT.parents[1] / ".github/workflows/daily-workflow-security-model.md").read_text()
        self.assertIn("https://github.com/tlaplus/tlaplus/releases/download/v1.8.0/tla2tools.jar",
                      workflow)
        match = re.search(
            r"printf '%s  %s\\n' \\\n\s+([a-f0-9]{64}) \\\n"
            r'\s+"\$tools/tla2tools\.jar" \| sha256sum --check --strict',
            workflow,
        )
        self.assertIsNotNone(match)
        self.assertEqual(match.group(1), expected)
        self.assertIn(expected, (ROOT / "README.md").read_text())

    def test_missing_or_mismatched_jar_fails_before_execution(self):
        for present in [False, True]:
            with self.subTest(present=present), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                jar = root / "tla2tools.jar"
                if present:
                    jar.write_bytes(b"not the pinned TLA+ runtime")
                results = root / "results"
                with patch.dict(os.environ, {"TLA2TOOLS_JAR": str(jar)}), \
                     patch.object(sys, "argv", ["check.py", "--results", str(results)]), \
                     patch.object(check, "run_case") as run_case, \
                     patch("sys.stderr"):
                    with self.assertRaises(SystemExit) as error:
                        check.main()
                self.assertEqual(error.exception.code, 2)
                run_case.assert_not_called()
                self.assertFalse(results.exists())

    def test_matching_jar_reaches_model_checker(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            jar = root / "tla2tools.jar"
            jar.write_bytes(b"local runtime fixture")
            results = root / "results"
            digest = hashlib.sha256(jar.read_bytes()).hexdigest()
            with patch.dict(os.environ, {"TLA2TOOLS_JAR": str(jar)}), \
                 patch.object(sys, "argv", ["check.py", "--results", str(results)]), \
                 patch.object(check, "JAR_SHA256", digest), \
                 patch.object(check, "run_case", side_effect=RuntimeError("fixture boundary")) as run_case:
                with self.assertRaisesRegex(RuntimeError, "fixture boundary"):
                    check.main()
            run_case.assert_called_once()
            self.assertEqual(run_case.call_args.args[1], jar.resolve())
            self.assertTrue(results.is_dir())


class ResultContractTest(unittest.TestCase):
    def test_success_requires_exhaustion(self):
        text = ("Model checking completed. No error has been found.\n"
                "10 states generated, 8 distinct states found")
        self.assertEqual(check_result(0, text), {"generated": 10, "distinct": 8})
        for status, output in [(1, text), (0, "Simulation finished")]:
            with self.assertRaises(RuntimeError):
                check_result(status, output)

    def test_counterexample_requires_exact_invariant_and_status(self):
        text = ("Error: Invariant DetectionGate is violated.\n"
                "10 states generated, 8 distinct states found")
        self.assertEqual(check_result(12, text, "DetectionGate")["distinct"], 8)
        for status, invariant in [(0, "DetectionGate"), (1, "DetectionGate"),
                                  (12, "TypeOK")]:
            with self.assertRaises(RuntimeError):
                check_result(status, text, invariant)

    def test_tool_failure_is_not_a_counterexample(self):
        with self.assertRaises(RuntimeError):
            check_result(12, "Error parsing module", "DetectionGate")

    def test_trace_shape(self):
        raw = {"counterexample": {"state": [[1, {"s": {"lastEvent": "Init"}}],
                                            [2, {"s": {"lastEvent": "Execute"}}]]}}
        trace = normalize_trace(raw)
        self.assertEqual(trace["states"][1]["s"]["lastEvent"], "Execute")
        for invalid in [{}, {"counterexample": {"state": []}},
                        {"counterexample": {"state": [[1, {}], [2, {}]]}}]:
            with self.assertRaises(RuntimeError):
                normalize_trace(invalid)

    def test_concretization_is_explicitly_synthetic(self):
        for fault, invariant in MUTATIONS.items():
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as directory:
                destination = Path(directory)
                concretize(fault, invariant, destination, {"states": [{}, {}]})
                report = json.loads((destination / "counterexample.json").read_text())
                self.assertFalse(report["confirmed_product_bug"])
                source = (destination / "source.md").read_text()
                self.assertIn("NOT a confirmed product vulnerability", source)
                if fault == "agent-write":
                    self.assertIn("contents: write", source)
                if fault == "retained-push-token":
                    self.assertIn("create-pull-request:", source)

    def test_examples_match_model_and_include_privileged_git(self):
        examples = json.loads((ROOT / "examples.json").read_text())
        self.assertEqual(examples["model_sha256"],
                         hashlib.sha256((ROOT / "CompiledWorkflow.tla").read_bytes()).hexdigest())
        self.assertFalse(examples["confirmed_product_bug"])
        push = next(example for example in examples["examples"] if example["case"] == "privileged-push")
        self.assertTrue(any(operation["job"] == "safe_outputs" and operation["op"] == "push"
                            for operation in push["final_state"]["operations"]))
        retained = next(example for example in examples["examples"]
                        if example["case"] == "failure-token-retention")
        self.assertLess(retained["events"].index("Fail:safe_outputs"),
                        retained["events"].index("AppTokenPost"))
        self.assertEqual(retained["final_state"]["status"]["safe_outputs"], "failure")
        self.assertEqual(retained["final_state"]["appTokenPost"], "complete")
        self.assertIn("app-write", retained["final_state"]["live"])

    def test_compiled_corpus_contract(self):
        for outcome in ["success", "violation", "tool-failure", "malformed-policy"]:
            with self.subTest(outcome=outcome), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                (root / "sample.lock.yml").write_text("jobs: {}\n")
                report = {"conforms": outcome == "success",
                          "detection_policy": {"mode": "enabled"}}
                if outcome == "malformed-policy":
                    report["detection_policy"] = None
                output = json.dumps(report)
                code = 0 if outcome == "success" else 1
                if outcome == "tool-failure":
                    output, code = "not-json", 2
                script = root / "verifier"
                script.write_text(f"#!{sys.executable}\nimport sys\nprint({output!r})\nsys.exit({code})\n")
                script.chmod(0o700)
                if outcome == "success":
                    check_compiled_workflows(script, root, root)
                else:
                    with self.assertRaises(RuntimeError):
                        check_compiled_workflows(script, root, root)
                saved = json.loads((root / "compiled-workflows.json").read_text())
                self.assertEqual(saved["failed"], 0 if outcome == "success" else 1)


if __name__ == "__main__":
    unittest.main()
