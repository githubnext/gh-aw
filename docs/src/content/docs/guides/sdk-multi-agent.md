---
title: SDK Multi-Agent Orchestration
description: Guide to coordinating multiple specialized agents in SDK mode workflows.
sidebar:
  order: 728
---

SDK mode enables sophisticated multi-agent architectures where multiple specialized agents collaborate to complete complex tasks.

## Overview

**Multi-agent orchestration** allows you to:

- Run multiple specialized agents in parallel or sequence
- Share context and state between agents
- Implement hierarchical agent structures
- Coordinate agent specialization (research, code, review)
- Manage agent resource allocation
- Handle inter-agent communication

## Basic Multi-Agent Setup

### Sequential Agents

Agents execute one after another, passing context:

```yaml
---
engine:
  id: copilot
  mode: sdk
  agents:
    - name: researcher
      role: "Research and gather requirements"
      session:
        persistent: true
    
    - name: implementer
      role: "Implement the solution"
      session:
        restore: inherit  # Restore from researcher
    
    - name: reviewer
      role: "Review and validate"
      session:
        restore: inherit  # Restore from implementer
tools:
  github:
    allowed: [issue_read, add_issue_comment, create_pull_request]
---
# Multi-Stage Development

Stage 1: Researcher gathers requirements
Stage 2: Implementer creates solution
Stage 3: Reviewer validates and approves
```

### Parallel Agents

Agents execute simultaneously:

```yaml
---
engine:
  id: copilot
  mode: sdk
  agents:
    - name: security_checker
      role: "Check security vulnerabilities"
      parallel: true
    
    - name: performance_analyzer
      role: "Analyze performance"
      parallel: true
    
    - name: quality_auditor
      role: "Check code quality"
      parallel: true
  
  coordination:
    strategy: parallel
    merge: aggregate  # Combine all agent results
tools:
  github:
    allowed: [issue_read]
---
# Parallel Code Analysis

Run security, performance, and quality checks simultaneously.
```

### Hybrid (Sequential + Parallel)

Combination of sequential and parallel execution:

```yaml
---
engine:
  id: copilot
  mode: sdk
  agents:
    # Stage 1: Single agent
    - name: planner
      role: "Create implementation plan"
      session:
        persistent: true
    
    # Stage 2: Parallel agents
    - name: backend_dev
      role: "Implement backend"
      parallel: true
      session:
        restore: inherit
    
    - name: frontend_dev
      role: "Implement frontend"
      parallel: true
      session:
        restore: inherit
    
    # Stage 3: Single agent
    - name: integrator
      role: "Integrate and test"
      session:
        restore: merge  # Merge backend_dev + frontend_dev
---
# Hybrid Development Pipeline

1. Planner creates strategy
2. Backend and frontend developed in parallel
3. Integrator combines and tests
```

## Agent Configuration

### Agent Definition

```yaml
agents:
  - name: agent_name           # Unique identifier (required)
    role: "Agent purpose"      # Description of role (required)
    model: gpt-5               # Override default model
    parallel: false            # Run in parallel (default: false)
    session:                   # Session configuration
      persistent: true
      max-turns: 10
      restore: inherit         # inherit, merge, disabled
    budget:                    # Agent-specific budget
      max-tokens: 50000
    tools:                     # Agent-specific tools
      allowed: [tool1, tool2]
    priority: 1                # Execution priority (1-10)
    dependencies: []           # Required agents
```

### Agent Properties

**`name`** (string, required)  
Unique identifier for the agent.

**`role`** (string, required)  
Description of the agent's purpose and responsibilities.

**`model`** (string, optional)  
Override the default model for this agent (e.g., `gpt-5`, `claude-sonnet-4`).

**`parallel`** (boolean, default: `false`)  
Whether agent can execute in parallel with others.

**`session`** (object, optional)  
Agent-specific session configuration.

**`budget`** (object, optional)  
Agent-specific token budget limits.

**`tools`** (object, optional)  
Agent-specific tool allowlist (overrides global).

