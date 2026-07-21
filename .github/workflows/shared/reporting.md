---
### Report formatting guidelines
---

### Report structure guidelines

- Use `###` (or lower) headers only.
- Keep summary and critical actions visible; move long detail into `<details>` blocks.
- Structure reports as: overview → key metrics/issues → collapsible detail → next actions.
- For long lists/tables (>5 items), use:
  ```markdown
  <details>
  <summary><b>View Details</b></summary>

  - item 1
  - item 2

  </details>
  ```

### Workflow run references

- Format run IDs as links: `[§12345](https://github.com/owner/repo/actions/runs/12345)`
- Include up to 3 most relevant run URLs at end under `**References:**`
- Do NOT add footer attribution (system adds automatically)
