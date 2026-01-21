# Agentic Workflow Runs Analysis

**Generated:** January 21, 2026  
**Repository:** githubnext/gh-aw  
**Workflows Analyzed:** code-scanning-fixer, security-fix-pr, security-review

## Executive Summary

This report analyzes the latest runs of three security-focused agentic workflows in the gh-aw repository. The workflows use AI agents to automatically fix security vulnerabilities and review code changes for security implications.

### Quick Stats

| Workflow | Success Rate (Last 10) | Latest Status | Total Runs | Schedule |
|----------|------------------------|---------------|------------|----------|
| 🔍 Code Scanning Fixer | 60% | ❌ Failed | 279 | Every 30 minutes |
| 🛡️ Security Fix PR | 100% | ✅ Success | 419 | Every 4 hours |
| 🔒 Security Review | N/A (Event-driven) | ⏭️ Skipped | 445 | Slash command |

## Detailed Analysis

### 🔍 Code Scanning Fixer

**Purpose:** Automatically fixes high severity code scanning alerts by creating pull requests with remediation.

**Latest Run:** [#279](https://github.com/githubnext/gh-aw/actions/runs/21199967712) - January 21, 2026 06:43 UTC - ❌ **Failed**

**Performance Analysis:**
- ⚠️ **60% Success Rate** in last 10 runs (6 success, 4 failures)
- Failures occur intermittently every 2-3 runs
- Most recent failure: Run #279, #274, #272, #270

**Failure Pattern:**
- Failures occur at step 24: "Execute GitHub Copilot CLI"
- Potential causes:
  - Code scanning alert API availability issues
  - Copilot CLI execution timeouts
  - MCP server connectivity problems

**Configuration:**
- **Engine:** GitHub Copilot
- **Timeout:** 20 minutes
- **Tools:** github (context, repos, code_security, pull_requests), edit, bash, cache-memory
- **Safe Outputs:** create-pull-request with "[code-scanning-fix]" prefix

**Recent Run History:**

| Run | Date | Status | Link |
|-----|------|--------|------|
| #279 | Jan 21, 06:43 UTC | ❌ Failure | [View](https://github.com/githubnext/gh-aw/actions/runs/21199967712) |
| #278 | Jan 21, 06:03 UTC | ✅ Success | [View](https://github.com/githubnext/gh-aw/actions/runs/21199116290) |
| #277 | Jan 21, 05:40 UTC | ✅ Success | [View](https://github.com/githubnext/gh-aw/actions/runs/21198652713) |
| #276 | Jan 21, 05:07 UTC | ✅ Success | [View](https://github.com/githubnext/gh-aw/actions/runs/21198001333) |
| #275 | Jan 21, 04:44 UTC | ✅ Success | [View](https://github.com/githubnext/gh-aw/actions/runs/21197544550) |
| #274 | Jan 21, 04:17 UTC | ❌ Failure | [View](https://github.com/githubnext/gh-aw/actions/runs/21197049083) |
| #273 | Jan 21, 03:30 UTC | ✅ Success | [View](https://github.com/githubnext/gh-aw/actions/runs/21195733978) |
| #272 | Jan 21, 02:38 UTC | ❌ Failure | [View](https://github.com/githubnext/gh-aw/actions/runs/21195191029) |
| #271 | Jan 21, 01:30 UTC | ✅ Success | [View](https://github.com/githubnext/gh-aw/actions/runs/21193631622) |
| #270 | Jan 21, 00:46 UTC | ❌ Failure | [View](https://github.com/githubnext/gh-aw/actions/runs/21192886778) |

---

### 🛡️ Security Fix PR

**Purpose:** Identifies and automatically fixes code security issues by creating autofixes via GitHub Code Scanning.

**Latest Run:** [#419](https://github.com/githubnext/gh-aw/actions/runs/21197224076) - January 21, 2026 04:26 UTC - ✅ **Success**

**Performance Analysis:**
- ✅ **100% Success Rate** in last 10 runs
- No failures detected in recent history
- Most reliable of the three security workflows

**Configuration:**
- **Engine:** GitHub Copilot
- **Timeout:** 20 minutes
- **Tools:** github (context, repos, code_security, pull_requests), cache-memory
- **Safe Outputs:** autofix-code-scanning-alert (max: 5 alerts per run)

**Recent Run History:**

| Run | Date | Status | Link |
|-----|------|--------|------|
| #419 | Jan 21, 04:26 UTC | ✅ Success | [View](https://github.com/githubnext/gh-aw/actions/runs/21197224076) |
| #418 | Jan 21, 00:40 UTC | ✅ Success | [View](https://github.com/githubnext/gh-aw/actions/runs/21192746757) |
| #417 | Jan 20, 20:28 UTC | ✅ Success | [View](https://github.com/githubnext/gh-aw/actions/runs/21186127873) |
| #416 | Jan 20, 16:14 UTC | ✅ Success | [View](https://github.com/githubnext/gh-aw/actions/runs/21178735965) |
| #415 | Jan 20, 12:17 UTC | ✅ Success | [View](https://github.com/githubnext/gh-aw/actions/runs/21171181019) |

**Key Strengths:**
- Excellent reliability and stability
- Processes up to 5 alerts per run
- Uses GitHub's autofix API directly
- No recent failures or issues

---

### 🔒 Security Review Agent

**Purpose:** Reviews pull requests to identify changes that could weaken security posture or extend the security boundaries of the Agentic Workflow Firewall (AWF).

**Latest Run:** [#445](https://github.com/githubnext/gh-aw/actions/runs/21200221014) - January 21, 2026 06:55 UTC - ⏭️ **Skipped**

**Performance Analysis:**
- ⏭️ **90% Skip Rate** in last 10 runs (expected for event-driven workflow)
- 1 run required action (active review in progress)
- Skipped runs are normal behavior - workflow only activates when `/security-review` command is used

**Configuration:**
- **Engine:** Default (Copilot)
- **Timeout:** 15 minutes
- **Trigger:** Slash command (`/security-review`)
- **Tools:** github (all toolsets), agentic-workflows, bash (*), edit, web-fetch, cache-memory
- **Safe Outputs:** 
  - add-comment (max: 1)
  - create-pull-request-review-comment (max: 10, side: RIGHT)

**Recent Run History:**

| Run | Date | Status | Link |
|-----|------|--------|------|
| #445 | Jan 21, 06:55 UTC | ⏭️ Skipped | [View](https://github.com/githubnext/gh-aw/actions/runs/21200221014) |
| #444 | Jan 21, 06:17 UTC | ⏭️ Skipped | [View](https://github.com/githubnext/gh-aw/actions/runs/21199412818) |
| #443 | Jan 21, 06:17 UTC | ⏭️ Skipped | [View](https://github.com/githubnext/gh-aw/actions/runs/21199409219) |
| #442 | Jan 21, 06:16 UTC | ⏭️ Skipped | [View](https://github.com/githubnext/gh-aw/actions/runs/21199402689) |
| #441 | Jan 21, 06:13 UTC | ⏭️ Skipped | [View](https://github.com/githubnext/gh-aw/actions/runs/21199317146) |

**Security Review Focus Areas:**
- AWF (Agent Workflow Firewall) configuration changes
- Network access and domain allow/block lists
- Sandbox configuration modifications
- Permission escalations (read to write)
- Tool and MCP server configuration changes
- Safe outputs and inputs configuration
- Workflow trigger security (forks, roles, bots)
- Go code validation logic changes
- JSON schema security patterns
- JavaScript security vulnerabilities

---

## Comparison Matrix

| Metric | Code Scanning Fixer | Security Fix PR | Security Review |
|--------|---------------------|-----------------|-----------------|
| **Trigger Type** | Scheduled (30m) | Scheduled (4h) | Slash Command |
| **Total Runs** | 279 | 419 | 445 |
| **Success Rate** | 60% (last 10) | 100% (last 10) | N/A (event-driven) |
| **Output Type** | Pull Request | Code Scanning Autofix | PR Review Comments |
| **Max Fixes/Run** | 1 alert | 5 alerts | 10 comments |
| **Primary Tool** | GitHub + Edit | GitHub | GitHub + All Tools |
| **Timeout** | 20 minutes | 20 minutes | 15 minutes |

## Key Findings

### ✅ Strengths

1. **Security Fix PR** demonstrates excellent reliability with 100% success rate
2. All three workflows use appropriate tooling and safe output mechanisms
3. Event-driven **Security Review** workflow properly handles conditional execution
4. Workflows are well-documented with clear purposes and configurations

### ⚠️ Areas of Concern

1. **Code Scanning Fixer** experiencing intermittent failures (40% failure rate in last 10 runs)
2. Failures occur at Copilot CLI execution step, suggesting timeout or connectivity issues
3. Failure pattern shows regularity (approximately every 2-3 runs), indicating systemic issue

### 🔍 Root Cause Analysis - Code Scanning Fixer Failures

Based on job analysis, failures occur at:
- **Step 24:** "Execute GitHub Copilot CLI"
- **Failure Point:** Agent job completes with failure conclusion
- **Supporting Steps:** All setup steps (pre-activation, activation, checkout, setup) succeed

**Potential Root Causes:**
1. **Timeout Issues:** Complex fixes may exceed implicit timeout within the 20-minute window
2. **MCP Gateway Connectivity:** Intermittent connectivity to MCP servers
3. **Code Scanning API:** Rate limiting or availability issues with GitHub's Code Scanning API
4. **Cache Memory:** Possible issues with cache-memory file operations

## Recommendations

### 🚨 High Priority - Code Scanning Fixer

1. **Investigate Failure Logs**
   - Download and analyze logs from failed runs (#279, #274, #272, #270)
   - Focus on Copilot CLI execution output and error messages
   - Check MCP gateway logs for connectivity issues

2. **Increase Timeout**
   - Consider increasing workflow timeout from 20 to 25 minutes
   - Allow more time for complex security fixes

3. **Add Retry Logic**
   - Implement retry mechanism for transient API failures
   - Add exponential backoff for rate-limited requests

4. **Enhance Monitoring**
   - Add detailed logging for MCP gateway connectivity
   - Track cache-memory operations
   - Monitor Code Scanning API response times

5. **Review Cache Memory**
   - Verify `fixed-alerts.jsonl` format and integrity
   - Ensure proper alert deduplication logic

### ✅ Maintain Excellence - Security Fix PR

- Continue current configuration (no changes needed)
- Monitor for any degradation in performance
- Document success patterns for other workflows

### 📝 Enhance Usage - Security Review

- Promote usage of `/security-review` command in PR workflows
- Consider creating documentation on when to invoke security review
- Track action_required runs to measure impact

## Conclusion

The security workflow ecosystem in gh-aw is robust, with two workflows performing excellently. The **Security Fix PR** workflow demonstrates best-in-class reliability, while the **Security Review** workflow operates correctly as an event-driven system.

The primary concern is the **Code Scanning Fixer** workflow's 40% failure rate, which requires immediate investigation. The failure pattern suggests a systemic issue rather than random failures, making it both predictable and fixable.

**Immediate Action Items:**
1. ⚠️ Investigate Code Scanning Fixer failures (Priority: High)
2. 📊 Download and analyze failure logs
3. 🔧 Implement recommended improvements (timeout, retry logic)
4. 📈 Monitor success rate after changes

---

**Report Files:**
- HTML Report: `workflow-runs-analysis.html` (detailed interactive report)
- Markdown Summary: `workflow-runs-analysis.md` (this file)

**Data Sources:**
- GitHub Actions API via GitHub MCP Server
- Workflow run data from githubnext/gh-aw
- Analysis period: Last 10-20 runs per workflow
