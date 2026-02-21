---
title: Workflow Editors
description: A curated list of editors for authoring and previewing agentic workflows.
---

> [!WARNING]
> All editors listed here are **experimental**. They may change without notice and are not covered by any stability guarantees.

The following editors can be used to author, compile, and preview agentic workflows. Some are built-in tools maintained alongside gh-aw; others are community-created projects.

## Built-in editors

### Compiler Playground

**Maintained by:** GitHub Next  
**Status:** Experimental, built-in

An interactive browser-based playground that runs the gh-aw compiler entirely in the browser using [WebAssembly](/gh-aw/reference/wasm-compilation/). It demonstrates how to use the WASM build of the compiler directly in a webpage and shows how to compile workflows in the browser using the WASM-based execution engine.

Features:

- Live compilation of workflow markdown into GitHub Actions YAML
- Syntax highlighting and hover tooltips for frontmatter keys
- Several built-in sample workflows to get started
- Auto-saves editor content to `localStorage` across sessions
- No installation or server required — runs entirely in the browser

[Open Playground →](/gh-aw/editor/)

## Community editors

> [!NOTE]
> Community editors are created and maintained by independent contributors. They are not officially supported by the gh-aw project.

### Graphical Workflow Editor

**Maintained by:** [Mossaka](https://github.com/mossaka)  
**Status:** Experimental, community-maintained

A visual, graphical workflow editor that provides a richer UI for editing agentic workflows. Rather than working directly with markdown and YAML, this editor focuses on a more interactive and visual editing experience.

[Open Graphical Editor →](https://mossaka.github.io/gh-aw-editor-visualizer/)
