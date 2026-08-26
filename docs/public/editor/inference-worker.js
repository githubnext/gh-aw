/**
 * Browser-local workflow repair worker.
 *
 * This module is only requested after the user clicks "Run locally", so both
 * Transformers.js and the model stay off the network during normal editing.
 */

import { pipeline } from "https://cdn.jsdelivr.net/npm/@huggingface/transformers@4.2.0/dist/transformers.min.js";

const MODEL = {
  repository: "onnx-community/SmolLM2-360M-Instruct-ONNX",
  revision: "fe7c7db",
  dtype: "q4f16",
  device: "webgpu",
};

let generator = null;
let initialization = null;
let generating = false;

function send(message) {
  self.postMessage(message);
}

function progressValue(event) {
  if (!event || typeof event !== "object") return null;
  const value = Reflect.get(event, "progress");
  if (typeof value !== "number" || !Number.isFinite(value)) return null;
  return Math.max(0, Math.min(100, value));
}

async function assertWebGpu() {
  if (!self.navigator.gpu) {
    throw new Error("Local AI requires a browser with WebGPU support.");
  }

  const adapter = await self.navigator.gpu.requestAdapter({ powerPreference: "high-performance" });
  if (!adapter) {
    throw new Error("A WebGPU adapter is not available on this device.");
  }
}

async function getGenerator(id) {
  if (generator) return generator;
  if (initialization) return initialization;

  initialization = (async () => {
    send({ id, type: "progress", stage: "Checking WebGPU", progress: 0 });
    await assertWebGpu();
    send({ id, type: "progress", stage: "Loading local model", progress: 0 });

    return pipeline("text-generation", MODEL.repository, {
      revision: MODEL.revision,
      dtype: MODEL.dtype,
      device: MODEL.device,
      progress_callback: event => {
        const progress = progressValue(event);
        if (progress !== null) {
          send({ id, type: "progress", stage: "Loading local model", progress });
        }
      },
    });
  })();

  try {
    generator = await initialization;
    return generator;
  } finally {
    initialization = null;
  }
}

function validateRequest(request) {
  if (!request || typeof request !== "object" || request.type !== "generate") {
    throw new Error("Invalid local AI request.");
  }
  if (!Number.isInteger(request.id)) {
    throw new Error("Invalid local AI request ID.");
  }
  if (typeof request.prompt !== "string" || request.prompt.length < 1 || request.prompt.length > 500) {
    throw new Error("The prompt must contain between 1 and 500 characters.");
  }
  if (typeof request.workflow !== "string" || request.workflow.length > 30000) {
    throw new Error("The workflow is too large for local AI.");
  }
  if (typeof request.diagnostic !== "string" || request.diagnostic.length > 8000) {
    throw new Error("The compiler diagnostic is too large for local AI.");
  }
}

function buildMessages(request) {
  return [
    {
      role: "system",
      content: ["You edit GitHub Agentic Workflow Markdown.", "Return only the complete updated Markdown document inside <workflow> and </workflow> tags.", "Preserve unrelated content. Never return shell commands or explanations."].join(
        " "
      ),
    },
    {
      role: "user",
      content: [`Requested change: ${request.prompt}`, request.diagnostic ? `Compiler diagnostic to fix:\n${request.diagnostic}` : "", `Current workflow:\n${request.workflow}`].filter(Boolean).join("\n\n"),
    },
  ];
}

function extractText(output) {
  if (!Array.isArray(output)) return "";
  const generated = output[0]?.generated_text;
  if (typeof generated === "string") return generated;
  if (!Array.isArray(generated)) return "";
  const last = generated.at(-1);
  return typeof last?.content === "string" ? last.content : "";
}

self.addEventListener("message", event => {
  const request = event.data;
  const id = Number.isInteger(request?.id) ? request.id : -1;

  void (async () => {
    validateRequest(request);
    if (generating) throw new Error("Local AI is already generating a workflow.");

    generating = true;
    try {
      const model = await getGenerator(id);
      send({ id, type: "progress", stage: "Generating workflow", progress: null });
      const output = await model(buildMessages(request), {
        add_generation_prompt: true,
        do_sample: false,
        max_new_tokens: 1024,
      });
      send({ id, type: "result", text: extractText(output) });
    } finally {
      generating = false;
    }
  })().catch(error => {
    send({ id, type: "error", error: error instanceof Error ? error.message : String(error) });
  });
});
