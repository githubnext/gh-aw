---
# Pattern: Documentation Bot
# Complexity: Intermediate
# Use Case: Generate or update documentation based on code changes
name: Documentation Bot
description: Automatically generates and updates documentation
on:
  schedule:
    # TODO: Customize schedule for doc updates
    - cron: "0 3 * * 0"  # Weekly on Sunday at 3 AM
  push:
    branches:
      - main
    paths:
      # TODO: Specify paths that trigger doc updates
      - "src/**"
      - "pkg/**"
      - "cmd/**"
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [repos, pull_requests]
  bash:
    - "git *"
    - "grep *"
    - "find *"
safe-outputs:
  create-pull-request:
timeout-minutes: 30
strict: true
---

# Documentation Bot

Automatically generate or update documentation when code changes, ensuring docs stay in sync with implementation.

## Documentation Types

# TODO: Choose which documentation to generate/update

### 1. API Documentation

Generate API docs from code comments:

```markdown
**Sources**: Code files with API endpoints
**Output**: API reference documentation
**Format**: Markdown with examples

**Example Tasks**:
- Extract API endpoints from code
- Document request/response formats
- Generate code examples
- Update OpenAPI/Swagger specs
```

### 2. README Updates

Keep README.md synchronized with code:

```markdown
**Sources**: Code structure, examples, configuration
**Output**: Updated README sections
**Updates**:
- Installation instructions
- Usage examples
- Configuration options
- CLI command documentation
- Feature list
```

### 3. Code Examples

Generate runnable code examples:

```markdown
**Sources**: API code, test files
**Output**: Example code snippets
**Includes**:
- Quick start examples
- Common use cases
- Integration examples
- Best practices
```

### 4. Changelog

Maintain CHANGELOG.md from commits:

```markdown
**Sources**: Git commits, PRs, issues
**Output**: CHANGELOG.md entries
**Format**: Keep a Changelog standard

**Categories**:
- Added (new features)
- Changed (changes in functionality)
- Deprecated (soon-to-be removed features)
- Removed (removed features)
- Fixed (bug fixes)
- Security (security fixes)
```

### 5. Architecture Docs

Document system architecture:

```markdown
**Sources**: Code structure, dependencies
**Output**: Architecture diagrams and docs
**Includes**:
- Component diagrams
- Data flow diagrams
- Dependency graphs
- Deployment architecture
```

## Implementation Steps

### Step 1: Analyze Changes

```bash
# Get list of changed files since last doc update
git diff origin/main --name-only > /tmp/changed-files.txt

# Or get changes in last commit if triggered by push
git diff HEAD~1 --name-only > /tmp/changed-files.txt

# Identify what docs need updating
echo "Changed files:"
cat /tmp/changed-files.txt

# TODO: Determine which docs to update based on changed files
```

### Step 2: Extract Documentation Content

