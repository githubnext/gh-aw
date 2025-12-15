#!/usr/bin/env python3
"""
Render checkpoint timeline as GitHub Actions Job Summary

Takes a trace directory and generates a markdown table showing:
- Checkpoint ID
- Type
- Links to diff + tool I/O artifact paths
- "Replay from here" command snippet
"""

import json
import os
import sys
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List

def format_duration(duration_str: str) -> str:
    """Format duration string for display"""
    if not duration_str:
        return "-"
    if "ms" in duration_str:
        ms = int(duration_str.replace("ms", ""))
        if ms < 1000:
            return f"{ms}ms"
        return f"{ms/1000:.1f}s"
    return duration_str

def format_timestamp(ts_str: str) -> str:
    """Format timestamp for display"""
    try:
        dt = datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
        return dt.strftime("%H:%M:%S")
    except:
        return ts_str

def get_checkpoint_icon(kind: str) -> str:
    """Get emoji icon for checkpoint kind"""
    icons = {
        "tool_call": "🔧",
        "patch": "📝",
        "eval": "✅",
        "safe_output": "🚀",
        "decision": "🤔",
        "risk_gate": "🛡️",
    }
    return icons.get(kind, "📍")

def get_checkpoint_links(checkpoint: Dict[str, Any], trace_dir: Path) -> List[str]:
    """Generate links to checkpoint artifacts"""
    links = []
    
    # Link to diff
    if checkpoint.get("diff_path"):
        diff_path = checkpoint["diff_path"]
        if (trace_dir / diff_path).exists():
            links.append(f"[diff]({diff_path})")
    
    # Link to request
    if checkpoint.get("req_path"):
        req_path = checkpoint["req_path"]
        if (trace_dir / req_path).exists():
            links.append(f"[req]({req_path})")
    
    # Link to response
    if checkpoint.get("resp_path"):
        resp_path = checkpoint["resp_path"]
        if (trace_dir / resp_path).exists():
            links.append(f"[resp]({resp_path})")
    
    return links

def generate_replay_command(checkpoint_id: str, run_id: str, workflow: str) -> str:
    """Generate replay command snippet"""
    return f"""
<details>
<summary>Replay from here</summary>

```bash
gh workflow run "{workflow}" \\
  -f replay_run_id="{run_id}" \\
  -f start_checkpoint="{checkpoint_id}" \\
  -f tool_mode="cached"
```

Or via GitHub UI:
- Go to Actions → {workflow}
- Click "Run workflow"
- Set replay_run_id: `{run_id}`
- Set start_checkpoint: `{checkpoint_id}`

</details>
"""

def render_checkpoint_table(trace_dir: Path, manifest: Dict[str, Any]) -> str:
    """Render checkpoints as a markdown table"""
    checkpoints_file = trace_dir / "checkpoints.jsonl"
    
    if not checkpoints_file.exists():
        return "No checkpoints recorded."
    
    # Read all checkpoints
    checkpoints = []
    with open(checkpoints_file, 'r') as f:
        for line in f:
            if line.strip():
                checkpoints.append(json.loads(line))
    
    if not checkpoints:
        return "No checkpoints recorded."
    
    # Build table header
    table = [
        "## Checkpoint Timeline",
        "",
        f"**Run ID:** `{manifest['run_id']}` | **Workflow:** `{manifest['workflow']}` | **Engine:** `{manifest['engine']}`",
        "",
        "| Checkpoint | Type | What Changed | Time | Links | Replay |",
        "|------------|------|--------------|------|-------|--------|",
    ]
    
    run_id = manifest['run_id']
    workflow = manifest['workflow']
    
    # Add checkpoint rows
    for checkpoint in checkpoints:
        cp_id = checkpoint['id']
        kind = checkpoint['kind']
        name = checkpoint['name']
        ts = format_timestamp(checkpoint['ts'])
        icon = get_checkpoint_icon(kind)
        
        # Get metadata
        metadata = checkpoint.get('metadata', {})
        duration = format_duration(metadata.get('duration', ''))
        success = metadata.get('success', True)
        success_icon = "✓" if success else "✗"
        
        # Build "what changed" description
        what_changed = name
        if kind == "patch":
            added = metadata.get('added', 0)
            removed = metadata.get('removed', 0)
            what_changed = f"{name} (+{added}/-{removed})"
        elif kind == "eval":
            what_changed = f"{name} {success_icon}"
        
        # Get artifact links
        links = get_checkpoint_links(checkpoint, trace_dir)
        links_str = " · ".join(links) if links else "-"
        
        # Replay button
        replay_link = f"[▶️](#replay-{cp_id})"
        
        # Add row
        table.append(
            f"| {icon} `{cp_id}` | {kind} | {what_changed} | {ts} | {links_str} | {replay_link} |"
        )
    
    # Add replay commands section
    table.extend([
        "",
        "---",
        "",
        "## Replay Commands",
        "",
        "To replay execution from any checkpoint:",
        "",
    ])
    
    for checkpoint in checkpoints:
        cp_id = checkpoint['id']
        table.append(f"<a id=\"replay-{cp_id}\"></a>")
        table.append(generate_replay_command(cp_id, run_id, workflow))
    
    return "\n".join(table)

def render_summary_stats(trace_dir: Path) -> str:
    """Render summary statistics"""
    checkpoints_file = trace_dir / "checkpoints.jsonl"
    
    if not checkpoints_file.exists():
        return ""
    
    # Count checkpoints by type
    counts = {}
    with open(checkpoints_file, 'r') as f:
        for line in f:
            if line.strip():
                checkpoint = json.loads(line)
                kind = checkpoint['kind']
                counts[kind] = counts.get(kind, 0) + 1
    
    if not counts:
        return ""
    
    stats = ["## Summary", ""]
    for kind, count in sorted(counts.items()):
        icon = get_checkpoint_icon(kind)
        stats.append(f"- {icon} **{kind}**: {count}")
    
    stats.append("")
    return "\n".join(stats)

def render_trace_summary(trace_dir: Path) -> str:
    """Render complete trace summary"""
    manifest_file = trace_dir / "manifest.json"
    
    if not manifest_file.exists():
        return "⚠️ No trace manifest found. Trace capture may have failed."
    
    with open(manifest_file, 'r') as f:
        manifest = json.load(f)
    
    output = [
        "# 📊 Agent Execution Trace",
        "",
        f"**Trace Version:** {manifest['trace_version']}",
        f"**Created:** {manifest['created_at']}",
        f"**Repo SHA:** `{manifest['repo_sha']}`",
        "",
    ]
    
    output.append(render_summary_stats(trace_dir))
    output.append(render_checkpoint_table(trace_dir, manifest))
    
    return "\n".join(output)

def main():
    if len(sys.argv) < 2:
        print("Usage: render_trace_summary.py <trace_dir>", file=sys.stderr)
        sys.exit(1)
    
    trace_dir = Path(sys.argv[1])
    
    if not trace_dir.exists():
        print(f"Error: Trace directory not found: {trace_dir}", file=sys.stderr)
        sys.exit(1)
    
    summary = render_trace_summary(trace_dir)
    print(summary)

if __name__ == "__main__":
    main()
