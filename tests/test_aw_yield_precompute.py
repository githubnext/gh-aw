from __future__ import annotations

import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPTS = ROOT / "scripts"
sys.path.insert(0, str(SCRIPTS))

import aw_yield_precompute as pre


def write_workflow(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def test_workflow_discovery_excludes_shared(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    write_workflow(workflows / "alpha.md", "---\non: workflow_dispatch\n---\n# Alpha\n")
    write_workflow(workflows / "shared" / "helper.md", "---\non: workflow_dispatch\n---\n# Helper\n")
    discovered = pre.discover_workflow_files(workflows)
    assert [path.name for path in discovered] == ["alpha.md"]


def test_frontmatter_parsing_works() -> None:
    frontmatter = """name: Portfolio Yield\ndescription: Example\nstrict: true\ntimeout-minutes: 15\nimports:\n  - uses: shared/otel-observability.md\n    with:\n      mode: summary\ntools:\n  github:\n    mode: gh-proxy\n  bash: true\nsafe-outputs:\n  create-issue:\n    max: 1\n"""
    parsed = pre.parse_frontmatter_text(frontmatter)
    assert parsed["name"] == "Portfolio Yield"
    assert parsed["strict"] is True
    assert parsed["timeout-minutes"] == 15
    assert parsed["imports"][0]["uses"] == "shared/otel-observability.md"
    assert parsed["tools"]["github"]["mode"] == "gh-proxy"
    assert parsed["safe-outputs"]["create-issue"]["max"] == 1


def test_imports_are_detected(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    workflow = workflows / "alpha.md"
    write_workflow(workflow, "---\nimports:\n  - shared/otel-observability.md\n---\n# Alpha\n")
    imports = pre.normalize_import_paths(workflow, pre.read_workflow(workflow)[0])
    assert imports == [workflows / "shared" / "otel-observability.md"]


def test_imported_observability_is_detected(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    shared = workflows / "shared" / "otel-observability.md"
    write_workflow(
        shared,
        "---\nobservability:\n  otlp:\n    endpoint:\n      url: ${{ secrets.OTLP_ENDPOINT }}\n---\n",
    )
    workflow = workflows / "alpha.md"
    write_workflow(workflow, "---\nimports:\n  - shared/otel-observability.md\n---\n# Alpha\n")
    frontmatter, _ = pre.read_workflow(workflow)
    assert pre.has_imported_observability(workflow, frontmatter) is True


def test_telemetry_entry_preserves_observed_and_validated_flags() -> None:
    normalized = pre.normalize_telemetry_entry(
        {
            "workflow_path": ".github/workflows/alpha.md",
            "workflow_invocation_count": 4,
            "success_rate": 0.75,
            "runtime_duration": 12,
            "observed": True,
            "validated": True,
            "source": "github-actions-runs",
        }
    )
    assert normalized["metrics"]["workflow_invocation_count"] == 4
    assert normalized["observed"] is True
    assert normalized["validated"] is True
    assert normalized["source"] == "github-actions-runs"


def test_declared_observability_without_telemetry_stays_low_evidence(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    workflow = workflows / "alpha.md"
    write_workflow(
        workflow,
        "---\nobservability:\n  otlp:\n    endpoint:\n      url: ${{ secrets.OTLP_ENDPOINT }}\nstrict: true\nsafe-outputs:\n  create-issue:\n    max: 1\n---\n# Alpha\n",
    )
    record = pre.build_workflow_record(workflow, workflows, {})
    assert record["observability_declared"] is True
    assert record["telemetry_observed"] is False
    assert record["telemetry_validated"] is False
    assert record["evidence_quality"] == "low"
    assert "telemetry not observed" in record["notes"]


def test_portfolio_yield_workflow_imports_otel_observability() -> None:
    workflow = ROOT / ".github" / "workflows" / "aw-portfolio-yield.md"
    frontmatter, _ = pre.read_workflow(workflow)
    imports = pre.normalize_import_paths(workflow, frontmatter)
    assert workflow.parent / "shared" / "observability-otlp.md" in imports
    assert pre.has_imported_observability(workflow, frontmatter) is True


def test_telemetry_prefers_repo_relative_path_over_name_collisions(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    first = workflows / "foo" / "alpha.md"
    second = workflows / "bar" / "alpha.md"
    write_workflow(first, "---\nname: Alpha One\n---\n# Alpha One\n")
    write_workflow(second, "---\nname: Alpha Two\n---\n# Alpha Two\n")
    telemetry = tmp_path / "summary.json"
    telemetry.write_text(
        """
{
  "workflows": [
    {"workflow_path": ".github/workflows/foo/alpha.md", "workflow_invocation_count": 1, "observed": true, "validated": true},
    {"workflow_path": ".github/workflows/bar/alpha.md", "workflow_invocation_count": 7, "observed": true, "validated": true}
  ]
}
""".strip(),
        encoding="utf-8",
    )
    telemetry_index = pre.load_otel_summary(str(telemetry))
    record = pre.build_workflow_record(second, workflows, telemetry_index)
    assert record["telemetry_metrics"]["workflow_invocation_count"] == 7


def test_portfolio_metrics_split_declared_observed_and_validated_coverage() -> None:
    workflows = [
        {"yield": 0.3, "cost": 0.2, "risk": 0.2, "maintenance_drag": 0.2, "agentic_fraction": 0.4, "deterministic_fraction": 0.6, "observability_declared": True, "telemetry_observed": True, "telemetry_validated": True, "evidence_quality": "high"},
        {"yield": 0.2, "cost": 0.2, "risk": 0.2, "maintenance_drag": 0.2, "agentic_fraction": 0.5, "deterministic_fraction": 0.5, "observability_declared": True, "telemetry_observed": False, "telemetry_validated": False, "evidence_quality": "low"},
        {"yield": 0.1, "cost": 0.2, "risk": 0.2, "maintenance_drag": 0.2, "agentic_fraction": 0.6, "deterministic_fraction": 0.4, "observability_declared": False, "telemetry_observed": False, "telemetry_validated": False, "evidence_quality": "low"},
    ]
    metrics = pre.compute_portfolio_metrics(workflows, overlap_drag_value=0.0)
    assert metrics["observability_declared_coverage"] == 0.6667
    assert metrics["telemetry_observed_coverage"] == 0.3333
    assert metrics["telemetry_validated_coverage"] == 0.3333
    assert metrics["telemetry_coverage"] == 0.3333


def test_relative_import_escapes_are_rejected(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    escaped = tmp_path / "outside.md"
    write_workflow(
        escaped,
        "---\nobservability:\n  otlp:\n    endpoint:\n      url: https://example.invalid\n---\n",
    )
    workflow = workflows / "alpha.md"
    write_workflow(workflow, "---\nimports:\n  - ../outside.md\n---\n# Alpha\n")
    frontmatter, _ = pre.read_workflow(workflow)
    assert pre.normalize_import_paths(workflow, frontmatter) == []
    assert pre.has_imported_observability(workflow, frontmatter) is False


def test_absolute_imports_are_rejected(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    escaped = tmp_path / "outside.md"
    write_workflow(
        escaped,
        "---\nobservability:\n  otlp:\n    endpoint:\n      url: https://example.invalid\n---\n",
    )
    workflow = workflows / "alpha.md"
    write_workflow(workflow, f"---\nimports:\n  - {escaped}\n---\n# Alpha\n")
    frontmatter, _ = pre.read_workflow(workflow)
    assert pre.normalize_import_paths(workflow, frontmatter) == []
    assert pre.has_imported_observability(workflow, frontmatter) is False


def test_windows_absolute_imports_are_rejected(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    workflow = workflows / "alpha.md"
    write_workflow(workflow, "---\nimports:\n  - \\\\server\\share\\outside.md\n---\n# Alpha\n")
    frontmatter, _ = pre.read_workflow(workflow)
    assert pre.normalize_import_paths(workflow, frontmatter) == []


def test_shared_import_escapes_are_rejected(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    escaped = workflows / "outside.md"
    write_workflow(
        escaped,
        "---\nobservability:\n  otlp:\n    endpoint:\n      url: https://example.invalid\n---\n",
    )
    workflow = workflows / "alpha.md"
    write_workflow(workflow, "---\nimports:\n  - shared/../outside.md\n---\n# Alpha\n")
    frontmatter, _ = pre.read_workflow(workflow)
    assert pre.normalize_import_paths(workflow, frontmatter) == []
    assert pre.has_imported_observability(workflow, frontmatter) is False


def test_missing_safe_outputs_increases_risk(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    base = "---\non:\n  workflow_dispatch:\npermissions:\n  contents: read\nstrict: true\ntimeout-minutes: 10\n---\n# Alpha\n"
    with_safe = workflows / "with-safe.md"
    without_safe = workflows / "without-safe.md"
    write_workflow(with_safe, base.replace("---\n# Alpha", "safe-outputs:\n  create-issue:\n    max: 1\n---\n# Alpha"))
    write_workflow(without_safe, base)
    risk_with = pre.build_workflow_record(with_safe, workflows, {})["risk"]
    risk_without = pre.build_workflow_record(without_safe, workflows, {})["risk"]
    assert risk_without > risk_with


def test_missing_lockfile_is_detected(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    workflow = workflows / "alpha.md"
    write_workflow(workflow, "---\non: workflow_dispatch\nstrict: true\n---\n# Alpha\n")
    record = pre.build_workflow_record(workflow, workflows, {})
    assert record["has_lockfile"] is False


def test_stale_lockfile_is_detected_where_mtimes_allow(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    workflow = workflows / "alpha.md"
    lockfile = workflows / "alpha.lock.yml"
    write_workflow(workflow, "---\non: workflow_dispatch\nstrict: true\n---\n# Alpha\n")
    lockfile.write_text("name: alpha\n", encoding="utf-8")
    os.utime(lockfile, (1, 1))
    os.utime(workflow, (10, 10))
    record = pre.build_workflow_record(workflow, workflows, {})
    assert record["has_lockfile"] is True
    assert record["lockfile_stale"] is True


def test_missing_strict_mode_increases_risk(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    strict_path = workflows / "strict.md"
    loose_path = workflows / "loose.md"
    write_workflow(strict_path, "---\non: workflow_dispatch\nstrict: true\ntimeout-minutes: 10\nsafe-outputs:\n  create-issue:\n    max: 1\n---\n# Strict\n")
    write_workflow(loose_path, "---\non: workflow_dispatch\nstrict: false\ntimeout-minutes: 10\nsafe-outputs:\n  create-issue:\n    max: 1\n---\n# Loose\n")
    assert pre.build_workflow_record(loose_path, workflows, {})["risk"] > pre.build_workflow_record(strict_path, workflows, {})["risk"]


def test_missing_timeout_increases_risk(tmp_path: Path) -> None:
    workflows = tmp_path / ".github" / "workflows"
    timed = workflows / "timed.md"
    untimed = workflows / "untimed.md"
    write_workflow(timed, "---\non: workflow_dispatch\nstrict: true\ntimeout-minutes: 10\nsafe-outputs:\n  create-issue:\n    max: 1\n---\n# Timed\n")
    write_workflow(untimed, "---\non: workflow_dispatch\nstrict: true\nsafe-outputs:\n  create-issue:\n    max: 1\n---\n# Untimed\n")
    assert pre.build_workflow_record(untimed, workflows, {})["risk"] > pre.build_workflow_record(timed, workflows, {})["risk"]


def test_id_token_permission_increases_risk() -> None:
    base = pre.permissions_risk({"contents": "write"})
    with_id_token = pre.permissions_risk({"contents": "write", "id-token": "write"})
    assert with_id_token > base


def test_overlap_detection_finds_similar_workflows() -> None:
    workflows = [
        {"path": "a.md", "intent_text": "review pull request code quality security review", "agentic_fraction": 0.4},
        {"path": "b.md", "intent_text": "review pull request security and code quality", "agentic_fraction": 0.4},
    ]
    similarities, _docs = pre.compute_similarity_matrix(workflows)
    assert max(similarities.values()) >= 0.7


def test_high_overlap_clusters_are_produced() -> None:
    workflows = [
        {"path": "a.md", "intent_text": "review pull request code quality security review", "agentic_fraction": 0.4},
        {"path": "b.md", "intent_text": "review pull request security and code quality", "agentic_fraction": 0.4},
        {"path": "c.md", "intent_text": "weekly release note generation", "agentic_fraction": 0.4},
    ]
    similarities, docs = pre.compute_similarity_matrix(workflows)
    clusters = pre.build_overlap_clusters(workflows, similarities, docs)
    assert clusters
    assert {"a.md", "b.md"}.issubset(set(clusters[0]["workflows"]))


def test_awy_formula_is_computed_correctly() -> None:
    result = pre.compute_workflow_yield(0.6, 0.5, 0.8, 0.2, 0.1, 0.1, 0.0)
    assert result == round((0.6 * 0.5 * 0.8) / (1 + 0.2 + 0.1 + 0.1 + 0.0), 4)


def test_portfolio_overlap_drag_is_computed_correctly() -> None:
    drag = pre.portfolio_overlap_drag({("a", "b"): 0.8, ("a", "c"): 0.5})
    assert drag == round((0.8**2 + 0.5**2) * 2, 4)


def test_agentic_fraction_is_computed_and_bounded() -> None:
    frontmatter = {
        "pre-agent-steps": [{"run": "python3 make_summary.py"}],
        "post-steps": [{"run": "jq . report.json"}],
        "tools": {"bash": True, "github": {"mode": "gh-proxy"}},
    }
    agentic_fraction, deterministic_fraction = pre.estimate_agentic_fraction(frontmatter, "word " * 1000)
    assert 0.0 <= agentic_fraction <= 1.0
    assert 0.0 <= deterministic_fraction <= 1.0
    assert round(agentic_fraction + deterministic_fraction, 4) == 1.0
