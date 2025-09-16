import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createMockDocument, mockVSCode } from './__mocks__/vscode';

// Mock the vscode module
vi.mock('vscode', () => mockVSCode);

// Import the functions we want to test
import { isAgenticWorkflowFile, isInFrontmatter } from './extension';

describe('Extension Functions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('isAgenticWorkflowFile', () => {
    it('should detect agentic workflow files correctly', () => {
      const validWorkflowContent = `---
on: push
engine: claude
---
# Test Workflow`;

      const document = createMockDocument(validWorkflowContent, '/test/.github/workflows/test.md');
      
      const isWorkflowFile = isAgenticWorkflowFile(document as any);
      expect(isWorkflowFile).toBe(true);
    });

    it('should not detect regular markdown files as agentic workflows', () => {
      const regularContent = 'Regular markdown content without frontmatter';
      const document = createMockDocument(regularContent, '/test/regular.md');
      
      const isWorkflowFile = isAgenticWorkflowFile(document as any);
      expect(isWorkflowFile).toBe(false);
    });

    it('should not detect files outside .github/workflows as agentic workflows', () => {
      const workflowContent = `---
on: push
engine: claude
---
# Test Workflow`;

      const document = createMockDocument(workflowContent, '/test/other/test.md');
      
      const isWorkflowFile = isAgenticWorkflowFile(document as any);
      expect(isWorkflowFile).toBe(false);
    });

    it('should not detect non-markdown files as agentic workflows', () => {
      const workflowContent = `---
on: push
engine: claude
---
# Test Workflow`;

      const document = createMockDocument(workflowContent, '/test/.github/workflows/test.txt');
      
      const isWorkflowFile = isAgenticWorkflowFile(document as any);
      expect(isWorkflowFile).toBe(false);
    });

    it('should not detect markdown files without frontmatter as agentic workflows', () => {
      const contentWithoutFrontmatter = '# Regular Markdown\n\nThis is just regular markdown content.';
      const document = createMockDocument(contentWithoutFrontmatter, '/test/.github/workflows/no-frontmatter.md');
      
      const isWorkflowFile = isAgenticWorkflowFile(document as any);
      expect(isWorkflowFile).toBe(false);
    });

    it('should handle empty files correctly', () => {
      const document = createMockDocument('', '/test/.github/workflows/empty.md');
      
      const isWorkflowFile = isAgenticWorkflowFile(document as any);
      expect(isWorkflowFile).toBe(false);
    });
  });

  describe('isInFrontmatter', () => {
    it('should detect positions in frontmatter correctly', () => {
      const workflowContent = `---
on: push
engine: claude
---
# Workflow Content

This is the markdown content section.`;

      const document = createMockDocument(workflowContent);
      
      // Position in frontmatter (line 1: "on: push")
      const frontmatterPosition = new mockVSCode.Position(1, 0);
      const isInFM = isInFrontmatter(document as any, frontmatterPosition);
      expect(isInFM).toBe(true);
    });

    it('should not detect positions in markdown content as frontmatter', () => {
      const workflowContent = `---
on: push
engine: claude
---
# Workflow Content

This is the markdown content section.`;

      const document = createMockDocument(workflowContent);
      
      // Position in markdown content (line 4: "# Workflow Content")  
      const markdownPosition = new mockVSCode.Position(4, 0);
      const isInFM = isInFrontmatter(document as any, markdownPosition);
      expect(isInFM).toBe(false);
    });

    it('should handle files without frontmatter', () => {
      const contentWithoutFrontmatter = `# Regular Markdown

This is just regular markdown content without any frontmatter.`;

      const document = createMockDocument(contentWithoutFrontmatter);
      
      const position = new mockVSCode.Position(0, 0);
      const isInFM = isInFrontmatter(document as any, position);
      expect(isInFM).toBe(false);
    });

    it('should handle positions beyond frontmatter end', () => {
      const workflowContent = `---
on: push
---
# Content
More content`;

      const document = createMockDocument(workflowContent);
      
      // Position after frontmatter end
      const position = new mockVSCode.Position(10, 0);
      const isInFM = isInFrontmatter(document as any, position);
      expect(isInFM).toBe(false);
    });
  });
});