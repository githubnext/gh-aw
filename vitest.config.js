import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'node',
    globals: true,
    include: ['pkg/workflow/js/**/*.test.{js,cjs}'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      include: ['pkg/workflow/js/**/*.cjs'],
      exclude: ['pkg/workflow/js/**/*.test.{js,cjs}']
    }
  }
});