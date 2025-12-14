# Secure Template Substitution Patterns

> **Note:** This is a documentation file showing secure patterns for handling user-controlled data in custom GitHub Actions workflows.
>
> **Important for gh-aw users:** gh-aw's built-in security validation intentionally blocks direct use of expressions like `github.event.issue.body`, `github.event.comment.body`, etc., and automatically sanitizes this data before passing it to your agent through `needs.activation.outputs.text`. The patterns shown here explain WHY these protections exist and demonstrate secure alternatives for custom workflows outside of gh-aw.

## Security Context

User-controlled data (issue bodies, PR titles, comments, etc.) must never be directly interpolated into shell commands or processed with tools like `envsubst` that perform shell expansion. This can lead to code injection attacks.

## The Secure Pattern

### Step 1: Capture Untrusted Data in Environment Variables

Always pass untrusted data through environment variables, not directly in template expressions.

```bash
env:
  # Security: Pass untrusted data through environment variables
  ISSUE_TITLE: ${{ github.event.issue.title }}
  ISSUE_BODY: ${{ github.event.issue.body }}
  COMMENT_BODY: ${{ github.event.comment.body }}
  USER_LOGIN: ${{ github.event.comment.user.login }}
run: |
  echo "Processing issue: $ISSUE_TITLE"
```

### Step 2: Create Template with Placeholders

Use distinctive placeholder tokens (like `__VARIABLE__`) that are easy to identify and search for.

```bash
run: |
  # Create template file with placeholders
  # Note: Use single quotes in heredoc to prevent shell expansion
  cat << 'TEMPLATE_EOF' > /tmp/analysis-prompt.txt
  # Issue Analysis Request
  
  **Reported by:** __USER_LOGIN__
  
  ## Issue Title
  __ISSUE_TITLE__
  
  ## Issue Description
  __ISSUE_BODY__
  
  ## Latest Comment
  __COMMENT_BODY__
  
  Please analyze this issue and provide recommendations.
  TEMPLATE_EOF
```

### Step 3: Safe Substitution with Sed

Use `sed` with proper escaping to replace placeholders. This treats all input as literal strings, preventing code execution.

```bash
run: |
  # Security: Safe substitution using sed (no shell expansion)
  # The ${VAR//|/\\|} syntax escapes pipe characters in the variable
  sed -i "s|__USER_LOGIN__|${USER_LOGIN//|/\\|}|g" /tmp/analysis-prompt.txt
  sed -i "s|__ISSUE_TITLE__|${ISSUE_TITLE//|/\\|}|g" /tmp/analysis-prompt.txt
  sed -i "s|__ISSUE_BODY__|${ISSUE_BODY//|/\\|}|g" /tmp/analysis-prompt.txt
  sed -i "s|__COMMENT_BODY__|${COMMENT_BODY//|/\\|}|g" /tmp/analysis-prompt.txt
  
  echo "Template created successfully at /tmp/analysis-prompt.txt"
```

### Step 4: Verify and Use the Template

```bash
run: |
  # Verify the template was created
  if [ ! -f /tmp/analysis-prompt.txt ]; then
    echo "Error: Template file not created"
    exit 1
  fi
  
  # Use the template (e.g., pass to AI agent)
  cat /tmp/analysis-prompt.txt
```

## Complete Secure Example

Here's a complete workflow step using the secure pattern:

```yaml
- name: Create Analysis Prompt with User Data
  env:
    # Security: All untrusted data passed through environment variables
    ISSUE_TITLE: ${{ github.event.issue.title }}
    ISSUE_BODY: ${{ github.event.issue.body }}
    COMMENT_BODY: ${{ github.event.comment.body }}
    USER_LOGIN: ${{ github.event.comment.user.login }}
  run: |
    # Create output directory
    mkdir -p /tmp/prompts
    OUTPUT_FILE="/tmp/prompts/issue-analysis.txt"
    
    # Create template with placeholders (single quotes prevent expansion)
    cat << 'TEMPLATE_EOF' > "$OUTPUT_FILE"
    # Issue Analysis Request
    
    **Reported by:** __USER_LOGIN__
    
    ## Issue Title
    __ISSUE_TITLE__
    
    ## Issue Description
    __ISSUE_BODY__
    
    ## Latest Comment
    __COMMENT_BODY__
    
    Please analyze this issue and provide:
    1. Root cause analysis
    2. Recommended next steps
    3. Related issues or patterns
    TEMPLATE_EOF
    
    # Security: Safe substitution with sed (no shell expansion)
    # Escapes pipe characters to prevent sed delimiter conflicts
    sed -i "s|__USER_LOGIN__|${USER_LOGIN//|/\\|}|g" "$OUTPUT_FILE"
    sed -i "s|__ISSUE_TITLE__|${ISSUE_TITLE//|/\\|}|g" "$OUTPUT_FILE"
    sed -i "s|__ISSUE_BODY__|${ISSUE_BODY//|/\\|}|g" "$OUTPUT_FILE"
    sed -i "s|__COMMENT_BODY__|${COMMENT_BODY//|/\\|}|g" "$OUTPUT_FILE"
    
    # Verify and display results
    echo "✓ Template created successfully"
    echo "File: $OUTPUT_FILE"
    echo "Size: $(wc -c < "$OUTPUT_FILE") bytes"
    
    # Display first 20 lines for verification
    echo "Preview:"
    head -n 20 "$OUTPUT_FILE"
```