**`priority`** (integer, default: `5`)  
Execution priority (1=highest, 10=lowest). Used when coordinating parallel agents.

**`dependencies`** (array, optional)  
List of agent names that must complete before this agent starts.

## Session Management

### Session Restoration Strategies

**`inherit`** - Restore from previous agent
```yaml
agents:
  - name: agent1
    session:
      persistent: true
  
  - name: agent2
    session:
      restore: inherit  # Continue from agent1
```

**`merge`** - Combine multiple agent contexts
```yaml
agents:
  - name: backend
    parallel: true
  
  - name: frontend
    parallel: true
  
  - name: integrator
    session:
      restore: merge  # Merge backend + frontend
```

**`shared`** - Real-time shared state
```yaml
agents:
  - name: agent1
    parallel: true
    session:
      shared: true
  
  - name: agent2
    parallel: true
    session:
      shared: true  # Both share same state
```

**`isolated`** - Independent state
```yaml
agents:
  - name: agent1
    session:
      restore: isolated  # Start fresh
```

### Context Sharing

Share specific data between agents:

```yaml
---
engine:
  id: copilot
  mode: sdk
  agents:
    - name: analyzer
      session:
        persistent: true
        exports:  # Export data to next agent
          - requirements
          - constraints
    
    - name: implementer
      session:
        restore: inherit
        imports:  # Import from analyzer
          - requirements
          - constraints
---
```

## Coordination Strategies

### Sequential Coordination

Agents execute in order:

```yaml
engine:
  coordination:
    strategy: sequential
    on-failure: stop  # stop, continue, retry
```

**Flow:**
```
Agent1 → Agent2 → Agent3 → Complete
```

### Parallel Coordination

Agents execute simultaneously:

```yaml
engine:
  coordination:
    strategy: parallel
    max-concurrent: 3  # Limit concurrent agents
    merge: aggregate   # aggregate, first, vote
```

**Flow:**
```
        ┌─ Agent1 ─┐
Start ──┼─ Agent2 ─┼→ Merge → Complete
        └─ Agent3 ─┘
```

### Pipeline Coordination

Staged execution with handoffs:

```yaml
engine:
  coordination:
    strategy: pipeline
    stages:
      - name: research
        agents: [researcher]
      - name: development
        agents: [backend_dev, frontend_dev]
        parallel: true
      - name: review
        agents: [reviewer]
```

**Flow:**
```
Stage 1: researcher
         ↓
Stage 2: backend_dev + frontend_dev (parallel)
         ↓
Stage 3: reviewer
```

### Hierarchical Coordination

Manager agent coordinates workers:

```yaml
engine:
  coordination:
    strategy: hierarchical
    manager: orchestrator
    workers:
      - worker1
      - worker2
      - worker3
```

**Flow:**
```
    Orchestrator (Manager)
         ↓
    ┌────┴────┐
Worker1  Worker2  Worker3
    │      │      │
    └──────┴──────┘
         ↓
    Orchestrator (Consolidate)
```

## Agent Specialization Patterns

### Pattern 1: Domain Specialists

Each agent specializes in specific domain:

```yaml
---
engine:
  id: copilot
  mode: sdk
  agents:
    - name: security_expert
      role: "Security vulnerability analysis"
      model: gpt-5
      tools:
        allowed: [scan_dependencies, check_secrets]
    
    - name: performance_expert
      role: "Performance optimization"
      model: claude-sonnet-4
      tools:
        allowed: [profile_code, analyze_complexity]
    
    - name: ux_expert
      role: "User experience review"
      tools:
        allowed: [check_accessibility, validate_ui]
  
  coordination:
    strategy: parallel
    merge: aggregate
---
# Specialist Review

Each expert provides domain-specific feedback.
```

### Pattern 2: Pipeline Stages

Agents handle different workflow stages:

