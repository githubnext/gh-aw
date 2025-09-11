# Visual Studio Code Integration

The `gh aw` cli provides a few tools to improve your developer experience in Visual Studio Code (or other IDEs).

## Copilot instructions <a id="copilot-instructions"></a>

If you add the `--instructions` flag to the compile command, it will also
write a [custom Copilot intructions file](https://code.visualstudio.com/docs/copilot/copilot-customization) at `.github/instructions/github-agentic-workflows.instructions.md`.

```sh
gh aw compile --instructions
```

The instructions will automatically be imported by Copilot when authoring markdown
files under the `.github/workflows` folder.

Once configured, you will notice that Copilot Chat will be much more efficient at
generating Agentic Workflows.

## Background Compilation using Tasks

You can leverage tasks in Visual Studio Code to configure a background compilation of Agentic Workflows.

- open or create `.vscode/tasks.json`
- add or merge the following JSON:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Compile Github Agentic Workflows",
      "dependsOn": ["Compile gh-aw"],
      "type": "shell",
      "command": "./gh-aw",
      "args": ["compile", "--watch"],
      "isBackground": true,
      "problemMatcher": {
        "owner": "gh-aw",
        "fileLocation": "relative",
        "pattern": {
          "regexp": "^(.*?):(\\d+):(\\d+):\\s(error|warning):\\s(.+)$",
          "file": 1,
          "line": 2,
          "column": 3,
          "severity": 4,
          "message": 5
        },
        "background": {
          "activeOnStart": true,
          "beginsPattern": "Watching for file changes",
          "endsPattern": "Recompiled"
        }
      },
      "group": { "kind": "build", "isDefault": true },
      "runOptions": { "runOn": "folderOpen" }
    }
  ]
}
```

The background compilation should start as soon as you open a Markdown file under `.github/workflows/`. If it does not start, 

- open the command palette (`Ctrl + Shift + P`)
- type `Tasks: Run Task` to start the task once
- or type `Tasks: Managed Automatic Tasks` and select `Allow Automatic Tasks` to start it automatically.
