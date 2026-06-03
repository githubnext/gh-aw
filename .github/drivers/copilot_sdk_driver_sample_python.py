#!/usr/bin/env python3
import os
import subprocess
import sys


def main() -> int:
    built_in_driver = f"{os.getenv('RUNNER_TEMP', '/tmp')}/gh-aw/actions/copilot_sdk_driver.cjs"
    completed = subprocess.run(["node", built_in_driver, *sys.argv[1:]], env=os.environ)
    return completed.returncode


if __name__ == "__main__":
    raise SystemExit(main())
