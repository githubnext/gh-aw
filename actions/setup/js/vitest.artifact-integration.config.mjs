import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    globals: true,
    include: ["artifact_client_live_api.test.cjs"],
    // Allow enough time for real network I/O against GitHub's artifact storage.
    testTimeout: 120000,
    hookTimeout: 10000,
  },
});