```yaml
---
engine:
  id: copilot
  mode: sdk
  agents:
    - name: requirements_analyst
      role: "Analyze and document requirements"
      session:
        persistent: true
        exports: [requirements_doc]
    
    - name: architect
      role: "Design system architecture"
      session:
        restore: inherit
        exports: [architecture_doc]
    
    - name: developer
      role: "Implement solution"
      session:
        restore: inherit
        exports: [implementation]
    
    - name: tester
      role: "Test and validate"
      session:
        restore: inherit
---
# Software Development Pipeline

Requirements → Architecture → Implementation → Testing
```

### Pattern 3: Checker Pattern

Multiple agents verify different aspects:

```yaml
---
engine:
  id: copilot
  mode: sdk
  agents:
    - name: style_checker
      role: "Check code style compliance"
      parallel: true
    
    - name: test_checker
      role: "Verify test coverage"
      parallel: true
    
    - name: doc_checker
      role: "Validate documentation"
      parallel: true
    
    - name: security_checker
      role: "Security audit"
      parallel: true
  
  coordination:
    strategy: parallel
    merge: all_pass  # All must pass
---
# Multi-Aspect Validation

All checkers must pass for approval.
```

### Pattern 4: Consensus Pattern

Agents vote on decisions:

```yaml
---
engine:
  id: copilot
  mode: sdk
  agents:
    - name: reviewer1
      role: "Code review perspective 1"
      parallel: true
    
    - name: reviewer2
      role: "Code review perspective 2"
      parallel: true
    
    - name: reviewer3
      role: "Code review perspective 3"
      parallel: true
  
  coordination:
    strategy: parallel
    merge: vote
    threshold: 2  # Require 2/3 approval
---
# Consensus-Based Review

Require majority approval from reviewers.
```

## Resource Management

### Budget Allocation

Distribute token budget across agents:

```yaml
engine:
  id: copilot
  mode: sdk
  budget:
    max-tokens: 100000
    allocation:
      researcher: 30000    # 30%
      implementer: 50000   # 50%
      reviewer: 20000      # 20%
  
  agents:
    - name: researcher
      budget:
        max-tokens: 30000
    
    - name: implementer
      budget:
        max-tokens: 50000
    
    - name: reviewer
      budget:
        max-tokens: 20000
```

### Priority-Based Scheduling

Control agent execution order:

```yaml
agents:
  - name: critical_agent
    priority: 1  # Highest priority
  
  - name: important_agent
    priority: 3
  
  - name: optional_agent
    priority: 7  # Lower priority
```

### Resource Limits

Limit concurrent resource usage:

```yaml
engine:
  coordination:
    max-concurrent: 3        # Max 3 agents at once
    max-memory-per-agent: 1024  # 1GB per agent
    max-duration: 3600       # 1 hour total
```

## Inter-Agent Communication

### Message Passing

Agents send messages to each other:

```yaml
---
engine:
  id: copilot
  mode: sdk
  agents:
    - name: analyzer
      tools:
        inline:
          - name: send_message
            parameters:
              type: object
              required: [to, message]
              properties:
                to:
                  type: string
                message:
                  type: string
            implementation: |
              global.messageQueue = global.messageQueue || {};
              global.messageQueue[to] = message;
              return { sent: true };
    
    - name: implementer
      tools:
        inline:
          - name: receive_message
            parameters:
              type: object
              required: [from]
              properties:
                from:
                  type: string
            implementation: |
              const message = global.messageQueue?.[from] || null;
              return { message };
---
# Message-Based Coordination

Agents communicate via message passing.
```

### Shared State

Agents access shared state object:

```yaml
---
engine:
  id: copilot
  mode: sdk
  shared-state:
    enabled: true
  
  agents:
    - name: agent1
      tools:
        inline:
          - name: update_shared_state
            implementation: |
              global.sharedState = global.sharedState || {};
              global.sharedState.agent1_result = result;
    
    - name: agent2
      tools:
        inline:
          - name: read_shared_state
            implementation: |
              return global.sharedState || {};
---
# Shared State Coordination

Agents coordinate via shared state.
```

### Event-Based Communication

