# Workflow Timing Analysis

**Workflow:** Daily Repo Status  
**Run ID:** [21479574343](https://github.com/githubnext/gh-aw-trial-hono/actions/runs/21479574343)  
**Date:** January 29, 2026  
**Total Duration:** 3.8 minutes (225.9 seconds)

## Executive Summary

The workflow consists of 6 jobs running sequentially. The **agent job dominates execution time at 64.7%** (146.2s), with actual coding agent work taking 85.4s after a 32.7s startup delay. Pre-activation and activation jobs are minimal overhead (2.2% and 2.7% respectively).

**Critical Path:** Pre-activation → Activation → **Agent** → Detection → Safe Outputs → Conclusion

---

## Complete Workflow Timeline

```
13:15:17 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 13:19:03
         ╱pre╲──╱activation╲────────╱ agent ╲────╱detection╲╱safe╲╱conclusion╲
          5.0s     6.2s              146.2s       21.7s     7.4s    7.4s
```

### Job-Level Breakdown

| Job | Duration | % of Total | Start → End | Purpose |
|-----|----------|-----------|-------------|---------|
| **pre_activation** | 5.0s | 2.2% | 13:15:17 → 13:15:22 | Workflow validation, permission checks |
| **activation** | 6.2s | 2.7% | 13:15:29 → 13:15:35 | Environment setup, context preparation |
| **agent** | 146.2s | **64.7%** | 13:15:41 → 13:18:07 | ⚡ AI agent execution (main work) |
| **detection** | 21.7s | 9.6% | 13:18:14 → 13:18:36 | Security threat analysis |
| **safe_outputs** | 7.4s | 3.3% | 13:18:41 → 13:18:48 | Output validation and sanitization |
| **conclusion** | 7.4s | 3.3% | 13:18:56 → 13:19:03 | Workflow summary and cleanup |
| **TOTAL** | **225.9s** | **100%** | | **3 min 46s** |

**Note:** Gaps between jobs (e.g., 7s between activation and agent) represent GitHub Actions job scheduling overhead.

---

## Agent Job Deep Dive (146.2s, 64.7% of workflow)

The agent job is the performance bottleneck. Breaking it into 5 key stages:

### Stage-by-Stage Breakdown

| Stage | Duration | % of Agent | Cumulative | Activities |
|-------|----------|-----------|------------|------------|
| **1. Infrastructure Setup** | 23.7s | 16.7% | 23.7s | Runner init, action downloads, checkout, git config |
| **2. Dependencies Install** | 8.8s | 6.2% | 32.5s | Claude Code CLI (npm), awf binary |
| **3. Container & Network** | 32.7s | 23.1% | 65.2s | Docker pulls, firewall setup, container start |
| **4. Start Coding Agent** | 32.7s | **23.1%** | 97.9s | ⚡ Agent initialization, MCP connections |
| **5. Agent Execution** | 85.4s | **60.2%** | 146.2s | 🤖 Actual AI coding work |

### Detailed Agent Job Timeline

```
Stage 1: Infrastructure Setup (23.7s)
├─ Job Start (13:15:41)
│  └─ GitHub Actions runner initialization
├─ Setup Scripts (13:15:45) [+3.5s]
│  └─ Copy 334 JavaScript files to /opt/gh-aw/actions
├─ Checkout repository (13:15:46) [+1.1s]
├─ Configure Git credentials (13:15:47) [+0.2s]
├─ Validate secrets (13:15:47) [+0.2s]
└─ Setup Node.js (13:15:49) [+2.6s]

Stage 2: Dependencies Installation (8.8s)
├─ Install awf binary v0.11.2 (13:15:50) [+0.8s]
└─ Install Claude Code CLI @2.1.22 (13:15:53) [+3.1s]

Stage 3: Container & Network Setup (11.4s)
├─ Download Docker images (13:15:53)
│  ├─ github-mcp-server:v0.30.2
│  ├─ gh-aw-mcpg:v0.0.84
│  └─ node:lts-alpine
├─ Write safe outputs config (13:15:58) [+4.8s]
├─ Generate MCP server config (13:15:58) [+0.3s]
├─ Start safe outputs MCP server (13:15:59) [+0.1s]
└─ Start MCP gateway (13:16:04) [+5.2s]

Stage 4: Start Coding Agent (32.7s) ⚡ KEY OPTIMIZATION TARGET
├─ Execute Claude Code CLI (13:16:05)
│  ├─ Initialize AWF firewall (13:16:06) [+0.9s]
│  ├─ Setup host-level iptables rules (13:16:06) [+0.2s]
│  ├─ Pull Docker images for agent container (13:16:06)
│  │  ├─ Pulling squid-proxy layers [8 layers]
│  │  └─ Pulling agent layers [17 layers]
│  └─ Containers started (13:16:37) [+32.0s]
└─ MCP initialization complete

Stage 5: Agent Execution (85.4s) 🤖 ACTUAL AI WORK
├─ Claude Code processing prompt (13:16:38)
├─ Making code changes
├─ Running tools (bash, file operations, MCP calls)
└─ Stop MCP gateway (13:18:03)
```

---

## 🎯 Top 5 Optimization Opportunities

### 1. **Pre-cache Docker Images** 
**Impact:** Save 15-20s (10-14% of agent job)  
**Current:** Stage 3 + Stage 4 spend 32.7s pulling containers on every run

**Problem:**
- `ghcr.io/github/github-mcp-server:v0.30.2` (~30MB)
- `ghcr.io/githubnext/gh-aw-mcpg:v0.0.84` (~268MB)
- `node:lts-alpine` (~40MB)
- Squid-proxy and agent container images
- **Total:** Multiple container layers pulled sequentially

**Solutions:**
- Add `docker pull` commands to workflow startup (runs once, cached)
- Use GitHub Actions runner image pre-installation
- Leverage Docker layer caching with BuildKit
- Consider using smaller base images (alpine variants)

**Implementation:**
```yaml
- name: Pre-cache Docker images
  run: |
    docker pull ghcr.io/github/github-mcp-server:v0.30.2 &
    docker pull ghcr.io/githubnext/gh-aw-mcpg:v0.0.84 &
    docker pull node:lts-alpine &
    wait
```

---

### 2. **Parallelize Setup Operations**
**Impact:** Save 8-12s (5-8% of agent job)  
**Current:** Sequential execution of independent tasks

**Parallelization Opportunities:**
- Docker image downloads (Stage 3) can run during dependency installation (Stage 2)
- MCP gateway start can overlap with Claude CLI installation
- Safe outputs config generation can run in parallel with Docker pulls

**Implementation:**
```bash
# Run independent operations in parallel
npm install -g @anthropic-ai/claude-code@2.1.22 &
docker pull ghcr.io/github/github-mcp-server:v0.30.2 &
docker pull ghcr.io/githubnext/gh-aw-mcpg:v0.0.84 &
wait  # Wait for all background jobs
```

---

### 3. **Optimize Agent Startup (Stage 4)**
**Impact:** Save 5-8s (3-6% of agent job)  
**Current:** 32.7s from "Execute Agent" to "Containers Started"

**Bottlenecks:**
- AWF firewall initialization: 0.9s
- iptables rules setup: 0.2s
- Container layer pulling: ~30s (overlaps with #1)
- Network bridge creation: ~0.5s

**Solutions:**
- Pre-create AWF network bridges (persist across runs)
- Cache iptables configurations
- Use faster container runtime (containerd vs Docker)
- Reduce MCP connection timeouts (current: 120s, 60s for tools)

**Configuration Change:**
```yaml
env:
  MCP_TIMEOUT: 30000        # Reduce from 120s to 30s
  MCP_TOOL_TIMEOUT: 20000   # Reduce from 60s to 20s
```

---

### 4. **Streamline JavaScript File Copying**
**Impact:** Save 2-3s (1-2% of agent job)  
**Current:** 1.1s to copy 334 JavaScript files in Stage 1

**Problem:**
- Setup Scripts copies all 334 .cjs files individually
- No compression or bundling
- Redundant test files (.test.cjs) copied to production

**Solutions:**
- Bundle JavaScript files into single archive
- Use tarball extraction instead of individual cp commands
- Exclude test files from production deployment
- Pre-install common files in runner image

---

### 5. **Reduce Detection Job Time**
**Impact:** Save 8-10s (3-4% of workflow)  
**Current:** 21.7s for threat detection (9.6% of total workflow)

**Analysis Needed:**
- What does the detection job analyze? (logs, artifacts, code)
- Can it run in parallel with agent job?
- Are all detection rules necessary?

**Potential Solutions:**
- Run detection concurrently with agent (non-blocking)
- Cache detection patterns/rules
- Use incremental analysis (only changed files)
- Reduce timeout thresholds

---

## Impact Summary

| Optimization | Time Saved | % Reduction | Effort | Priority |
|--------------|-----------|-------------|--------|----------|
| Docker image pre-caching | 15-20s | 10-14% | Medium | 🔴 High |
| Parallelize setup | 8-12s | 5-8% | Low | 🔴 High |
| Optimize agent startup | 5-8s | 3-6% | Medium | 🟡 Medium |
| JS file bundling | 2-3s | 1-2% | Low | 🟢 Low |
| Detection optimization | 8-10s | 3-4% | High | 🟡 Medium |
| **TOTAL POTENTIAL** | **38-53s** | **22-30%** | | |

**Expected New Total:** 173-188 seconds (2.9-3.1 minutes)  
**vs Current:** 225.9 seconds (3.8 minutes)

---

## Quick Wins (Implement First)

### Week 1: Immediate Improvements
1. ✅ **Pre-cache Docker images** (1-2 hour implementation)
2. ✅ **Parallelize NPM + Docker downloads** (30 min implementation)
3. ✅ **Reduce MCP timeouts** (5 min config change)

**Expected savings:** 20-25 seconds

### Week 2: Medium Effort
4. ⚙️ **Bundle JavaScript files** (4-6 hours)
5. ⚙️ **Optimize AWF network setup** (2-4 hours)

**Expected savings:** 5-8 seconds

### Future Optimization
6. 🔬 **Investigate detection parallelization** (research + design)
7. 🔬 **Custom runner image** with pre-installed tools

---

## Job Dependencies & Parallelization

Current: **All jobs run sequentially** (226s total)

```
pre_activation (5s) → activation (6s) → agent (146s) → detection (22s) → safe_outputs (7s) → conclusion (7s)
```

**Potential Parallel Execution:**

```
pre_activation (5s) → activation (6s) → agent (146s) ──┬→ safe_outputs (7s) → conclusion (7s)
                                                        └→ detection (22s) ──┘
```

**New Critical Path:** 171s (save 22s from detection not blocking safe_outputs)

---

## Time to Start Coding Agent

**Current:** 65.2 seconds from job start to containers ready

| Phase | Duration | Cumulative |
|-------|----------|------------|
| Infrastructure Setup | 23.7s | 23.7s |
| Dependencies Install | 8.8s | 32.5s |
| Container & Network Setup | 32.7s | 65.2s |
| **AGENT STARTS CODING** | | **65.2s** |

**With Optimizations:** ~40-45 seconds (37% improvement)

---

## Monitoring & Metrics

### Key Performance Indicators (KPIs)

1. **Time to Code** (TC): Job start → Agent starts coding
   - Current: 65.2s
   - Target: <45s

2. **Agent Startup Time** (AST): Execute command → Containers ready
   - Current: 32.7s
   - Target: <20s

3. **Total Workflow Time** (TWT): First job start → Last job end
   - Current: 225.9s
   - Target: <180s

4. **Agent Efficiency** (AE): Agent execution / Total agent job time
   - Current: 85.4s / 146.2s = 58.4%
   - Target: >70%

### Recommended Tracking

Add timing annotations to workflow:
```yaml
- name: Checkpoint - Containers Ready
  run: echo "::notice::Containers ready at $(date +%s)"
  
- name: Checkpoint - Agent Start
  run: echo "::notice::Agent execution started at $(date +%s)"
```

---

## Conclusion

The workflow's performance bottleneck is the **agent job (146.2s, 64.7%)**, specifically:
- **Container startup (32.7s, 23%)** - Primary optimization target
- **Agent execution (85.4s, 60%)** - Actual AI work (unavoidable)

Pre-activation (5.0s) and activation (6.2s) jobs are **already optimized** and represent minimal overhead (4.9% combined).

**Recommended Action Plan:**
1. Implement Docker image caching immediately (biggest impact)
2. Parallelize setup operations (low effort, medium impact)
3. Reduce MCP timeouts (quick win)
4. Investigate detection job parallelization
5. Monitor KPIs and iterate

**Expected Outcome:** Reduce total workflow time by 22-30% (from 3.8min to 2.9-3.1min) and improve "time to start coding" by 37% (from 65s to 40-45s).
