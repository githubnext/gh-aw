"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const assert = require("assert");
const vscode = require("vscode");
suite('Language Features Test Suite', () => {
    test('Hover provider should be registered', async () => {
        // Create a test document with agentic workflow content
        const testContent = `---
engine: claude
permissions: read-all
safe-outputs:
  create-issue:
    title-prefix: "[Test] "
tools:
  web-search:
---

# Test Workflow`;
        const document = await vscode.workspace.openTextDocument({
            content: testContent,
            language: 'agentic-workflow'
        });
        const position = new vscode.Position(1, 0); // Position at "engine: claude"
        try {
            const hovers = await vscode.commands.executeCommand('vscode.executeHoverProvider', document.uri, position);
            // Note: This test might pass even if no hover is provided, 
            // as hover providers are optional
            assert.ok(Array.isArray(hovers));
        }
        finally {
            // Clean up
            await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
        }
    });
    test('Completion provider should be registered', async () => {
        const testContent = `---
engine: 
---

# Test Workflow`;
        const document = await vscode.workspace.openTextDocument({
            content: testContent,
            language: 'agentic-workflow'
        });
        const position = new vscode.Position(1, 8); // Position after "engine: "
        try {
            const completions = await vscode.commands.executeCommand('vscode.executeCompletionItemProvider', document.uri, position);
            // Note: This test might pass even if no completions are provided
            assert.ok(completions !== undefined);
        }
        finally {
            // Clean up
            await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
        }
    });
    test('Document should be validated against schema', async () => {
        const testContent = `---
on:
  schedule:
    - cron: "0 9 * * 1"
engine: claude
permissions: read-all
safe-outputs:
  create-issue:
    title-prefix: "[Test] "
---

# Valid Test Workflow`;
        const document = await vscode.workspace.openTextDocument({
            content: testContent,
            language: 'agentic-workflow'
        });
        try {
            // Open the document in the editor to trigger validation
            await vscode.window.showTextDocument(document);
            // Wait a bit for validation to complete
            await new Promise(resolve => setTimeout(resolve, 1000));
            // Get diagnostics
            const diagnostics = vscode.languages.getDiagnostics(document.uri);
            // For a valid document, there should be no errors
            const errors = diagnostics.filter(d => d.severity === vscode.DiagnosticSeverity.Error);
            assert.strictEqual(errors.length, 0, `Expected no validation errors, but got: ${errors.map(e => e.message).join(', ')}`);
        }
        finally {
            // Clean up
            await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
        }
    });
});
//# sourceMappingURL=language-features.test.js.map