```python
#!/usr/bin/env python3
"""
Documentation Extractor
TODO: Customize for your code structure
"""
import re
import os
from pathlib import Path

class DocExtractor:
    def __init__(self, repo_path):
        self.repo_path = Path(repo_path)
        self.docs = {}
    
    def extract_api_docs(self):
        """Extract API documentation from code"""
        # TODO: Adjust for your API framework (Flask, Express, Go, etc.)
        
        api_routes = []
        
        # Example: Extract from Python Flask routes
        for py_file in self.repo_path.rglob('*.py'):
            content = py_file.read_text()
            
            # Find route decorators
            routes = re.finditer(
                r'@app\.route\(["\']([^"\']+)["\'].*?\)\s+'
                r'def (\w+)\([^)]*\):\s+'
                r'"""([^"]+)"""',
                content,
                re.DOTALL
            )
            
            for match in routes:
                route_path, func_name, docstring = match.groups()
                api_routes.append({
                    'path': route_path,
                    'function': func_name,
                    'description': docstring.strip(),
                    'file': str(py_file.relative_to(self.repo_path))
                })
        
        self.docs['api'] = api_routes
        return api_routes
    
    def extract_cli_commands(self):
        """Extract CLI command documentation"""
        # TODO: Adjust for your CLI framework
        
        commands = []
        
        # Example: Extract from click/argparse
        for py_file in self.repo_path.rglob('*.py'):
            content = py_file.read_text()
            
            # Find CLI commands
            cli_commands = re.finditer(
                r'@click\.command\(\)\s+'
                r'def (\w+)\([^)]*\):\s+'
                r'"""([^"]+)"""',
                content,
                re.DOTALL
            )
            
            for match in cli_commands:
                cmd_name, docstring = match.groups()
                commands.append({
                    'name': cmd_name,
                    'description': docstring.strip()
                })
        
        self.docs['cli'] = commands
        return commands
    
    def extract_configuration(self):
        """Extract configuration documentation"""
        config_docs = []
        
        # Look for config files
        for config_file in ['config.yaml', 'config.json', '.env.example']:
            file_path = self.repo_path / config_file
            if file_path.exists():
                content = file_path.read_text()
                
                # Extract comments explaining config options
                lines = content.split('\n')
                for i, line in enumerate(lines):
                    if line.strip().startswith('#'):
                        comment = line.strip('#').strip()
                        if i + 1 < len(lines):
                            config_line = lines[i + 1].strip()
                            if config_line:
                                config_docs.append({
                                    'option': config_line.split(':')[0],
                                    'description': comment,
                                    'file': config_file
                                })
        
        self.docs['config'] = config_docs
        return config_docs
    
    def save_docs(self, output_path):
        """Save extracted documentation"""
        import json
        with open(output_path, 'w') as f:
            json.dump(self.docs, f, indent=2)

# Extract documentation
extractor = DocExtractor('.')
extractor.extract_api_docs()
extractor.extract_cli_commands()
extractor.extract_configuration()
extractor.save_docs('/tmp/extracted-docs.json')

print(f"Extracted documentation for {len(extractor.docs)} sections")
```

### Step 3: Generate/Update Documentation

```python
#!/usr/bin/env python3
"""
Documentation Generator
"""
import json
from pathlib import Path

def generate_api_docs(api_routes):
    """Generate API documentation"""
    doc = "# API Reference\n\n"
    
    for route in api_routes:
        doc += f"## {route['path']}\n\n"
        doc += f"**Function**: `{route['function']}`\n\n"
        doc += f"{route['description']}\n\n"
        doc += f"**Defined in**: `{route['file']}`\n\n"
        doc += "---\n\n"
    
    return doc

def generate_cli_docs(commands):
    """Generate CLI documentation"""
    doc = "# CLI Reference\n\n"
    
    for cmd in commands:
        doc += f"## `{cmd['name']}`\n\n"
        doc += f"{cmd['description']}\n\n"
    
    return doc

def update_readme_section(readme_content, section_name, new_content):
    """Update a specific section in README"""
    import re
    
    # Find the section
    pattern = f"## {section_name}.*?(?=##|$)"
    match = re.search(pattern, readme_content, re.DOTALL)
    
    if match:
        # Replace the section
        updated = readme_content.replace(
            match.group(0),
            f"## {section_name}\n\n{new_content}\n\n"
        )
        return updated
    else:
        # Add new section at end
        return readme_content + f"\n## {section_name}\n\n{new_content}\n"

# Load extracted docs
with open('/tmp/extracted-docs.json') as f:
    docs = json.load(f)

# Generate documentation
api_doc = generate_api_docs(docs['api'])
cli_doc = generate_cli_docs(docs['cli'])

# Update README
readme_path = Path('README.md')
if readme_path.exists():
    readme_content = readme_path.read_text()
    
    # Update CLI section
    readme_content = update_readme_section(readme_content, 'CLI Reference', cli_doc)
    
    # Save updated README
    readme_path.write_text(readme_content)
    print("README.md updated")

# Save API docs
Path('docs/api.md').parent.mkdir(exist_ok=True)
Path('docs/api.md').write_text(api_doc)
print("API documentation generated")
```

### Step 4: Validate Documentation

```bash
# Check for broken links
echo "Checking for broken links..."

# TODO: Add link checking logic
for doc_file in docs/*.md README.md; do
  # Extract links
  links=$(grep -oP '\[.*?\]\(\K[^)]+' "$doc_file" 2>/dev/null || true)
  
  # Check internal links
  for link in $links; do
    if [[ $link == /* ]]; then
      if [ ! -f ".$link" ]; then
        echo "⚠️  Broken link in $doc_file: $link"
      fi
    fi
  done
done

# Check for TODOs in docs
echo "Checking for unfinished documentation..."
grep -r "TODO" docs/ README.md && echo "⚠️  Found TODOs in documentation"

# Spell check (if available)
if command -v aspell &> /dev/null; then
  echo "Running spell check..."
  find docs -name "*.md" -exec aspell check {} \;
fi
```

