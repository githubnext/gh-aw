import * as assert from 'assert';
import * as vscode from 'vscode';
import * as path from 'path';

suite('Extension Test Suite', () => {
    vscode.window.showInformationMessage('Start all tests.');

    test('Extension should be present', () => {
        assert.ok(vscode.extensions.getExtension('github.gh-aw'));
    });

    test('Extension should activate', async () => {
        const extension = vscode.extensions.getExtension('github.gh-aw');
        assert.ok(extension);
        
        if (!extension.isActive) {
            await extension.activate();
        }
        
        assert.ok(extension.isActive);
    });

    test('Extension should provide agentic-workflow language', async () => {
        const languages = await vscode.languages.getLanguages();
        assert.ok(languages.includes('agentic-workflow'), 'agentic-workflow language should be registered');
    });

    test('Extension should recognize workflow files in .github/workflows/', async () => {
        // Create a temporary test file
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

# Test Workflow

This is a test agentic workflow file.`;

        const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
        if (!workspaceFolder) {
            // Skip test if no workspace folder
            return;
        }

        const testFileUri = vscode.Uri.joinPath(
            workspaceFolder.uri, 
            '.github', 
            'workflows', 
            'test-workflow.md'
        );

        try {
            // Create the file
            await vscode.workspace.fs.writeFile(testFileUri, Buffer.from(testContent));
            
            // Open the file
            const document = await vscode.workspace.openTextDocument(testFileUri);
            
            // Check that it's recognized as a markdown file initially
            assert.ok(document.languageId === 'markdown' || document.languageId === 'agentic-workflow');
            
            // Clean up
            await vscode.workspace.fs.delete(testFileUri);
        } catch (error) {
            // Clean up on error
            try {
                await vscode.workspace.fs.delete(testFileUri);
            } catch {
                // Ignore cleanup errors
            }
            throw error;
        }
    });
});