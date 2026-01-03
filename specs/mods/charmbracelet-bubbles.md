# Charmbracelet Bubbles Usage Summary

**Module**: `github.com/charmbracelet/bubbles`  
**Current Version**: `v0.21.1-0.20250623103423-23b8fd6302d7` (pseudo-version based on v0.21.1)  
**Target Version**: `v2.0.0` (when stable; RC available as v2.0.0-rc.1)  
**Purpose**: TUI components for building terminal user interfaces with Bubble Tea  
**Documentation**: https://github.com/charmbracelet/bubbles

## Overview

Charmbracelet Bubbles provides pre-built, reusable components for terminal user interfaces built with the Bubble Tea framework. The gh-aw project uses three components: progress bars, spinners, and interactive lists.

## Components Used

### 1. Progress (`pkg/console/progress.go`)

**Purpose**: Display progress bars for long-running operations

**APIs Used**:
- `progress.New()` - Create progress bar model
- `progress.WithDefaultGradient()` - Configure gradient colors
- `progress.WithWidth()` - Set progress bar width
- `Model.ViewAs(percent float64)` - Render progress at percentage

**Usage Pattern**:
```go
prog := progress.New(
    progress.WithDefaultGradient(),
    progress.WithWidth(40),
)
prog.FullColor = styles.ColorSuccess.Dark
prog.EmptyColor = styles.ColorComment.Dark
return prog.ViewAs(percent)
```

**Used By**:
- `pkg/cli/logs_orchestrator.go` - Download progress for workflow logs
- 13 total usages across CLI commands

### 2. Spinner (`pkg/console/spinner.go`)

**Purpose**: Show loading/activity indicators with TTY detection

**APIs Used**:
- `spinner.New()` - Create spinner model
- `spinner.WithSpinner(spinner.Dot)` - Set spinner style
- `spinner.WithStyle()` - Apply lipgloss styling
- `Model.Tick()` - Get tick message for animation
- `Model.Update()` - Update spinner state
- `Model.View()` - Render spinner frame

**Usage Pattern**:
```go
s := spinner.New(
    spinner.WithSpinner(spinner.Dot),
    spinner.WithStyle(styles.Info),
)
msg := s.model.Tick()
s.model, _ = s.model.Update(msg)
view := s.model.View()
```

**Used By**:
- `pkg/cli/interactive.go` - Workflow compilation
- `pkg/cli/logs_download.go` - Artifact downloads
- `pkg/cli/mcp_registry.go` - MCP server fetching
- `pkg/cli/run_command.go` - Waiting for workflow runs
- `pkg/cli/status_command.go` - Fetching workflow status
- `pkg/cli/workflows.go` - GitHub API operations
- 12+ total usages across CLI commands

### 3. List (`pkg/console/list.go`)

**Purpose**: Interactive lists with keyboard navigation and filtering

**APIs Used**:
- `list.New(items, delegate, width, height)` - Create list model
- `list.Model.SetShowStatusBar()` - Configure status bar
- `list.Model.SetFilteringEnabled()` - Enable search
- `list.Model.SetShowHelp()` - Show help text
- `list.Model.SelectedItem()` - Get selected item
- `list.Model.Update()` - Handle input
- `list.Model.View()` - Render list
- `list.Item` interface - Custom item types

**Usage Pattern**:
```go
l := list.New(listItems, delegate, 80, 20)
l.Title = title
l.SetShowStatusBar(false)
l.SetFilteringEnabled(true)
l.SetShowHelp(true)
l.Styles.Title = lipgloss.NewStyle()...
```

**Used By**:
- `pkg/cli/add_command.go` - Interactive workflow selection
- `pkg/cli/mcp_list.go` - MCP server workflow selection
- 2 total usages for interactive selection

## Dependencies

### Direct
- `charm.land/bubbletea` (or `github.com/charmbracelet/bubbletea` pre-v2) - Core TUI framework
- `github.com/charmbracelet/lipgloss` - Styling and layout

### Related Packages
- `github.com/charmbracelet/huh` (v0.8.0) - Form components (separate from Bubbles)

## Migration to v2

**Status**: Planning phase, awaiting stable v2.0.0 release

See **[specs/bubbles-v2-migration.md](../bubbles-v2-migration.md)** for comprehensive migration guide.

### Key v2 Changes

1. **Import Path Change** (Critical):
   - Old: `github.com/charmbracelet/bubbles`
   - New: `charm.land/bubbles/v2`

2. **Bubble Tea v2 Required**:
   - Must also migrate to `charm.land/bubbletea/v2`

3. **API Compatibility**:
   - ✅ Progress: Compatible, improved blend algorithm
   - ✅ Spinner: Compatible, no significant changes
   - ✅ List: Compatible, cursor position bug fixes

4. **Files to Update**:
   - `pkg/console/progress.go`
   - `pkg/console/spinner.go`
   - `pkg/console/list.go`
   - `go.mod`

## Improvement Opportunities

### Quick Wins

1. **Multi-stop Gradients** (v2 feature)
   ```go
   // Current: 2-color gradient
   prog := progress.New(progress.WithDefaultGradient())
   
   // V2: Custom multi-stop gradient for visual stages
   stops := []progress.GradientStop{
       {Color: lipgloss.Color("#00FF00"), Pos: 0.0},   // Green: starting
       {Color: lipgloss.Color("#FFFF00"), Pos: 0.5},   // Yellow: processing
       {Color: lipgloss.Color("#FF0000"), Pos: 1.0},   // Red: finalizing
   }
   prog := progress.New(progress.WithGradientStops(stops))
   ```
   
   **Use case**: Provide visual feedback for different stages in long-running operations

