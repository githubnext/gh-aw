---
on:
  workflow_dispatch:

permissions:
  contents: read
  actions: read

engine: copilot

tools:
  bash:
    - "find *"
    - "ls *"
    - "du *"
    - "wc *"
    - "cat *"
    - "head *"
    - "tail *"
    - "sort *"
    - "uniq *"
    - "awk *"
    - "sed *"
    - "grep *"
    - "tree *"
    - "stat *"

safe-outputs:
  create-discussion:
    category: "dev"
    max: 1

timeout_minutes: 10
---

# Repository Tree Map Generator

Generate a comprehensive ASCII tree map visualization of the repository file structure.

## Mission

Your task is to analyze the repository structure and create an ASCII tree map that visualizes:
1. Directory hierarchy
2. File sizes (relative visualization)
3. File counts per directory
4. Key statistics about the repository

## Analysis Steps

### 1. Collect Repository Statistics

Use bash tools to gather:
- **Total file count** across the repository
- **Total repository size** (excluding .git directory)
- **File type distribution** (count by extension)
- **Largest files** in the repository (top 10)
- **Largest directories** by total size
- **Directory depth** and structure

Example commands you might use:
```bash
# Count total files
find . -type f -not -path "./.git/*" | wc -l

# Get repository size
du -sh . --exclude=.git

# Count files by extension
find . -type f -not -path "./.git/*" | sed 's/.*\.//' | sort | uniq -c | sort -rn | head -20

# Find largest files
find . -type f -not -path "./.git/*" -exec du -h {} + | sort -rh | head -10

# Directory sizes
du -h --max-depth=2 --exclude=.git . | sort -rh | head -15
```

### 2. Generate ASCII Tree Map

Create an ASCII visualization that shows:
- **Directory tree structure** with indentation
- **Size indicators** using symbols or bars (e.g., █ ▓ ▒ ░)
- **File counts** in brackets [count]
- **Relative size representation** (larger files/directories shown with more bars)

Example visualization format:
```
Repository Tree Map
===================

/ [1234 files, 45.2 MB]
│
├─ .github/ [156 files, 2.3 MB] ████████░░
│  ├─ workflows/ [89 files, 1.8 MB] ██████░░
│  └─ actions/ [12 files, 234 KB] ██░░
│
├─ pkg/ [456 files, 28.5 MB] ██████████████████░░
│  ├─ cli/ [78 files, 5.2 MB] ████░░
│  ├─ parser/ [34 files, 3.1 MB] ███░░
│  └─ workflow/ [124 files, 12.8 MB] ████████░░
│
├─ docs/ [234 files, 8.7 MB] ██████░░
│  └─ src/ [189 files, 7.2 MB] █████░░
│
└─ cmd/ [45 files, 2.1 MB] ██░░
```

### 3. Include Summary Statistics

Create a summary section with:

**Repository Overview:**
- Total files: [count]
- Total size: [size]
- Average file size: [size]
- Total directories: [count]
- Maximum directory depth: [levels]

**File Type Distribution (Top 10):**
- .go: [count] files ([percentage]%)
- .md: [count] files ([percentage]%)
- .js: [count] files ([percentage]%)
- ... etc

**Largest Files:**
1. path/to/file.ext - [size]
2. path/to/file2.ext - [size]
... (top 10)

**Largest Directories:**
1. pkg/workflow - [size] ([file count] files)
2. docs/src - [size] ([file count] files)
... (top 10)

### 4. Visualization Guidelines

- Use **box-drawing characters** for tree structure: │ ├ └ ─
- Use **block characters** for size bars: █ ▓ ▒ ░
- Scale the visualization bars **proportionally** to sizes
- Keep the tree **readable** - don't go too deep (max 3-4 levels recommended)
- Add **color indicators** using emojis:
  - 📁 for directories
  - 📄 for files
  - 🔧 for config files
  - 📚 for documentation
  - 🧪 for test files

### 5. Output Format

Create a GitHub discussion with:
- **Title**: "Repository Tree Map - [current date]"
- **Body**: Your complete tree map visualization with all sections
- Use proper markdown formatting with code blocks for the ASCII art
- Include a timestamp and repository information

## Important Notes

- **Exclude .git directory** from all calculations to avoid skewing results
- **Handle special characters** in filenames properly
- **Format sizes** in human-readable units (KB, MB, GB)
- **Round percentages** to 1-2 decimal places
- **Sort intelligently** - largest first for most sections
- **Be creative** with the ASCII visualization but keep it readable
- **Test your bash commands** before including them in analysis
- The tree map should give a **quick visual understanding** of the repository structure and size distribution

## Security

Treat all repository content as trusted since you're analyzing the repository you're running in. However:
- Don't execute any code files
- Don't read sensitive files (.env, secrets, etc.)
- Focus on file metadata (sizes, counts, names) rather than content
