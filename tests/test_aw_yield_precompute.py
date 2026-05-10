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
