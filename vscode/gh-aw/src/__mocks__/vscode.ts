import { vi } from 'vitest';

// Mock VSCode API
export const mockVSCode = {
  Position: class {
    line: number;
    character: number;
    constructor(line: number, character: number) {
      this.line = line;
      this.character = character;
    }
  },
  Range: class {
    start: any;
    end: any;
    constructor(start: any, end: any) {
      this.start = start;
      this.end = end;
    }
  },
  Uri: {
    file: vi.fn((path: string) => ({ fsPath: path, path })),
    joinPath: vi.fn((base: any, ...paths: string[]) => ({ 
      fsPath: base.fsPath + '/' + paths.join('/'),
      path: base.path + '/' + paths.join('/')
    }))
  },
  workspace: {
    workspaceFolders: [],
    textDocuments: [],
    onDidOpenTextDocument: vi.fn((callback) => {
      // Return a disposable
      return { dispose: vi.fn() };
    }),
    openTextDocument: vi.fn(),
    fs: {
      writeFile: vi.fn(),
      delete: vi.fn()
    }
  },
  window: {
    showInformationMessage: vi.fn(),
    showTextDocument: vi.fn()
  },
  languages: {
    setTextDocumentLanguage: vi.fn(),
    registerHoverProvider: vi.fn(() => ({ dispose: vi.fn() })),
    registerCompletionItemProvider: vi.fn(() => ({ dispose: vi.fn() })),
    getLanguages: vi.fn().mockResolvedValue(['agentic-workflow']),
    getDiagnostics: vi.fn().mockReturnValue([])
  },
  extensions: {
    getExtension: vi.fn()
  },
  commands: {
    executeCommand: vi.fn()
  },
  DiagnosticSeverity: {
    Error: 0,
    Warning: 1,
    Information: 2,
    Hint: 3
  },
  CompletionItemKind: {
    Value: 12,
    Property: 10
  },
  MarkdownString: class {
    value: string;
    constructor(value: string) {
      this.value = value;
    }
  },
  CompletionItem: class {
    label: string;
    kind: any;
    constructor(label: string, kind?: any) {
      this.label = label;
      this.kind = kind;
    }
  },
  Hover: class {
    contents: any;
    constructor(contents: any) {
      this.contents = contents;
    }
  }
};

// Create a mock TextDocument
export const createMockDocument = (content: string, fileName: string = '/test/file.md', languageId: string = 'markdown') => ({
  getText: vi.fn().mockReturnValue(content),
  fileName,
  languageId,
  lineCount: content.split('\n').length,
  lineAt: vi.fn((line: number) => ({
    text: content.split('\n')[line] || '',
    lineNumber: line
  })),
  uri: { fsPath: fileName, path: fileName }
});

export default mockVSCode;