declare module "@github/copilot-sdk/extension" {
  export interface CanvasOpenResult {
    status?: string;
    title?: string;
    url?: string;
  }

  export interface CanvasContext<TInput = unknown> {
    canvasId: string;
    extensionId: string;
    host?: unknown;
    input?: TInput;
    instanceId: string;
    sessionId: string;
  }

  export interface CanvasAction<TInput = unknown, TResult = unknown> {
    description: string;
    handler: (ctx: CanvasContext<TInput> & { actionName: string }) => Promise<TResult> | TResult;
    inputSchema?: Record<string, unknown>;
    name: string;
  }

  export interface CanvasDefinition<TInput = unknown> {
    actions?: CanvasAction[];
    description: string;
    displayName: string;
    id: string;
    inputSchema?: Record<string, unknown>;
    onClose?: (ctx: CanvasContext<TInput>) => Promise<void> | void;
    open: (ctx: CanvasContext<TInput>) => Promise<CanvasOpenResult> | CanvasOpenResult;
  }

  export interface JoinedSession {
    on: (eventName: string, handler: (event: unknown) => void) => () => void;
    rpc?: unknown;
    send: (payload: { attachments?: Array<{ path: string; type: string }>; prompt: string }) => Promise<void>;
    sendAndWait: (payload: { attachments?: Array<{ path: string; type: string }>; prompt: string }) => Promise<unknown>;
    sessionId: string;
    workspacePath?: string;
  }

  export function createCanvas<TInput = unknown>(definition: CanvasDefinition<TInput>): CanvasDefinition<TInput>;

  export function joinSession(options: { canvases: CanvasDefinition[]; onPermissionRequest?: (request: unknown) => Promise<unknown>; systemMessage?: { content: string; mode: string }; tools?: unknown[] }): Promise<JoinedSession>;
}
