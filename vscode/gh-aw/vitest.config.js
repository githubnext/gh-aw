"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const config_1 = require("vitest/config");
const path_1 = require("path");
exports.default = (0, config_1.defineConfig)({
    test: {
        environment: 'node',
        globals: true,
        include: ['src/**/*.test.ts'],
        exclude: ['src/test/suite/**/*'], // Exclude integration tests
        testTimeout: 10000,
        hookTimeout: 10000,
        coverage: {
            provider: 'v8',
            reporter: ['text', 'html'],
            include: ['src/**/*.ts'],
            exclude: ['src/**/*.test.ts', 'src/test/**/*']
        },
        alias: {
            '@': (0, path_1.resolve)(__dirname, './src')
        }
    },
    resolve: {
        alias: {
            '@': (0, path_1.resolve)(__dirname, './src')
        }
    }
});
//# sourceMappingURL=vitest.config.js.map