Agents emit and listen for events:

```yaml
---
engine:
  id: copilot
  mode: sdk
  events:
    streaming: true
    handlers:
      - on: agent_event
        action: custom
        config:
          implementation: |
            // Route events between agents
            if (event.data.target_agent) {
              routeToAgent(event.data.target_agent, event.data.payload);
            }
---
```

## Best Practices

### 1. Clear Agent Responsibilities

```yaml
# ✅ Good: Specific, focused roles
agents:
  - name: syntax_checker
    role: "Check code syntax and formatting"
  
  - name: logic_reviewer
    role: "Review business logic correctness"

# ❌ Bad: Vague, overlapping roles
agents:
  - name: checker
    role: "Check things"
```

### 2. Appropriate Parallelization

```yaml
# ✅ Good: Independent tasks in parallel
agents:
  - name: unit_tests
    parallel: true
  
  - name: integration_tests
    parallel: true

# ❌ Bad: Dependent tasks in parallel
agents:
  - name: design
    parallel: true
  
  - name: implement
    parallel: true  # Needs design first!
```

### 3. Budget Distribution

```yaml
# ✅ Good: Budget matches complexity
agents:
  - name: quick_scan
    budget:
      max-tokens: 5000
  
  - name: deep_analysis
    budget:
      max-tokens: 50000

# ❌ Bad: Uniform budget
agents:
  - name: quick_scan
    budget:
      max-tokens: 25000  # Too much
  
  - name: deep_analysis
    budget:
      max-tokens: 25000  # Too little
```

### 4. Error Handling

```yaml
# ✅ Good: Graceful failure handling
engine:
  coordination:
    strategy: parallel
    on-failure: continue  # Don't stop other agents
  
  events:
    handlers:
      - on: agent_error
        action: log
        config:
          level: error

# ❌ Bad: No error handling
engine:
  coordination:
    strategy: parallel
```

### 5. Result Merging

```yaml
# ✅ Good: Appropriate merge strategy
engine:
  coordination:
    merge: aggregate  # Combine all results

# For voting scenario
engine:
  coordination:
    merge: vote
    threshold: 2  # Majority

# For first-success scenario
engine:
  coordination:
    merge: first
```

## Examples

### Example 1: Code Review Board

Multiple reviewers provide feedback:

```yaml
---
engine:
  id: copilot
  mode: sdk
  budget:
    max-tokens: 100000
  
  agents:
    - name: senior_reviewer
      role: "Senior engineer review for architecture"
      parallel: true
      priority: 1
    
    - name: security_reviewer
      role: "Security-focused review"
      parallel: true
      priority: 2
    
    - name: junior_reviewer
      role: "Code style and documentation review"
      parallel: true
      priority: 3
  
  coordination:
    strategy: parallel
    merge: aggregate
tools:
  github:
    allowed: [pull_request_read, add_pull_request_comment]
---
# PR Review Board

Multiple reviewers provide diverse feedback on pull request.
```

### Example 2: Research and Development

Sequential research → development pipeline:

```yaml
---
engine:
  id: copilot
  mode: sdk
  agents:
    - name: market_researcher
      role: "Research market and user needs"
      session:
        persistent: true
        max-turns: 8
        exports: [research_findings]
      budget:
        max-tokens: 30000
    
    - name: product_designer
      role: "Design product based on research"
      session:
        restore: inherit
        max-turns: 10
        exports: [design_spec]
      budget:
        max-tokens: 40000
    
    - name: developer
      role: "Implement product"
      session:
        restore: inherit
        max-turns: 15
      budget:
        max-tokens: 50000
    
    - name: qa_tester
      role: "Test and validate product"
      session:
        restore: inherit
        max-turns: 10
      budget:
        max-tokens: 30000
tools:
  github:
    allowed: [issue_read, create_pull_request]
---
# Product Development Pipeline

Research → Design → Development → Testing
```

### Example 3: Distributed Testing

Parallel test execution across different suites:

