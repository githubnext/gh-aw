# Absurd - Postgres-based Durable Execution

## Overview

[Absurd](https://github.com/earendil-works/absurd) is a durable execution workflow system built entirely on Postgres. It provides long-lived, reliable function execution that can survive crashes, restarts, and network failures without losing state or duplicating work.

**Warning**: This is an early experiment and should not be used in production.

## Key Concepts

- **Durable Execution**: Tasks are decomposed into smaller pieces (step functions) with every step and decision recorded
- **Pull-based System**: Workers pull tasks from Postgres as they have capacity
- **Postgres-only**: No additional services required beyond Postgres
- **Checkpointing**: Step results are stored in the database to avoid repeating work on retries

## How It Works

- Tasks dispatch onto queues where workers pick them up
- Tasks are subdivided into steps executed in sequence
- Tasks can suspend, fail, or sleep for events
- Failed tasks retry at the task level with steps acting as checkpoints
- Events are cached for race-free suspension

## Potential Use Cases for gh-aw

Absurd could be relevant for gh-aw in scenarios requiring:
- Long-running agent tasks that span minutes, hours, or days
- Reliable multi-step workflows with automatic retry logic
- Stateful workflow execution that survives process crashes
- Event-driven workflows that wait for external triggers

## Resources

- **Repository**: https://github.com/earendil-works/absurd
- **Announcement Post**: https://lucumr.pocoo.org/2025/11/3/absurd-workflows/
- **Installation**: Single `.sql` file ([absurd.sql](https://github.com/earendil-works/absurd/blob/main/sql/absurd.sql))
- **Tools**:
  - `absurdctl`: CLI for queue and task management
  - `habitat`: Web UI for monitoring tasks
- **SDKs**: TypeScript/JavaScript, Python (unpublished)

## Comparison with Other Systems

Absurd is designed to be simpler than alternatives:
- [Temporal](https://temporal.io/): More comprehensive durable execution platform
- [Inngest](https://www.inngest.com/): Event-driven workflow system
- [DBOS](https://docs.dbos.dev/): Another Postgres-based durable workflows system
- [Cadence](https://github.com/cadence-workflow/cadence): Original Uber durable execution system
- [pgmq](https://github.com/pgmq/pgmq): Lightweight Postgres message queue (influenced Absurd)

## Implementation Details

See the [repository documentation](https://github.com/earendil-works/absurd) for:
- Installation and setup
- TypeScript SDK usage and examples
- Code change handling strategies
- Retry and idempotency patterns

---

**Status**: 📋 Exploratory - documenting potential integration opportunities

**Last Updated**: 2025-12-09
