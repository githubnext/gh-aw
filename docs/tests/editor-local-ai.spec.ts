import { test, expect } from "@playwright/test";

test("loads local AI on demand and retries compiler errors", async ({ page }) => {
  await page.addInitScript(() => {
    const workerUrls: string[] = [];
    const aiRequests: Array<{ diagnostic: string }> = [];

    class MockWorker {
      url: string;
      onmessage: ((event: { data: unknown }) => void) | null = null;
      onerror: ((event: { message?: string }) => void) | null = null;
      listeners = new Map<string, Array<(event: { data?: unknown; message?: string }) => void>>();

      constructor(url: string | URL) {
        this.url = String(url);
        workerUrls.push(this.url);
        if (this.url.includes("compiler-worker.js")) {
          queueMicrotask(() => {
            this.emit("message", { type: "progress", stage: "Downloading compiler", progress: 50 });
            this.emit("message", { type: "ready" });
          });
        }
      }

      addEventListener(type: string, listener: (event: { data?: unknown; message?: string }) => void) {
        const listeners = this.listeners.get(type) ?? [];
        listeners.push(listener);
        this.listeners.set(type, listeners);
      }

      postMessage(message: { type: string; id: number; markdown?: string; diagnostic?: string }) {
        if (message.type === "compile") {
          queueMicrotask(() => {
            if (message.markdown?.includes("BROKEN")) {
              this.emit("message", { type: "error", id: message.id, error: "invalid workflow" });
            } else {
              this.emit("message", { type: "result", id: message.id, yaml: "name: valid", warnings: [] });
            }
          });
          return;
        }

        if (message.type === "generate") {
          aiRequests.push({ diagnostic: message.diagnostic ?? "" });
          const text = aiRequests.length === 1 ? "<workflow>BROKEN</workflow>" : "<workflow>---\nname: locally-fixed\non: workflow_dispatch\n---\n\n# Fixed</workflow>";
          queueMicrotask(() => {
            this.emit("message", { type: "progress", id: message.id, stage: "Loading local model", progress: 75 });
            this.emit("message", { type: "result", id: message.id, text });
          });
        }
      }

      emit(type: string, data: unknown) {
        if (type === "message") this.onmessage?.({ data });
        for (const listener of this.listeners.get(type) ?? []) {
          listener({ data });
        }
      }

      terminate() {}
    }

    Object.assign(window, {
      Worker: MockWorker,
      __localAiTest: { workerUrls, aiRequests },
    });
  });

  await page.goto("/gh-aw/editor/");
  const runButton = page.getByRole("button", { name: "Run locally" });
  await expect(runButton).toBeEnabled();

  const workersBeforeRun = await page.evaluate(() => (window as typeof window & { __localAiTest: { workerUrls: string[] } }).__localAiTest.workerUrls.filter(url => url.includes("inference-worker.js")).length);
  expect(workersBeforeRun).toBe(0);

  await page.getByLabel("Local AI prompt").fill("Fix this workflow");
  await runButton.click();

  await expect(page.locator("#aiProgress")).toBeVisible();
  await expect(page.locator("#aiProgressText")).toHaveText("Workflow updated (100%)");
  await expect(page.locator("#editorTextarea")).toHaveValue(/name: locally-fixed/);

  const result = await page.evaluate(
    () =>
      (
        window as typeof window & {
          __localAiTest: { workerUrls: string[]; aiRequests: Array<{ diagnostic: string }> };
        }
      ).__localAiTest
  );
  expect(result.workerUrls.filter(url => url.includes("inference-worker.js"))).toHaveLength(1);
  expect(result.aiRequests).toHaveLength(2);
  expect(result.aiRequests[1].diagnostic).toBe("invalid workflow");
});
