import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { createHash } from "node:crypto";

// Development-only generation; the executable uses only the local snapshots.
const root = new URL("../../../../", import.meta.url);
const output = new URL("../catalogs/", import.meta.url);
mkdirSync(output, { recursive: true });
const sources = ["actions/setup/js/github_notification_webhooks.json", "pkg/workflow/data/provider_issue_webhooks.json", "pkg/workflow/data/webhook_events.json"];
const provenance = { baseline: "527e59970ad970f26f8e1038b6f301cef80cf876", files: {} };
for (const source of sources) {
  const raw = readFileSync(new URL(source, root));
  let catalog = JSON.parse(raw);
  if (source.endsWith("/webhook_events.json")) {
    catalog = Object.fromEntries(["projects_v2_item", "code_scanning_alert", "dependabot_alert"].map(event => [event, catalog[event]]));
  }
  const filename = source.split("/").at(-1);
  provenance.files[filename] = { source, sha256: createHash("sha256").update(raw).digest("hex") };
  writeFileSync(new URL(filename, output), `${JSON.stringify(catalog, null, 2)}\n`);
}
writeFileSync(new URL("provenance.json", output), `${JSON.stringify(provenance, null, 2)}\n`);
