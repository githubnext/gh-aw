"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const path = require("path");
const test_electron_1 = require("@vscode/test-electron");
async function main() {
    try {
        // The folder containing the Extension Manifest package.json
        const extensionDevelopmentPath = path.resolve(__dirname, '../');
        // The path to test runner
        const extensionTestsPath = path.resolve(__dirname, './out-integration/suite/index');
        // Download VS Code, unzip it and run the integration test
        await (0, test_electron_1.runTests)({
            extensionDevelopmentPath,
            extensionTestsPath,
            launchArgs: ['--disable-extensions']
        });
    }
    catch (err) {
        console.error('Failed to run tests:', err);
        process.exit(1);
    }
}
main();
//# sourceMappingURL=runTest.js.map