**⚠️ Zero-Token Early CLI Exit**: The Copilot CLI exited before any model inference turn started, and no effective token usage was recorded.

This failure mode is distinct from timeout and effective-token budget exhaustion. It usually indicates an early startup/auth/config rejection in CLI initialization.

{run_line}