```yaml
---
engine:
  id: copilot
  mode: sdk
  agents:
    - name: unit_tester
      role: "Run unit tests"
      parallel: true
      tools:
        inline:
          - name: run_unit_tests
            implementation: |
              const { execSync } = require('child_process');
              const result = execSync('npm run test:unit');
              return { result: result.toString() };
    
    - name: integration_tester
      role: "Run integration tests"
      parallel: true
      tools:
        inline:
          - name: run_integration_tests
            implementation: |
              const { execSync } = require('child_process');
              const result = execSync('npm run test:integration');
              return { result: result.toString() };
    
    - name: e2e_tester
      role: "Run end-to-end tests"
      parallel: true
      tools:
        inline:
          - name: run_e2e_tests
            implementation: |
              const { execSync } = require('child_process');
              const result = execSync('npm run test:e2e');
              return { result: result.toString() };
    
    - name: test_aggregator
      role: "Aggregate and report all test results"
      session:
        restore: merge
      dependencies: [unit_tester, integration_tester, e2e_tester]
  
  coordination:
    strategy: hybrid
    merge: aggregate
---
# Parallel Test Execution

Run all test suites in parallel, aggregate results.
```

### Example 4: Consensus-Based Decision

Voting mechanism for important decisions:

```yaml
---
engine:
  id: copilot
  mode: sdk
  agents:
    - name: reviewer_a
      role: "Reviewer A perspective"
      parallel: true
      model: gpt-5
    
    - name: reviewer_b
      role: "Reviewer B perspective"
      parallel: true
      model: claude-sonnet-4
    
    - name: reviewer_c
      role: "Reviewer C perspective"
      parallel: true
      model: gpt-5
    
    - name: decision_maker
      role: "Make final decision based on majority vote"
      session:
        restore: merge
      dependencies: [reviewer_a, reviewer_b, reviewer_c]
      tools:
        inline:
          - name: tally_votes
            implementation: |
              // Count approval votes
              const votes = {
                approve: 0,
                reject: 0,
                neutral: 0
              };
              
              // Process each reviewer's vote
              [reviewer_a, reviewer_b, reviewer_c].forEach(review => {
                if (review.decision === 'approve') votes.approve++;
                else if (review.decision === 'reject') votes.reject++;
                else votes.neutral++;
              });
              
              return {
                votes,
                decision: votes.approve >= 2 ? 'approve' : 'reject'
              };
  
  coordination:
    strategy: hybrid
    merge: vote
    threshold: 2  # Need 2/3 approval
tools:
  github:
    allowed: [pull_request_read, add_pull_request_comment]
---
# Consensus-Based PR Approval

Require majority approval from 3 reviewers.
```

## Troubleshooting

### Issue: Agent Conflicts

**Symptoms:** Agents interfering with each other

**Solutions:**
```yaml
# 1. Use isolated sessions
agents:
  - name: agent1
    session:
      restore: isolated
  
  - name: agent2
    session:
      restore: isolated

# 2. Limit concurrent agents
engine:
  coordination:
    max-concurrent: 1  # Sequential execution
```

### Issue: Budget Overrun

**Symptoms:** Single agent consuming all budget

**Solutions:**
```yaml
# Allocate per-agent budgets
agents:
  - name: agent1
    budget:
      max-tokens: 30000
  
  - name: agent2
    budget:
      max-tokens: 30000
```

### Issue: Deadlock

**Symptoms:** Agents waiting for each other

**Solutions:**
```yaml
# Use explicit dependencies
agents:
  - name: agent1
  
  - name: agent2
    dependencies: [agent1]  # Clear dependency chain
```

## Related Documentation

- [SDK Engine Reference](/gh-aw/reference/engines-sdk/) - Complete SDK configuration
- [Session Management](/gh-aw/guides/sdk-sessions/) - Multi-turn conversations
- [Event Handling](/gh-aw/guides/sdk-events/) - Real-time event processing
- [Migration Guide](/gh-aw/guides/migrate-to-sdk/) - Migrating from CLI mode