2. **Better Error Context in Wrappers**
   - Current wrappers (`pkg/console/*.go`) provide basic TTY detection
   - Could add error context and fallback messaging
   - Example: Explain why spinner is disabled in CI environments

### Feature Opportunities

1. **Viewport Component** (Not currently used)
   - Add scrollable log viewers for workflow output
   - Use horizontal scrolling for wide output
   - Integrate syntax highlighting for JSON/YAML logs
   
   ```go
   // Potential use case: Interactive log viewer
   viewport := viewport.New(80, 20)
   viewport.SetContent(workflowLogs)
   // Enable horizontal mouse wheel scrolling
   ```

2. **Textarea Component** (Not currently used)
   - Add interactive workflow editing
   - Use for creating workflow templates
   - Leverage PageUp/PageDown, ScrollPosition methods
   
   ```go
   // Potential use case: Edit workflow frontmatter
   textarea := textarea.New()
   textarea.SetValue(existingFrontmatter)
   textarea.ShowLineNumbers = true
   ```

3. **Table Component** (Not currently used)
   - Display workflow run history in table format
   - Show MCP server configurations
   - Sortable columns, row selection

### Best Practices

1. **TTY Detection Consistency**
   - ✅ Current implementation correctly wraps components with TTY detection
   - ✅ Provides text-based fallbacks for non-TTY environments
   - Keep this pattern when adding new components

2. **Component Wrapping**
   - ✅ Isolate Bubbles usage in `pkg/console/` package
   - ✅ Provide simple APIs for CLI commands
   - Continue this abstraction layer

3. **Testing**
   - ✅ Each component has tests (`*_test.go`)
   - Consider adding visual regression tests for v2 migration
   - Test both TTY and non-TTY modes

4. **Error Handling**
   - Current: Components don't expose detailed errors
   - Consider: Add error channels or callbacks for async operations

## Architecture

### Component Isolation

```
CLI Commands (pkg/cli/*)
    ↓
Console Wrappers (pkg/console/*)
    ↓
Bubbles Components
    ↓
Bubble Tea Framework
    ↓
Terminal (with TTY detection)
```

**Benefits**:
- CLI code doesn't directly depend on Bubbles
- Easy to mock/test console operations
- Migration to v2 isolated to 3 files
- Can swap implementations if needed

### TTY Detection Strategy

All components detect TTY and provide fallbacks:

| Component | TTY Mode | Non-TTY Mode |
|-----------|----------|--------------|
| **Progress** | Gradient bar with colors | Text percentage (e.g., "50% (512MB/1024MB)") |
| **Spinner** | Animated dot with color | Disabled (no output) |
| **List** | Interactive with arrow keys | Numbered list with stdin selection |

Implementation in `pkg/tty/tty.go`:
```go
func IsStderrTerminal() bool {
    return term.IsTerminal(int(os.Stderr.Fd()))
}
```

## Research Findings

### From Bubbles Repository

1. **v2 Release Status** (as of 2026-01-03):
   - RC released November 2024
   - Stable v2.0.0 pending
   - Community testing ongoing
   - Few breaking changes reported

2. **Notable v2 Features**:
   - Real cursor support in textarea/textinput (🖱️ mouse support)
   - Automatic color detection for light/dark modes (🎨 adaptive theming)
   - Viewport syntax highlighting and line numbers (📜 better code display)
   - Improved blend algorithm in progress bars (🌈 smoother gradients)

3. **Community Adoption**:
   - Active development and maintenance
   - Strong community support
   - Well-documented API changes
   - Migration guide available

4. **Stability**:
   - RC has been available for 2+ months
   - Bug fixes and refinements ongoing
   - Recommend waiting for stable v2.0.0

### Performance Considerations

From changelog and discussions:
- Improved blend algorithm may have performance impact (profile before/after)
- Cursor blink fixes reduce data races
- List cursor position fixes improve responsiveness

## Version History

| Version | Date | Notes |
|---------|------|-------|
| v0.21.1 | 2024 | Current production version |
| v0.21.1-0.20250623103423-23b8fd6302d7 | 2025-06-23 | Pseudo-version we're using |
| v2.0.0-rc.1 | 2024-11 | Release candidate with breaking changes |
| v2.0.0 | TBD | Awaiting stable release |

## References

- [Bubbles GitHub Repository](https://github.com/charmbracelet/bubbles)
- [Bubbles v2.0.0-rc.1 Release Notes](https://github.com/charmbracelet/bubbles/releases/tag/v2.0.0-rc.1)
- [v2 Migration Guide](../bubbles-v2-migration.md)
- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [Bubble Tea v2 Migration](https://github.com/charmbracelet/bubbletea/discussions/1374)
- [API Documentation (v2)](https://pkg.go.dev/charm.land/bubbles/v2)
- [Charm Ecosystem](https://charm.land/)

## Related Specifications

- [Console Rendering](../console-rendering.md) - Struct tag-based rendering system
- [Styles Guide](../styles-guide.md) - Color and styling conventions
- [TTY Detection](../../pkg/tty/tty.go) - Terminal detection implementation

---

**Last Updated**: 2026-01-03  
**Next Review**: Upon v2.0.0 stable release  
**Owner**: Development team
