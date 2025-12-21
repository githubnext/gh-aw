#!/usr/bin/env python3
"""
Audit safe-outputs configuration across all workflows.

This script:
1. Scans all workflow markdown files
2. Categorizes workflows by type
3. Compares limits against guidelines
4. Identifies outliers and generates recommendations
"""

import os
import sys
import yaml
import re
from pathlib import Path
from typing import Dict, List, Optional, Tuple
from collections import defaultdict

# Guidelines by category (from docs/src/content/docs/guides/safe-output-limits.md)
GUIDELINES = {
    "Meta-Orchestrator": {
        "create-issue": (3, 5),
        "add-comment": (10, 15),
        "create-discussion": (2, 3),
    },
    "Daily/Hourly Monitor": {
        "create-issue": (0, 1),
        "create-discussion": 1,
        "add-comment": (1, 3),
    },
    "Worker": {
        "create-issue": (0, 1),
        "create-pull-request": (1, 3),
        "add-comment": (1, 5),
    },
    "Campaign": {
        "create-issue": (5, 10),
        "update-project": (20, 50),
        "add-comment": (10, 20),
        "create-pull-request": (1, 5),
    },
    "Analysis/Research": {
        "create-discussion": 1,
        "create-issue": (0, 2),
        "add-comment": (1, 3),
    },
    "PR/Event Responder": {
        "add-comment": (1, 3),
        "create-issue": 0,
        "update-pull-request": 1,
    },
}

# Default max values when not specified
DEFAULTS = {
    "create-issue": 1,
    "create-pull-request": 1,
    "create-discussion": 1,
    "add-comment": 1,
    "close-issue": 1,
    "close-pull-request": 1,
    "update-issue": 1,
    "update-pull-request": 1,
    "update-project": 10,
    "add-labels": 3,
    "add-reviewer": 3,
}


def categorize_workflow(name: str, description: str, on_config: any) -> str:
    """Determine workflow category based on name, description, and triggers."""
    name_lower = name.lower()
    desc_lower = description.lower() if description else ""
    
    # Meta-Orchestrators
    if any(x in name_lower for x in ["campaign-manager", "workflow-health", "agent-performance"]):
        return "Meta-Orchestrator"
    if "meta-orchestrator" in desc_lower:
        return "Meta-Orchestrator"
    
    # Daily/Hourly Monitors
    if any(x in name_lower for x in ["daily-", "hourly-", "weekly-"]):
        return "Daily/Hourly Monitor"
    if isinstance(on_config, dict) and "schedule" in on_config:
        sched = on_config["schedule"]
        if isinstance(sched, str) and sched in ["daily", "hourly", "weekly"]:
            return "Daily/Hourly Monitor"
    
    # PR/Event Responders
    if any(x in name_lower for x in ["dev-hawk", "reviewer", "ci-coach", "ci-doctor"]):
        return "PR/Event Responder"
    if "responder" in desc_lower or "monitors development" in desc_lower:
        return "PR/Event Responder"
    if isinstance(on_config, dict) and any(k in on_config for k in ["pull_request", "pull_request_target", "workflow_run"]):
        return "PR/Event Responder"
    
    # Campaigns
    if "campaign" in name_lower and "manager" not in name_lower:
        return "Campaign"
    if "campaign" in desc_lower and "coordinate" in desc_lower:
        return "Campaign"
    
    # Workers
    if any(x in name_lower for x in ["fixer", "updater", "tidy", "formatter"]):
        return "Worker"
    if any(x in desc_lower for x in ["automatically", "fixes", "updates", "formats"]):
        return "Worker"
    
    # Analysis/Research
    if any(x in name_lower for x in ["analysis", "analyzer", "audit", "report", "research", "metrics"]):
        return "Analysis/Research"
    if any(x in desc_lower for x in ["analyzes", "audits", "reports", "research", "tracks"]):
        return "Analysis/Research"
    
    return "Other"


def get_guideline_range(category: str, output_type: str) -> Optional[Tuple[int, int]]:
    """Get the recommended range for a given category and output type."""
    if category not in GUIDELINES:
        return None
    
    cat_guidelines = GUIDELINES[category]
    if output_type not in cat_guidelines:
        return None
    
    value = cat_guidelines[output_type]
    if isinstance(value, tuple):
        return value
    else:
        return (value, value)


def is_within_guidelines(value: int, category: str, output_type: str) -> Tuple[bool, Optional[str]]:
    """Check if a value is within guidelines and return (is_compliant, message)."""
    guideline = get_guideline_range(category, output_type)
    
    if guideline is None:
        return (True, None)  # No guideline for this combination
    
    min_val, max_val = guideline
    
    if value < min_val:
        return (True, f"Below range ({min_val}-{max_val}), consider increasing")
    elif value > max_val:
        return (False, f"Exceeds guideline ({min_val}-{max_val})")
    else:
        return (True, f"Within guideline ({min_val}-{max_val})")


def extract_frontmatter(content: str) -> Optional[Dict]:
    """Extract YAML frontmatter from markdown content."""
    match = re.match(r'^---\n(.*?)\n---', content, re.DOTALL)
    if not match:
        return None
    
    frontmatter_text = match.group(1)
    try:
        return yaml.safe_load(frontmatter_text)
    except yaml.YAMLError:
        return None


