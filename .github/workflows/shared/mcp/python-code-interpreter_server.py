#!/usr/bin/env python3
"""
FastMCP v2.0 Python Data Agent (with file support)
--------------------------------------------------

Features:
- Each request runs inside /app/runs/<uuid>
- Optional `files` parameter: list of paths to copy into the run folder
- Executes Python code with real-time streaming via MCP Stream
- Returns run metadata and list of generated files
"""

from fastmcp import MCP, Stream
import subprocess, os, uuid, shutil

BASE_DIR = "/app/runs"
os.makedirs(BASE_DIR, exist_ok=True)

mcp = MCP(name="python-code-interpreter", version="2.0.0")


@mcp.tool(
    "run_python_query",
    description=(
        "Execute Python data-analysis code in an isolated folder.\n"
        "Optionally provide 'files' (list of paths) to copy into the run directory."
    ),
)
def run_python_query(code: str, stream: Stream, files: list[str] | None = None):
    # Create isolated run folder
    run_id = str(uuid.uuid4())[:8]
    run_path = os.path.join(BASE_DIR, run_id)
    os.makedirs(run_path, exist_ok=True)

    # Copy any provided input files
    copied = []
    if files:
        for src in files:
            if not os.path.exists(src):
                stream.send(f"⚠️  Skipping missing file: {src}")
                continue
            dst = os.path.join(run_path, os.path.basename(src))
            shutil.copy(src, dst)
            copied.append(os.path.basename(src))
        stream.send(f"📂 Copied input files: {copied}")

    # Write Python code
    script_path = os.path.join(run_path, "script.py")
    with open(script_path, "w") as f:
        f.write(code)

    stream.send(f"🧪 Starting run {run_id} in {run_path} ...")

    # Execute the script
    proc = subprocess.Popen(
        ["python3", "script.py"],
        cwd=run_path,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        bufsize=1,
    )

    for line in proc.stdout:
        stream.send(line.rstrip())

    proc.wait()
    exit_code = proc.returncode

    # Collect generated files
    files_out = sorted(os.listdir(run_path))
    stream.send(f"✅ Done. Files generated: {files_out}")

    return {
        "run_id": run_id,
        "run_path": run_path,
        "input_files": copied,
        "exit_code": exit_code,
        "files": files_out,
    }


if __name__ == "__main__":
    mcp.run()
