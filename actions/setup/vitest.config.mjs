import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    globals: true,
    include: ["js/**/*.test.{js,cjs}"],
    testTimeout: 10000,
    hookTimeout: 10000,
    coverage: {
      provider: "v8",
      reporter: ["text", "html"],
      include: ["js/**/*.cjs"],
      exclude: ["js/**/*.test.{js,cjs}"],
    },
  },
});