def audit_workflow(workflow_path: Path) -> Optional[Dict]:
    """Audit a single workflow file."""
    with open(workflow_path, 'r') as f:
        content = f.read()
    
    frontmatter = extract_frontmatter(content)
    if not frontmatter or 'safe-outputs' not in frontmatter:
        return None
    
    safe_outputs = frontmatter['safe-outputs']
    workflow_name = workflow_path.stem
    description = frontmatter.get('description', '')
    on_config = frontmatter.get('on', {})
    
    # Categorize workflow
    category = categorize_workflow(workflow_name, description, on_config)
    
    # Extract limits
    limits = {}
    issues = []
    
    for output_type in ['create-issue', 'create-pull-request', 'add-comment', 
                        'create-discussion', 'update-project', 'close-issue',
                        'close-pull-request', 'update-issue', 'update-pull-request']:
        if output_type in safe_outputs:
            config = safe_outputs[output_type]
            if isinstance(config, dict):
                max_val = config.get('max', DEFAULTS.get(output_type, 1))
            else:
                max_val = DEFAULTS.get(output_type, 1)
            
            limits[output_type] = max_val
            
            # Check against guidelines
            is_compliant, message = is_within_guidelines(max_val, category, output_type)
            if not is_compliant:
                issues.append({
                    'type': output_type,
                    'value': max_val,
                    'message': message
                })
    
    return {
        'workflow': workflow_name,
        'category': category,
        'description': description[:100] if description else '',
        'limits': limits,
        'issues': issues,
    }


def generate_report(results: List[Dict]) -> str:
    """Generate a comprehensive audit report."""
    lines = []
    
    lines.append("=" * 80)
    lines.append("SAFE OUTPUT LIMITS AUDIT REPORT")
    lines.append("=" * 80)
    lines.append("")
    
    # Group by category
    by_category = defaultdict(list)
    for result in results:
        by_category[result['category']].append(result)
    
    # Sort categories
    category_order = [
        "Meta-Orchestrator",
        "Daily/Hourly Monitor", 
        "Worker",
        "Campaign",
        "Analysis/Research",
        "PR/Event Responder",
        "Other"
    ]
    
    total_workflows = len(results)
    total_issues = sum(len(r['issues']) for r in results)
    
    lines.append(f"Total Workflows Analyzed: {total_workflows}")
    lines.append(f"Total Issues Found: {total_issues}")
    lines.append("")
    
    # Summary by category
    lines.append("=" * 80)
    lines.append("SUMMARY BY CATEGORY")
    lines.append("=" * 80)
    lines.append("")
    
    for category in category_order:
        workflows = by_category.get(category, [])
        if not workflows:
            continue
        
        workflows_with_issues = [w for w in workflows if w['issues']]
        lines.append(f"{category}:")
        lines.append(f"  Total Workflows: {len(workflows)}")
        lines.append(f"  Workflows with Issues: {len(workflows_with_issues)}")
        
        if workflows_with_issues:
            lines.append(f"  Workflows exceeding guidelines:")
            for w in workflows_with_issues:
                lines.append(f"    - {w['workflow']}")
        lines.append("")
    
    # Detailed findings
    lines.append("=" * 80)
    lines.append("DETAILED FINDINGS")
    lines.append("=" * 80)
    lines.append("")
    
    for category in category_order:
        workflows = by_category.get(category, [])
        if not workflows:
            continue
        
        workflows_with_issues = [w for w in workflows if w['issues']]
        if not workflows_with_issues:
            continue
        
        lines.append("-" * 80)
        lines.append(f"CATEGORY: {category}")
        lines.append("-" * 80)
        lines.append("")
        
        for workflow in workflows_with_issues:
            lines.append(f"Workflow: {workflow['workflow']}")
            if workflow['description']:
                lines.append(f"Description: {workflow['description']}")
            lines.append(f"Current Limits: {workflow['limits']}")
            lines.append("Issues:")
            for issue in workflow['issues']:
                lines.append(f"  ⚠️  {issue['type']}: {issue['value']} - {issue['message']}")
            lines.append("")
    
    # Compliant workflows
    lines.append("=" * 80)
    lines.append("COMPLIANT WORKFLOWS (Sample)")
    lines.append("=" * 80)
    lines.append("")
    
    compliant_count = 0
    for category in category_order:
        workflows = by_category.get(category, [])
        compliant = [w for w in workflows if not w['issues']]
        
        if compliant and compliant_count < 10:  # Show up to 10 examples
            lines.append(f"{category}:")
            for w in compliant[:3]:  # Show up to 3 per category
                lines.append(f"  ✅ {w['workflow']}: {w['limits']}")
                compliant_count += 1
                if compliant_count >= 10:
                    break
            lines.append("")
    
    lines.append("=" * 80)
    lines.append("END OF REPORT")
    lines.append("=" * 80)
    
    return "\n".join(lines)


def main():
    """Main audit function."""
    workflows_dir = Path('.github/workflows')
    
    if not workflows_dir.exists():
        print("Error: .github/workflows directory not found", file=sys.stderr)
        print("Please run this script from the repository root", file=sys.stderr)
        sys.exit(1)
    
    workflow_files = list(workflows_dir.glob('*.md'))
    
    if not workflow_files:
        print("No workflow markdown files found", file=sys.stderr)
        sys.exit(1)
    
    print(f"Scanning {len(workflow_files)} workflow files...", file=sys.stderr)
    print("", file=sys.stderr)
    
    results = []
    for workflow_path in workflow_files:
        result = audit_workflow(workflow_path)
        if result:
            results.append(result)
    
    # Generate and print report
    report = generate_report(results)
    print(report)


if __name__ == '__main__':
    main()
