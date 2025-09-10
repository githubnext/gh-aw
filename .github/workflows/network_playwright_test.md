---
engine: claude

network:
  allowed:
    - defaults
    - playwright

tools:
  playwright:
    docker_image_version: "v1.41.0"
---

# Network Playwright Test  

Test Playwright with network permissions.