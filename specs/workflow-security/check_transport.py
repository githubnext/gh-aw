#!/usr/bin/env python3
"""Check transport lifecycle profiles, negative controls, and reachability."""

import hashlib
import os
from pathlib import Path
import subprocess
import tempfile

from check import JAR_SHA256, ROOT, check_result


FAULTS = {
    "blocking-open": "NonblockingOpen",
    "reopen-original": "StableConsumption",
    "rebaseline": "OriginalBaseline",
    "skip-publication-check": "PublicationIntegrity",
    "retry-integrity": "PermanentFailure",
    "cleanup-blocked": "CleanupDisposition",
    "cleanup-error-hidden": "CleanupDisposition",
    "unsafe-diagnostics": "SafeDiagnostics",
    "redundant-hashing": "BoundedLocalCost",
}
WITNESSES = [
    "NoSuccessfulWrite", "NoRejectedMutation", "NoRateLimitRetry",
    "NoCleanupFailure", "NoConflictCleanup",
]


def main():
    jar_env = os.environ.get("TLA2TOOLS_JAR")
    if not jar_env:
        raise RuntimeError("Set TLA2TOOLS_JAR to the official v1.8.0 tla2tools.jar")
    jar = Path(jar_env).resolve()
    if not jar.is_file() or hashlib.sha256(jar.read_bytes()).hexdigest() != JAR_SHA256:
        raise RuntimeError("TLA2TOOLS_JAR is missing or does not match the pinned SHA-256")
    model = (ROOT / "SafeOutputTransport.tla").read_text()
    template = (ROOT / "SafeOutputTransport.cfg").read_text()
    cases = [("secure", "none", None)]
    cases += [(fault, fault, invariant) for fault, invariant in FAULTS.items()]
    cases += [(name, "none", name) for name in WITNESSES]
    with tempfile.TemporaryDirectory(prefix="gh-aw-transport-tlc-") as temporary:
        for transport in ["patch", "bundle"]:
            for name, fault, invariant in cases:
                directory = Path(temporary) / f"{transport}-{name}"
                directory.mkdir()
                (directory / "SafeOutputTransport.tla").write_text(model)
                config = template.replace('Transport = "patch"', f'Transport = "{transport}"')
                config = config.replace('Fault = "none"', f'Fault = "{fault}"')
                if name in WITNESSES:
                    config += f"\nINVARIANT {name}\n"
                (directory / "model.cfg").write_text(config)
                result = subprocess.run(
                    [os.environ.get("JAVA_BIN", "java"), "-XX:+UseParallelGC",
                     "-Xmx1g", "-cp", str(jar), "tlc2.TLC", "-workers", "1",
                     "-seed", "1", "-fp", "0", "-config", "model.cfg",
                     "-metadir", "states", "SafeOutputTransport.tla"],
                    cwd=directory, capture_output=True, text=True, timeout=120,
                )
                try:
                    counts = check_result(result.returncode, result.stdout + result.stderr, invariant)
                except RuntimeError as error:
                    raise RuntimeError(
                        f"{transport}/{name}: {error}\n{result.stdout}{result.stderr}"
                    ) from error
                print(f"{transport}/{name}: expected result "
                      f"({counts['distinct']} distinct states)", flush=True)


if __name__ == "__main__":
    main()
