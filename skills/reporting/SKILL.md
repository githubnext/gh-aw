---
name: reporting
description: Guidelines for formatting reports using HTML details/summary tags
---

# Report Format Guidelines

This skill provides guidelines for formatting reports with collapsible sections.

## Use HTML Details/Summary Tags

To prevent excessive scrolling and improve readability, **wrap your reports in HTML `<details>` and `<summary>` tags**. This allows users to expand and collapse sections as needed.

**Basic Structure:**

```markdown
<details>
<summary>📊 Report Title - [Date]</summary>

## Report Content

Your detailed report content goes here...

### Section 1

Content for section 1...

### Section 2

Content for section 2...

</details>
```
