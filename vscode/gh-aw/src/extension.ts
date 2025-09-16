import * as vscode from 'vscode';
import * as path from 'path';

export function activate(context: vscode.ExtensionContext) {
    console.log('GitHub Agentic Workflows extension is now active');

    // Register language configuration
    const documentSelector: vscode.DocumentSelector = [
        {
            language: 'markdown',
            pattern: '**/.github/workflows/*.md'
        }
    ];

    // Set language ID for agentic workflow files
    vscode.workspace.onDidOpenTextDocument((document) => {
        if (isAgenticWorkflowFile(document)) {
            vscode.languages.setTextDocumentLanguage(document, 'agentic-workflow');
        }
    });

    // Check already open documents
    vscode.workspace.textDocuments.forEach((document) => {
        if (isAgenticWorkflowFile(document)) {
            vscode.languages.setTextDocumentLanguage(document, 'agentic-workflow');
        }
    });

    // Provide hover information for agentic workflow properties
    const hoverProvider = vscode.languages.registerHoverProvider(documentSelector, {
        provideHover(document, position, token) {
            const line = document.lineAt(position);
            const text = line.text;

            // Provide hover information for common agentic workflow properties
            if (text.includes('engine:')) {
                return new vscode.Hover('**AI Engine**: Specifies which AI engine to use (claude, codex, gpt-4)');
            }
            if (text.includes('safe-outputs:')) {
                return new vscode.Hover('**Safe Outputs**: Configuration for allowed GitHub API actions the AI can perform');
            }
            if (text.includes('tools:')) {
                return new vscode.Hover('**Tools**: Available tools and APIs the AI agent can use');
            }
            if (text.includes('permissions:')) {
                return new vscode.Hover('**Permissions**: GitHub token permissions required for the workflow');
            }
            if (text.includes('@include')) {
                return new vscode.Hover('**Include Directive**: Include content from another markdown file');
            }

            return undefined;
        }
    });

    context.subscriptions.push(hoverProvider);

    // Provide completion items for agentic workflow frontmatter
    const completionProvider = vscode.languages.registerCompletionItemProvider(
        documentSelector,
        {
            provideCompletionItems(document, position) {
                const line = document.lineAt(position);
                const linePrefix = line.text.substr(0, position.character);

                // Only provide completions in frontmatter section
                if (!isInFrontmatter(document, position)) {
                    return undefined;
                }

                const completions: vscode.CompletionItem[] = [];

                // Engine completions
                if (linePrefix.includes('engine:') || linePrefix.includes('id:')) {
                    ['claude', 'codex', 'gpt-4'].forEach(engine => {
                        const item = new vscode.CompletionItem(engine, vscode.CompletionItemKind.Value);
                        item.detail = `AI Engine: ${engine}`;
                        completions.push(item);
                    });
                }

                // Permission completions
                if (linePrefix.includes('permissions:')) {
                    ['read-all', 'write-all'].forEach(perm => {
                        const item = new vscode.CompletionItem(perm, vscode.CompletionItemKind.Value);
                        item.detail = `Permission level: ${perm}`;
                        completions.push(item);
                    });
                }

                // Root level property completions
                if (linePrefix.match(/^\s*$/)) {
                    const rootProperties = [
                        'on', 'engine', 'permissions', 'safe-outputs', 'tools', 
                        'network', 'cache', 'timeout_minutes', 'if'
                    ];
                    
                    rootProperties.forEach(prop => {
                        const item = new vscode.CompletionItem(prop, vscode.CompletionItemKind.Property);
                        item.insertText = `${prop}: `;
                        item.detail = 'Agentic workflow property';
                        completions.push(item);
                    });
                }

                return completions;
            }
        },
        ':', ' '
    );

    context.subscriptions.push(completionProvider);
}

function isAgenticWorkflowFile(document: vscode.TextDocument): boolean {
    // Check if it's a markdown file in .github/workflows directory
    const filePath = document.fileName;
    const isMarkdown = path.extname(filePath) === '.md';
    const isInWorkflowsDir = filePath.includes('.github/workflows');
    
    if (!isMarkdown || !isInWorkflowsDir) {
        return false;
    }

    // Check if it has YAML frontmatter
    const text = document.getText();
    return text.startsWith('---');
}

function isInFrontmatter(document: vscode.TextDocument, position: vscode.Position): boolean {
    const text = document.getText();
    const lines = text.split('\n');
    
    let inFrontmatter = false;
    let frontmatterEnd = -1;
    
    for (let i = 0; i < lines.length; i++) {
        if (i === 0 && lines[i].startsWith('---')) {
            inFrontmatter = true;
            continue;
        }
        if (inFrontmatter && lines[i].startsWith('---')) {
            frontmatterEnd = i;
            break;
        }
    }
    
    return position.line < frontmatterEnd;
}

export function deactivate() {
    console.log('GitHub Agentic Workflows extension is now deactivated');
}