### Step 5: Create Pull Request

```markdown
Use create-pull-request safe-output:

**Title**: 📚 Update documentation

**Description**:
## Documentation Updates

This PR updates documentation to reflect recent code changes.

### Changes Made

#### Updated Files
- [x] `README.md` - Updated CLI reference
- [x] `docs/api.md` - Regenerated API documentation
- [x] `docs/examples.md` - Added new examples

#### New Documentation
- [x] Added configuration reference
- [x] Added architecture diagram

### What Changed in Code

Recent changes that triggered doc updates:
- Added new API endpoint: `POST /api/users`
- Updated CLI command: `app deploy`
- Changed configuration format

### Validation

- [x] All links checked (no broken links)
- [x] Code examples tested
- [x] Spell check completed
- [ ] Manual review needed

### Review Checklist

- [ ] Documentation is accurate
- [ ] Examples work correctly
- [ ] Formatting is consistent
- [ ] No sensitive information exposed

---
*Generated by [Documentation Bot]({run_url})*
```

## Customization Guide

### Configure Doc Triggers

```yaml
# TODO: Customize which changes trigger doc updates

on:
  push:
    branches: [main]
    paths:
      - "src/**/*.py"        # Python source changes
      - "cmd/**/*.go"        # Go CLI changes
      - "api/**"             # API changes
      - "!**/*_test.*"       # Ignore test files
```

### Add Custom Sections

```python
# TODO: Add documentation sections specific to your project

def generate_deployment_docs():
    """Generate deployment documentation"""
    # Extract from Dockerfile, k8s configs, etc.
    pass

def generate_troubleshooting_guide():
    """Generate troubleshooting guide from known issues"""
    # Extract from closed issues labeled "troubleshooting"
    pass
```

### Configure Output Format

```yaml
# TODO: Choose documentation format

output-format:
  api-docs: markdown  # or: openapi, swagger, postman
  cli-docs: markdown  # or: man-pages, html
  examples: markdown  # or: jupyter-notebook
```

## Example Outputs

```markdown
## 📚 Documentation Update PR

### Summary

Updated documentation to reflect changes in v2.1.0.

### Changes

#### API Documentation (docs/api.md)
**New Endpoints**:
- `POST /api/v2/webhooks` - Register webhook endpoints
- `GET /api/v2/metrics` - Retrieve system metrics

**Updated Endpoints**:
- `POST /api/users` - Now requires email verification

#### README.md
**Updated Sections**:
- Installation - Added Docker installation method
- Configuration - Documented new `WEBHOOK_SECRET` env var
- CLI Reference - Updated `deploy` command with new flags

#### Examples
**New Examples**:
- `examples/webhook-setup.md` - Webhook configuration guide
- `examples/metrics-dashboard.md` - Metrics integration example

### Validation Results

✓ All code examples tested  
✓ No broken internal links  
✓ Spell check passed  
⚠️  2 external links need verification (marked in comments)

---
*Auto-generated by Documentation Bot*
```

## Tips

- **Keep docs close to code**: Co-locate docs with source files
- **Automate where possible**: Let AI handle repetitive doc tasks
- **Test examples**: Ensure code examples actually work
- **Version docs**: Keep old versions for reference
- **Link liberally**: Connect related documentation
- **Review regularly**: Schedule periodic doc audits

## Related Examples

- **Production examples**:
  - `.github/workflows/technical-doc-writer.md` - Technical documentation
  - `.github/workflows/daily-doc-updater.md` - Daily doc maintenance

## Security Considerations

- Don't expose sensitive information in docs
- Sanitize code examples (remove real keys, passwords)
- Review generated docs before merging
- Use read-only permissions where possible

---

**Pattern Info**:
- Complexity: Intermediate
- Trigger: Scheduled, push, or manual
- Safe Outputs: create_pull_request
- Tools: GitHub (repos, pull_requests), bash (git, grep)
