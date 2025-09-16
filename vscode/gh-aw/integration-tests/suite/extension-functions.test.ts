import * as assert from 'assert';
import * as vscode from 'vscode';

// Import the extension module to test individual functions
import { isAgenticWorkflowFile, isInFrontmatter } from '../../extension';

suite('Extension Functions Test Suite', () => {
    
    test('isAgenticWorkflowFile should detect workflow files correctly', async () => {
        // Test with a valid agentic workflow file path
        const validWorkflowContent = `---
on: push
engine: claude
---
# Test Workflow`;

        const workflowDocument = await vscode.workspace.openTextDocument({
            content: validWorkflowContent,
            language: 'markdown'
        });

        // Mock the fileName property
        Object.defineProperty(workflowDocument, 'fileName', {
            value: '/test/.github/workflows/test.md',
            writable: false
        });

        const isWorkflowFile = isAgenticWorkflowFile(workflowDocument);
        assert.ok(isWorkflowFile, 'Should detect .md file in .github/workflows with frontmatter as agentic workflow');

        // Test with non-workflow file
        const nonWorkflowDocument = await vscode.workspace.openTextDocument({
            content: 'Regular markdown content without frontmatter',
            language: 'markdown'
        });

        Object.defineProperty(nonWorkflowDocument, 'fileName', {
            value: '/test/regular.md',
            writable: false
        });

        const isNotWorkflowFile = isAgenticWorkflowFile(nonWorkflowDocument);
        assert.ok(!isNotWorkflowFile, 'Should not detect regular markdown file as agentic workflow');

        // Clean up
        await vscode.commands.executeCommand('workbench.action.closeAllEditors');
    });

    test('isInFrontmatter should detect frontmatter positions correctly', async () => {
        const workflowContent = `---
on: push
engine: claude
---
# Workflow Content

This is the markdown content section.`;

        const document = await vscode.workspace.openTextDocument({
            content: workflowContent,
            language: 'agentic-workflow'
        });

        try {
            // Position in frontmatter (line 1: "on: push")
            const frontmatterPosition = new vscode.Position(1, 0);
            const isInFM = isInFrontmatter(document, frontmatterPosition);
            assert.ok(isInFM, 'Should detect position in frontmatter');

            // Position in markdown content (line 5: "# Workflow Content")  
            const markdownPosition = new vscode.Position(4, 0);
            const isNotInFM = isInFrontmatter(document, markdownPosition);
            assert.ok(!isNotInFM, 'Should not detect position in markdown content as frontmatter');
        } finally {
            // Clean up
            await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
        }
    });

    test('Extension should handle files without frontmatter', async () => {
        const contentWithoutFrontmatter = `# Regular Markdown

This is just regular markdown content without any frontmatter.`;

        const document = await vscode.workspace.openTextDocument({
            content: contentWithoutFrontmatter,
            language: 'markdown'
        });

        Object.defineProperty(document, 'fileName', {
            value: '/test/.github/workflows/no-frontmatter.md',
            writable: false
        });

        try {
            const isWorkflowFile = isAgenticWorkflowFile(document);
            assert.ok(!isWorkflowFile, 'Should not detect markdown file without frontmatter as agentic workflow');
        } finally {
            // Clean up
            await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
        }
    });

    test('Extension should handle empty files', async () => {
        const emptyDocument = await vscode.workspace.openTextDocument({
            content: '',
            language: 'markdown'
        });

        Object.defineProperty(emptyDocument, 'fileName', {
            value: '/test/.github/workflows/empty.md',
            writable: false
        });

        try {
            const isWorkflowFile = isAgenticWorkflowFile(emptyDocument);
            assert.ok(!isWorkflowFile, 'Should not detect empty file as agentic workflow');
        } finally {
            // Clean up
            await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
        }
    });
});