## What NOT To Do

### ❌ Unsafe: Direct Template Expression in Shell

```yaml
# NEVER DO THIS
run: |
  echo "Issue: ${{ github.event.issue.body }}"
```

**Why it's unsafe:** Template expressions are evaluated before the shell runs, allowing code injection through the issue body.

### ❌ Unsafe: Using envsubst

```yaml
# NEVER DO THIS
env:
  ISSUE_BODY: ${{ github.event.issue.body }}
run: |
  export ISSUE_BODY
  envsubst < template.txt > output.txt
```

**Why it's unsafe:** `envsubst` performs shell variable expansion, which can execute arbitrary commands embedded in the data.

### ❌ Unsafe: Direct String Concatenation

```yaml
# NEVER DO THIS
run: |
  PROMPT="Analyze this: ${{ github.event.issue.body }}"
  echo "$PROMPT"
```

**Why it's unsafe:** Template expressions are evaluated during workflow compilation, not at runtime.

## Security Validation

### Manual Review Checklist

When reviewing workflows for template injection vulnerabilities:

- [ ] All user-controlled data passed through environment variables
- [ ] No `envsubst` usage on untrusted data
- [ ] Templates use distinctive placeholders (e.g., `__VAR__`)
- [ ] Sed substitution includes proper pipe escaping: `${VAR//|/\\|}`
- [ ] Heredocs use single quotes to prevent premature expansion: `<< 'EOF'`
- [ ] No direct template expressions in shell commands

### Automated Validation

```bash
# Compile workflow with security scanning enabled
./gh-aw compile secure-templating --zizmor --actionlint --poutine

# Zizmor checks for template injection patterns
# Actionlint validates workflow syntax
# Poutine detects security misconfigurations
```

## References

- [DEVGUIDE.md - Secure Template Substitution](../DEVGUIDE.md#secure-template-substitution)
- [SECURITY.md - Template Injection Prevention](../SECURITY.md#template-injection-prevention)
- [specs/template-injection-prevention.md](../specs/template-injection-prevention.md)
- [GitHub Actions Security Hardening](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
- [Understanding Script Injection Risk](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions#understanding-the-risk-of-script-injections)

## How gh-aw Protects You

**For gh-aw agent workflows:** You don't need to implement these patterns manually! gh-aw automatically:

1. **Blocks unsafe expressions** - The compiler rejects direct use of `github.event.issue.body`, `github.event.comment.body`, etc.
2. **Sanitizes user input** - All untrusted data is processed through the activation step
3. **Provides safe contexts** - Access sanitized data through `needs.activation.outputs.text`

If you see compilation errors about "unauthorized expressions," that's gh-aw protecting you from template injection vulnerabilities!

## When to Use These Patterns

Use the manual patterns shown in this document when:
- Creating **custom GitHub Actions workflows** outside of gh-aw
- Writing **bash processing steps** in existing Actions workflows
- Building **templates from environment variables** manually
- Understanding **why gh-aw blocks certain expressions**

For gh-aw agent workflows, the framework handles security automatically - just use the sanitized inputs provided!

## Key Takeaways

1. ✅ **Always use environment variables** for untrusted data
2. ✅ **Use placeholders** like `__VAR__` in templates
3. ✅ **Use sed for substitution** with proper escaping
4. ✅ **Single-quote heredocs** to prevent shell expansion
5. ❌ **Never use envsubst** on user-controlled data
6. ❌ **Never use template expressions** directly in shell commands
7. 🔍 **Always validate** with security scanning tools
