---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
services:
  postgres:
    image: postgres:15
    ports:
      - 5432:5432
  redis:
    image: redis:7
    ports:
      - 6379:6379
---

# Test Service Ports

This workflow tests that the compiler automatically generates `--allow-host-service-ports`
from `services:` port mappings.

Expected: the compiled lock file includes `--allow-host-service-ports` with expressions for
both PostgreSQL (port 5432) and Redis (port 6379).
