#!/usr/bin/env python3
"""Deterministic postcompute for Agentic Workflow Portfolio Yield."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import sys
from pathlib import Path
from typing import Any

import aw_yield_precompute as pre

ALLOWED_BUCKETS = {name.lower(): name for name in pre.ALLOWED_RECOMMENDATIONS}
AGENT_SUMMARY_CANDIDATES = ("portfolio-yield-agent.json", "aw-portfolio-yield-agent.json")
MAX_REPORT_LENGTH = 45000


class FinalizeError(ValueError):
    """Raised when postcompute inputs are malformed."""


def load_json(path: Path) -> Any:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise FinalizeError(f"Missing JSON file: {path}") from exc
    except json.JSONDecodeError as exc:
        raise FinalizeError(f"Malformed JSON in {path}: {exc}") from exc


def clamp_workflow_scores(workflow: dict[str, Any]) -> dict[str, Any]:
    bounded = dict(workflow)
    for key in (
        "permissions_risk",
        "agentic_fraction",
        "deterministic_fraction",
        "usefulness",
        "adoption",
        "trust",
        "cost",
        "risk",
        "maintenance_drag",
        "overlap_drag",
        "yield",
    ):
        bounded[key] = round(pre.clamp(workflow.get(key, 0.0)), 4)
    bounded["deterministic_fraction"] = round(pre.clamp(1.0 - bounded["agentic_fraction"]), 4)
    bounded["notes"] = list(dict.fromkeys(workflow.get("notes", [])))
    return bounded


def recommendation_buckets(seed: dict[str, Any], workflows: dict[str, dict[str, Any]]) -> dict[str, list[str]]:
    buckets = {bucket: [] for bucket in ALLOWED_BUCKETS}
    for bucket, entries in (seed or {}).items():
        lower = bucket.lower()
        if lower not in buckets:
            continue
        for entry in entries or []:
            path = entry["path"] if isinstance(entry, dict) else str(entry)
            if path in workflows and path not in buckets[lower]:
                buckets[lower].append(path)
    return buckets


def read_agent_summary(agent_dir: Path) -> dict[str, Any]:
    for candidate in AGENT_SUMMARY_CANDIDATES:
        path = agent_dir / candidate
        if path.exists():
            payload = load_json(path)
            if not isinstance(payload, dict):
                raise FinalizeError(f"Agent summary must be an object: {path}")
            return payload
    return {}


def normalize_agent_buckets(agent_summary: dict[str, Any], workflows: dict[str, dict[str, Any]]) -> tuple[dict[str, list[str]], list[str]]:
    notes: list[str] = []
    raw_buckets = agent_summary.get("recommendations") if isinstance(agent_summary.get("recommendations"), dict) else agent_summary
    buckets = {bucket: [] for bucket in ALLOWED_BUCKETS}
    seen: dict[str, str] = {}
    for bucket in ALLOWED_BUCKETS:
        entries = raw_buckets.get(bucket, raw_buckets.get(ALLOWED_BUCKETS[bucket], [])) if isinstance(raw_buckets, dict) else []
        if entries is None:
            entries = []
        if not isinstance(entries, list):
            raise FinalizeError(f"Recommendation bucket '{bucket}' must be a list")
        for entry in entries:
            path = entry.get("path") if isinstance(entry, dict) else str(entry)
            path = pre.normalize_text(path)
            if not path:
                continue
            if path not in workflows:
                raise FinalizeError(f"Unknown workflow in recommendations: {path}")
            other = seen.get(path)
            if other and other != bucket:
                raise FinalizeError(f"Workflow '{path}' appears in multiple recommendation buckets")
            seen[path] = bucket
            if path not in buckets[bucket]:
                buckets[bucket].append(path)
    telemetry_claims = agent_summary.get("telemetry_claims", [])
    if isinstance(telemetry_claims, list):
        for claim in telemetry_claims:
            if not isinstance(claim, dict):
                notes.append("Ignored malformed telemetry claim from agent output.")
                continue
            path = pre.normalize_text(claim.get("path") or claim.get("workflow"))
            metric = pre.normalize_text(claim.get("metric"))
            if not path or path not in workflows or metric not in workflows[path].get("telemetry_metrics", {}):
                notes.append(f"Ignored invented telemetry claim: {path or 'unknown'}::{metric or 'unknown'}")
    return buckets, notes


def fill_missing_recommendations(current: dict[str, list[str]], seeds: dict[str, list[str]], workflows: dict[str, dict[str, Any]]) -> dict[str, list[str]]:
    assigned = {path for entries in current.values() for path in entries}
    for bucket, entries in seeds.items():
        for path in entries:
            if path in workflows and path not in assigned:
                current.setdefault(bucket, []).append(path)
                assigned.add(path)
    for path in workflows:
        if path not in assigned:
            current.setdefault("revise", []).append(path)
    return {bucket: sorted(dict.fromkeys(entries)) for bucket, entries in current.items()}


def recompute_overlap_drag(payload: dict[str, Any]) -> float:
    pairs = payload.get("overlap_pairs", [])
    if not isinstance(pairs, list):
        return 0.0
    drag = 0.0
    for pair in pairs:
        if not isinstance(pair, dict):
            continue
        score = pre.clamp(pair.get("score", 0.0))
        drag += score**2 * 2.0
    return round(drag, 4)


def derive_evidence_quality(workflows: list[dict[str, Any]], base_quality: str) -> str:
    observed_coverage = sum(1.0 for workflow in workflows if workflow.get("telemetry_observed")) / max(1, len(workflows))
    validated_coverage = sum(1.0 for workflow in workflows if workflow.get("telemetry_validated")) / max(1, len(workflows))
    derived = pre.portfolio_evidence_quality(workflows, observed_coverage, validated_coverage)
    order = {"low": 0, "medium": 1, "high": 2}
    return derived if order[derived] <= order.get(base_quality, 0) else base_quality


def top_actions(final_payload: dict[str, Any]) -> list[str]:
    actions: list[str] = []
    instrument = final_payload.get("instrument", [])
    merge = final_payload.get("merge", [])
    retire = final_payload.get("retire", [])
    revise = final_payload.get("revise", [])
    if instrument:
        actions.append(f"Instrument {instrument[0]} with stable OTel evidence and safe-output validation.")
    if merge:
        actions.append(f"Consolidate overlap around {merge[0]} to reduce portfolio drag.")
    if revise:
        actions.append(f"Revise {revise[0]} to shift deterministic work out of the agent path.")
    if retire and len(actions) < 3:
        actions.append(f"Retire or quarantine {retire[0]} if trust does not improve.")
    return actions[:3]


def build_report_markdown(final_payload: dict[str, Any], precompute_payload: dict[str, Any], agent_summary: dict[str, Any], post_notes: list[str]) -> str:
    metrics = {
        "Portfolio yield": final_payload["portfolio_yield"],
        "Workflow count": final_payload["workflow_count"],
        "Agentic fraction": final_payload["average_agentic_fraction"],
        "Deterministic fraction": round(1.0 - final_payload["average_agentic_fraction"], 4),
        "Observability declared coverage": precompute_payload.get("portfolio_metrics", {}).get("observability_declared_coverage", 0.0),
        "Telemetry observed coverage": precompute_payload.get("portfolio_metrics", {}).get("telemetry_observed_coverage", 0.0),
        "Telemetry validated coverage": precompute_payload.get("portfolio_metrics", {}).get("telemetry_validated_coverage", 0.0),
        "High-overlap clusters": len(final_payload.get("overlap_clusters", [])),
        "Estimated governance drag": final_payload.get("organizational_health_signals", {}).get("governance_drag", 0.0),
        "Estimated trust score": round(
            sum(workflow.get("trust", 0.0) for workflow in precompute_payload.get("workflows", []))
            / max(1, len(precompute_payload.get("workflows", []))),
            4,
        ),
    }
    workflow_rows = []
    for workflow in sorted(precompute_payload.get("workflows", []), key=lambda item: item.get("yield", 0.0), reverse=True):
        recommendation = next(
            (bucket.title() for bucket in ALLOWED_BUCKETS if workflow["path"] in final_payload.get(bucket, [])),
            workflow.get("recommendation_seed", "Revise"),
        )
        note_text = "; ".join(workflow.get("notes", [])[:2]) or "-"
        workflow_rows.append(
            f"| `{workflow['path']}` | {recommendation} | {workflow['yield']:.4f} | {workflow['trust']:.4f} | {workflow['cost']:.4f} | {workflow['risk']:.4f} | {workflow['overlap_drag']:.4f} | {workflow['adoption']:.4f} | {workflow['agentic_fraction']:.4f} | {note_text} |"
        )
    overlap_lines = [
        f"- {', '.join(cluster['workflows'])} (max overlap {cluster['max_overlap']:.4f}; {cluster['reason']})"
        for cluster in final_payload.get("overlap_clusters", [])
    ] or ["- No high-overlap clusters detected."]
    episode_lines = [
        f"- {episode['episode']}: workflows={', '.join(episode['workflows'])}; coordination drag={episode['coordination_drag']:.4f}; episode yield={episode['episode_yield']:.4f}"
        for episode in precompute_payload.get("episode_metrics", [])
    ]
    org = final_payload.get("organizational_health_signals", {})
    deterministic_findings = agent_summary.get("deterministic_vs_agentic_findings", [])
    if not deterministic_findings:
        deterministic_findings = [
            f"{workflow['path']} has agentic fraction {workflow['agentic_fraction']:.4f} despite limited deterministic scaffolding."
            for workflow in sorted(precompute_payload.get("workflows", []), key=lambda item: item.get("agentic_fraction", 0.0), reverse=True)[:3]
            if workflow.get("agentic_fraction", 0.0) > 0.6
        ]
    highest_value_actions = agent_summary.get("highest_value_actions") or top_actions(final_payload)
    retirement_candidates = agent_summary.get("retirement_candidates") or final_payload.get("retire", [])
    consolidation = agent_summary.get("consolidation_opportunities") or final_payload.get("merge", [])
    instrumentation_gaps = agent_summary.get("instrumentation_gaps") or final_payload.get("instrument", [])
    executive_summary = pre.normalize_text(agent_summary.get("executive_summary"))
    if not executive_summary:
        if final_payload["evidence_quality"] == "low":
            executive_summary = "The workflow ecosystem is under-instrumented, so the portfolio signal is directionally useful but not yet strong enough for confident optimization."
        elif org.get("fragmentation", 0.0) > 0.6:
            executive_summary = "The workflow ecosystem is fragmenting: overlap drag and governance drag are eroding portfolio yield."
        elif final_payload["portfolio_yield"] > 0.12:
            executive_summary = "The workflow ecosystem is producing positive value overall, with enough trust and reuse to justify continued investment."
        else:
            executive_summary = "The workflow ecosystem is mixed: some workflows are valuable, but overlap, cost, or trust gaps are holding the portfolio back."
    compact_json = json.dumps(
        {
            "portfolio_yield": final_payload["portfolio_yield"],
            "workflow_count": final_payload["workflow_count"],
            "observability_declared_coverage": final_payload.get("observability_declared_coverage", 0.0),
            "telemetry_observed_coverage": final_payload.get("telemetry_observed_coverage", 0.0),
            "telemetry_validated_coverage": final_payload.get("telemetry_validated_coverage", 0.0),
            "keep": final_payload.get("keep", []),
            "revise": final_payload.get("revise", []),
            "merge": final_payload.get("merge", []),
            "instrument": final_payload.get("instrument", []),
            "retire": final_payload.get("retire", []),
            "evidence_quality": final_payload["evidence_quality"],
        },
        separators=(",", ":"),
        sort_keys=True,
    )
    lines = [
        "# Agentic Workflow Portfolio Yield Report",
        "",
        "## Executive Summary",
        "",
        executive_summary,
        "",
        "## Portfolio Health",
        "",
        "| Metric | Value |",
        "|---|---:|",
    ]
    lines.extend(f"| {metric} | {value} |" for metric, value in metrics.items())
    lines.extend(
        [
            "",
            "## Workflow Portfolio",
            "",
            "| Workflow | Recommendation | Yield | Trust | Cost | Risk | Overlap | Adoption | Agentic Fraction | Notes |",
            "|---|---|---:|---:|---:|---:|---:|---:|---:|---|",
            *workflow_rows,
            "",
            "## Overlap Clusters",
            "",
            *overlap_lines,
        ]
    )
    if episode_lines:
        lines.extend(["", "## Episode-Level Observations", "", *episode_lines])
    lines.extend(
        [
            "",
            "## Organizational Health Signals",
            "",
            f"- fragmentation: {org.get('fragmentation', 0.0):.4f}",
            f"- reuse: {org.get('reuse', 0.0):.4f}",
            f"- trust concentration: {org.get('trust_concentration', 0.0):.4f}",
            f"- governance drag: {org.get('governance_drag', 0.0):.4f}",
            *[f"- {note}" for note in org.get("notes", [])],
            *[f"- {note}" for note in post_notes],
            "",
            "## Deterministic vs Agentic Findings",
            "",
            *([f"- {item}" for item in deterministic_findings] or ["- No outsized agentic misuse detected from current evidence."]),
            "",
            "## Highest-Value Actions",
            "",
            *([f"1. {item}" if index == 0 else f"{index + 1}. {item}" for index, item in enumerate(highest_value_actions[:3])] or ["1. Improve observability coverage."]),
            "",
            "## Retirement Candidates",
            "",
            *([f"- {item}" for item in retirement_candidates] or ["- No immediate retirement candidates."]),
            "",
            "## Consolidation Opportunities",
            "",
            *([f"- {item}" for item in consolidation] or ["- No consolidation opportunities identified."]),
            "",
            "## Instrumentation Gaps",
            "",
            *([f"- {item}" for item in instrumentation_gaps] or ["- No critical instrumentation gaps detected."]),
            "",
            "## Deterministic Portfolio JSON",
            "",
            "```json",
            compact_json,
            "```",
        ]
    )
    report = "\n".join(lines).strip() + "\n"
    if len(report) > MAX_REPORT_LENGTH:
        report = report[: MAX_REPORT_LENGTH - 16].rstrip() + "\n\n[truncated]\n"
    return report


def finalize(precompute_payload: dict[str, Any], agent_dir: Path) -> tuple[dict[str, Any], dict[str, Any], list[str]]:
    workflows_raw = precompute_payload.get("workflows")
    if not isinstance(workflows_raw, list):
        raise FinalizeError("Precompute JSON must contain a workflows array")
    workflows = [clamp_workflow_scores(workflow) for workflow in workflows_raw if isinstance(workflow, dict)]
    workflow_index: dict[str, dict[str, Any]] = {}
    for workflow in workflows:
        path = workflow.get("path")
        if not path or path in workflow_index:
            raise FinalizeError(f"Duplicate or missing workflow path: {path}")
        workflow_index[path] = workflow
    for workflow in workflows:
        workflow["yield"] = pre.compute_workflow_yield(
            workflow["usefulness"],
            workflow["adoption"],
            workflow["trust"],
            workflow["cost"],
            workflow["risk"],
            workflow["maintenance_drag"],
            workflow["overlap_drag"],
        )
    seeds = recommendation_buckets(precompute_payload.get("recommendations_seed", {}), workflow_index)
    agent_summary = read_agent_summary(agent_dir)
    post_notes: list[str] = []
    try:
        agent_buckets, notes = normalize_agent_buckets(agent_summary, workflow_index)
        post_notes.extend(notes)
    except FinalizeError:
        raise
    buckets = fill_missing_recommendations(agent_buckets, seeds, workflow_index)
    overlap_drag_value = recompute_overlap_drag(precompute_payload)
    portfolio_yield = round(sum(workflow["yield"] for workflow in workflows) / max(1, len(workflows)) - pre.LAMBDA * overlap_drag_value, 4)
    final_payload = {
        "portfolio_yield": portfolio_yield,
        "workflow_count": len(workflows),
        "portfolio_cost": round(sum(workflow["cost"] for workflow in workflows) / max(1, len(workflows)), 4),
        "portfolio_risk": round(sum(workflow["risk"] for workflow in workflows) / max(1, len(workflows)), 4),
        "portfolio_maintenance_drag": round(sum(workflow["maintenance_drag"] for workflow in workflows) / max(1, len(workflows)), 4),
        "portfolio_overlap_drag": overlap_drag_value,
        "average_agentic_fraction": round(sum(workflow["agentic_fraction"] for workflow in workflows) / max(1, len(workflows)), 4),
        "observability_declared_coverage": round(
            sum(1.0 for workflow in workflows if workflow.get("observability_declared")) / max(1, len(workflows)),
            4,
        ),
        "telemetry_observed_coverage": round(
            sum(1.0 for workflow in workflows if workflow.get("telemetry_observed")) / max(1, len(workflows)),
            4,
        ),
        "telemetry_validated_coverage": round(
            sum(1.0 for workflow in workflows if workflow.get("telemetry_validated")) / max(1, len(workflows)),
            4,
        ),
        "evidence_quality": derive_evidence_quality(workflows, precompute_payload.get("portfolio_metrics", {}).get("evidence_quality", "low")),
        "keep": buckets.get("keep", []),
        "revise": buckets.get("revise", []),
        "merge": buckets.get("merge", []),
        "instrument": buckets.get("instrument", []),
        "retire": buckets.get("retire", []),
        "overlap_clusters": precompute_payload.get("overlap_clusters", []),
        "organizational_health_signals": precompute_payload.get("organizational_health_signals", {}),
    }
    final_payload["report_markdown"] = build_report_markdown(final_payload, {**precompute_payload, "workflows": workflows}, agent_summary, post_notes)
    return final_payload, agent_summary, post_notes


def write_safe_output(agent_dir: Path, report_markdown: str) -> None:
    title = f"Agentic Workflow Portfolio Yield Report — {dt.date.today().isoformat()}"
    payload = {
        "items": [
            {
                "type": "create_issue",
                "title": title,
                "body": report_markdown,
            }
        ],
        "errors": [],
    }
    (agent_dir / "agent_output.json").write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


def error_payload(message: str) -> dict[str, Any]:
    return {
        "error": message,
        "portfolio_yield": 0.0,
        "workflow_count": 0,
        "portfolio_cost": 0.0,
        "portfolio_risk": 0.0,
        "portfolio_maintenance_drag": 0.0,
        "portfolio_overlap_drag": 0.0,
        "average_agentic_fraction": 0.0,
        "observability_declared_coverage": 0.0,
        "telemetry_observed_coverage": 0.0,
        "telemetry_validated_coverage": 0.0,
        "evidence_quality": "low",
        "keep": [],
        "revise": [],
        "merge": [],
        "instrument": [],
        "retire": [],
        "overlap_clusters": [],
        "organizational_health_signals": {"fragmentation": 0.0, "reuse": 0.0, "trust_concentration": 0.0, "governance_drag": 0.0, "notes": [message]},
        "report_markdown": "# Agentic Workflow Portfolio Yield Report\n\n## Executive Summary\n\nPostcompute failed safely.\n",
    }


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--precompute", required=True, help="Path to the precompute JSON")
    parser.add_argument("--agent-output", required=True, help="Directory containing agent outputs")
    parser.add_argument("--out", required=True, help="Output JSON path")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    agent_dir = Path(args.agent_output)
    agent_dir.mkdir(parents=True, exist_ok=True)
    try:
        precompute_payload = load_json(Path(args.precompute))
        if not isinstance(precompute_payload, dict):
            raise FinalizeError("Precompute JSON must be an object")
        final_payload, _agent_summary, _notes = finalize(precompute_payload, agent_dir)
        write_safe_output(agent_dir, final_payload["report_markdown"])
        out_path.write_text(json.dumps(final_payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        return 0
    except FinalizeError as exc:
        payload = error_payload(str(exc))
        out_path.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        return 1


if __name__ == "__main__":
    sys.exit(main())
