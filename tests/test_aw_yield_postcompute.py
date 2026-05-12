from __future__ import annotations

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPTS = ROOT / "scripts"
sys.path.insert(0, str(SCRIPTS))

import aw_yield_postcompute as post


def sample_precompute() -> dict:
    return {
        "portfolio_metrics": {
            "workflow_count": 2,
            "portfolio_yield": 0.1,
            "portfolio_overlap_drag": 0.4,
            "portfolio_cost": 0.3,
            "portfolio_risk": 0.2,
            "portfolio_maintenance_drag": 0.3,
            "average_agentic_fraction": 0.5,
            "average_deterministic_fraction": 0.5,
            "observability_declared_coverage": 0.5,
            "telemetry_observed_coverage": 0.5,
            "telemetry_validated_coverage": 0.5,
            "telemetry_coverage": 0.5,
            "evidence_quality": "low",
        },
        "workflows": [
            {
                "path": ".github/workflows/a.md",
                "name": "A",
                "description": "A",
                "has_lockfile": True,
                "lockfile_stale": False,
                "has_safe_outputs": True,
                "has_observability": False,
                "has_imported_observability": False,
                "observability_declared": False,
                "telemetry_observed": False,
                "telemetry_validated": False,
                "strict": True,
                "timeout_minutes": 10,
                "permissions_risk": 1.2,
                "tool_count": 2,
                "pre_agent_steps_count": 1,
                "post_steps_count": 1,
                "agentic_fraction": 1.4,
                "deterministic_fraction": -0.4,
                "usefulness": 0.8,
                "adoption": 0.6,
                "trust": 0.7,
                "cost": 1.2,
                "risk": -0.2,
                "maintenance_drag": 0.4,
                "overlap_drag": 0.5,
                "yield": 0.0,
                "intent_text": "review pull request",
                "recommendation_seed": "Instrument",
                "evidence_quality": "low",
                "notes": ["observability not declared"],
                "telemetry_metrics": {},
            },
            {
                "path": ".github/workflows/b.md",
                "name": "B",
                "description": "B",
                "has_lockfile": True,
                "lockfile_stale": False,
                "has_safe_outputs": True,
                "has_observability": True,
                "has_imported_observability": False,
                "observability_declared": True,
                "telemetry_observed": True,
                "telemetry_validated": True,
                "strict": True,
                "timeout_minutes": 10,
                "permissions_risk": 0.2,
                "tool_count": 2,
                "pre_agent_steps_count": 1,
                "post_steps_count": 1,
                "agentic_fraction": 0.4,
                "deterministic_fraction": 0.6,
                "usefulness": 0.7,
                "adoption": 0.5,
                "trust": 0.8,
                "cost": 0.3,
                "risk": 0.2,
                "maintenance_drag": 0.2,
                "overlap_drag": 0.5,
                "yield": 0.0,
                "intent_text": "review pull request security",
                "recommendation_seed": "Keep",
                "evidence_quality": "medium",
                "notes": [],
                "telemetry_metrics": {"success_rate": 0.9, "workflow_invocation_count": 4, "runtime_duration": 30},
            },
        ],
        "overlap_clusters": [{"workflows": [".github/workflows/a.md", ".github/workflows/b.md"], "max_overlap": 0.9, "reason": "review"}],
        "episode_metrics": [],
        "organizational_health_signals": {
            "fragmentation": 0.6,
            "reuse": 0.2,
            "trust_concentration": 0.2,
            "governance_drag": 0.7,
            "notes": [],
        },
        "recommendations_seed": {
            "keep": [".github/workflows/b.md"],
            "revise": [],
            "merge": [],
            "instrument": [".github/workflows/a.md"],
            "retire": [],
        },
        "overlap_pairs": [{"left": ".github/workflows/a.md", "right": ".github/workflows/b.md", "score": 0.9}],
    }


def test_scores_are_clamped() -> None:
    bounded = post.clamp_workflow_scores(sample_precompute()["workflows"][0])
    assert bounded["permissions_risk"] == 1.0
    assert bounded["cost"] == 1.0
    assert bounded["risk"] == 0.0
    assert bounded["agentic_fraction"] == 1.0
    assert bounded["deterministic_fraction"] == 0.0


def test_recommendations_are_valid_and_mutually_exclusive(tmp_path: Path) -> None:
    agent_dir = tmp_path / "agent"
    agent_dir.mkdir()
    (agent_dir / "portfolio-yield-agent.json").write_text(
        json.dumps(
            {
                "recommendations": {
                    "keep": [{"path": ".github/workflows/b.md"}],
                    "instrument": [{"path": ".github/workflows/a.md"}],
                }
            }
        ),
        encoding="utf-8",
    )
    final_payload, _summary, _notes = post.finalize(sample_precompute(), agent_dir)
    assert final_payload["keep"] == [".github/workflows/b.md"]
    assert final_payload["instrument"] == [".github/workflows/a.md"]
    assert set(final_payload["keep"]).isdisjoint(final_payload["instrument"])
    assert final_payload["observability_declared_coverage"] == 0.5
    assert final_payload["telemetry_observed_coverage"] == 0.5
    assert final_payload["telemetry_validated_coverage"] == 0.5


def test_postcompute_handles_malformed_agent_output_safely(tmp_path: Path) -> None:
    bad_precompute = tmp_path / "precompute.json"
    bad_precompute.write_text("{}", encoding="utf-8")
    out = tmp_path / "out.json"
    exit_code = post.main(["--precompute", str(bad_precompute), "--agent-output", str(tmp_path / "agent"), "--out", str(out)])
    payload = json.loads(out.read_text(encoding="utf-8"))
    assert exit_code == 1
    assert "error" in payload
    assert payload["evidence_quality"] == "low"


def test_postcompute_does_not_allow_invented_telemetry_to_increase_confidence(tmp_path: Path) -> None:
    agent_dir = tmp_path / "agent"
    agent_dir.mkdir()
    (agent_dir / "portfolio-yield-agent.json").write_text(
        json.dumps(
            {
                "recommendations": {
                    "keep": [{"path": ".github/workflows/b.md"}],
                    "instrument": [{"path": ".github/workflows/a.md"}],
                },
                "telemetry_claims": [{"path": ".github/workflows/a.md", "metric": "success_rate"}],
                "evidence_quality": "high",
            }
        ),
        encoding="utf-8",
    )
    final_payload, _summary, notes = post.finalize(sample_precompute(), agent_dir)
    assert final_payload["evidence_quality"] == "low"
    assert any("invented telemetry" in note.lower() for note in notes)


def test_recompute_overlap_drag_clamps_invalid_scores() -> None:
    assert post.recompute_overlap_drag({"overlap_pairs": [{"score": "bad"}]}) == 0.0
    assert post.recompute_overlap_drag({"overlap_pairs": [{"score": float("nan")}]}) == 0.0
    assert post.recompute_overlap_drag({"overlap_pairs": [{"score": float("inf")}]}) == 0.0
    assert post.recompute_overlap_drag({"overlap_pairs": [{"score": -1}]}) == 0.0
    assert post.recompute_overlap_drag({"overlap_pairs": [{"score": 0.5}]}) == 0.5
