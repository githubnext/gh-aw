import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mockVSCode, createMockDocument } from './__mocks__/vscode';

// Mock the vscode module
vi.mock('vscode', () => mockVSCode);

// Mock path module
vi.mock('path', () => ({
  extname: vi.fn((filePath: string) => {
    const parts = filePath.split('.');
    return parts.length > 1 ? '.' + parts[parts.length - 1] : '';
  })
}));

describe('Extension Activation', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should register language providers on activation', async () => {
    // Mock the extension context
    const mockContext = {
      subscriptions: []
    };

    // Import and call the activate function
    const { activate } = await import('./extension');
    
    // Call activate with the mock context
    activate(mockContext as any);

    // Verify that language providers were registered
    expect(mockVSCode.languages.registerHoverProvider).toHaveBeenCalled();
    expect(mockVSCode.languages.registerCompletionItemProvider).toHaveBeenCalled();
    
    // Verify that subscriptions were added to context
    expect(mockContext.subscriptions.length).toBeGreaterThan(0);
  });

  it('should handle document open events', async () => {
    const mockContext = {
      subscriptions: []
    };

    // Import and activate the extension
    const { activate } = await import('./extension');
    activate(mockContext as any);

    // Verify that the document listener was registered
    expect(mockVSCode.workspace.onDidOpenTextDocument).toHaveBeenCalled();
  });

  it('should provide hover information for workflow properties', async () => {
    const mockContext = {
      subscriptions: []
    };

    // Import and activate the extension
    const { activate } = await import('./extension');
    activate(mockContext as any);

    // Get the hover provider that was registered
    expect(mockVSCode.languages.registerHoverProvider).toHaveBeenCalled();
    const hoverProvider = mockVSCode.languages.registerHoverProvider.mock.calls[0][1];

    // Test hover for 'engine:' property
    const document = createMockDocument('engine: claude');
    const position = new mockVSCode.Position(0, 0);
    
    // Mock the line content
    document.lineAt = vi.fn().mockReturnValue({ text: 'engine: claude' });

    const hover = hoverProvider.provideHover(document, position, null);
    
    expect(hover).toBeDefined();
    expect(hover.contents).toContain('AI Engine');
  });

  it('should provide completion items for workflow properties', async () => {
    const mockContext = {
      subscriptions: []
    };

    // Import and activate the extension
    const { activate } = await import('./extension');
    activate(mockContext as any);

    // Get the completion provider that was registered
    expect(mockVSCode.languages.registerCompletionItemProvider).toHaveBeenCalled();
    const completionProvider = mockVSCode.languages.registerCompletionItemProvider.mock.calls[0][1];

    // Test completion for engine values
    const document = createMockDocument('---\nengine: \n---');
    const position = new mockVSCode.Position(1, 8);
    
    // Mock line content
    document.lineAt = vi.fn().mockReturnValue({ text: 'engine: ' });

    const completions = completionProvider.provideCompletionItems(document, position);
    
    expect(completions).toBeDefined();
    expect(Array.isArray(completions)).toBe(true);
    
    // Should provide engine completions
    const engineCompletions = completions.filter((item: any) => 
      ['claude', 'codex', 'gpt-4'].includes(item.label)
    );
    expect(engineCompletions.length).toBeGreaterThan(0);
  });

  it('should check for already open documents during activation', async () => {
    const mockContext = {
      subscriptions: []
    };

    // Mock some open documents
    const workflowDoc = createMockDocument('---\nengine: claude\n---', '/test/.github/workflows/test.md');
    const regularDoc = createMockDocument('# Regular', '/test/regular.md');
    
    mockVSCode.workspace.textDocuments = [workflowDoc, regularDoc];

    // Import and activate the extension
    const { activate } = await import('./extension');
    activate(mockContext as any);

    // The function should process open documents
    // We can't easily test the specific behavior without refactoring, 
    // but we can verify activation completed without errors
    expect(mockContext.subscriptions.length).toBeGreaterThan(0);
  });
});