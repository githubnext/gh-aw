---
"gh-aw": major
---
Renamed the agentic workflow engine's `supportsLLMGateway` flag to `llmGatewayPort`, made the gateway port mandatory and validated, removed the `SupportsLLMGateway` interface hooks, and consolidated API proxy/host-access workflow flags.

**⚠️ Breaking Change**: The `supportsLLMGateway` field has been renamed to `llmGatewayPort` and is now mandatory (previously optional). The `SupportsLLMGateway` interface hooks have been removed.

**Migration guide:**
- Replace `supportsLLMGateway: true` with `llmGatewayPort: <port>` in engine configuration, providing the explicit port number
- Example:
  ```yaml
  # Before
  supportsLLMGateway: true

  # After
  llmGatewayPort: 8080
  ```
- If you were using `SupportsLLMGateway` interface hooks in custom engine implementations, migrate to the new `llmGatewayPort`-based configuration
- The `llmGatewayPort` field is now required when enabling LLM gateway support
