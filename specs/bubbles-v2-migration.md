# Bubbles v2 Migration Guide

## Overview

This document provides a comprehensive migration plan for upgrading from Bubbles v0.21.x to v2.0.0 when it becomes the stable release. Charmbracelet Bubbles v2.0.0-rc.1 was released in November 2024 with significant improvements and breaking changes.

**Current version**: `v0.21.1-0.20250623103423-23b8fd6302d7` (pseudo-version based on v0.21.1)  
**Target version**: `v2.0.0` (when stable)  
**Release candidate**: `v2.0.0-rc.1` (available now for testing)

## Components We Use

The gh-aw project uses three Bubbles components:

1. **Progress** (`pkg/console/progress.go`) - Progress bars for long operations
2. **Spinner** (`pkg/console/spinner.go`) - Loading spinners with TTY detection
3. **List** (`pkg/console/list.go`) - Interactive lists with keyboard navigation

## Breaking Changes That Affect Us

### 1. Import Path Change (Critical)

**Most significant breaking change**: All imports now use the `charm.land` vanity domain instead of `github.com`.

**Current import:**
```go
import (
    "github.com/charmbracelet/bubbles/progress"
    "github.com/charmbracelet/bubbles/spinner"
    "github.com/charmbracelet/bubbles/list"
)
```

**New import (v2):**
```go
import (
    "charm.land/bubbles/v2/progress"
    "charm.land/bubbles/v2/spinner"
    "charm.land/bubbles/v2/list"
)
```

**Impact**: All three of our component files must be updated.

### 2. Bubble Tea Dependency

Bubbles v2 depends on Bubble Tea v2, which also uses the `charm.land` domain:

**Current:**
```go
tea "github.com/charmbracelet/bubbletea"
```

**New (v2):**
```go
tea "charm.land/bubbletea/v2"
```

**Files affected:**
- `pkg/console/list.go` (uses Bubble Tea for interactive list)

### 3. API Changes by Component

#### Progress Component

**Current API (v0.21.x):**
```go
prog := progress.New(
    progress.WithDefaultGradient(),
    progress.WithWidth(40),
)
prog.FullColor = styles.ColorSuccess.Dark
prog.EmptyColor = styles.ColorComment.Dark
```

**V2 improvements:**
- ✅ API remains compatible
- 🆕 Multiple stops support for gradient colors
- 🆕 Improved blend algorithm for smoother gradients
- No breaking changes detected in our usage pattern

**Migration status**: ✅ No code changes required (API compatible)

#### Spinner Component

**Current API (v0.21.x):**
```go
s.model = spinner.New(
    spinner.WithSpinner(spinner.Dot),
    spinner.WithStyle(styles.Info),
)
msg := s.model.Tick()
s.model, _ = s.model.Update(msg)
view := s.model.View()
```

**V2 status:**
- ✅ API remains compatible
- No significant changes to spinner component in v2.0.0-rc.1 changelog
- Our usage pattern (Tick, Update, View) remains the same

**Migration status**: ✅ No code changes required (API compatible)

#### List Component

**Current API (v0.21.x):**
```go
l := list.New(listItems, delegate, 80, 20)
l.Title = title
l.SetShowStatusBar(false)
l.SetFilteringEnabled(true)
l.SetShowHelp(true)
l.Styles.Title = lipgloss.NewStyle()...
```

**V2 improvements:**
- ✅ API remains compatible
- 🐛 Fixed cursor position issues with page/cursor methods (#831, #837)
- 🆕 Updated keybindings in setSize method
- Our usage pattern remains compatible

**Migration status**: ✅ No code changes required (API compatible)

## Required Code Changes

### Files to Modify

1. **pkg/console/progress.go**
   - Update import: `github.com/charmbracelet/bubbles/progress` → `charm.land/bubbles/v2/progress`
   - No API changes required

2. **pkg/console/spinner.go**
   - Update import: `github.com/charmbracelet/bubbles/spinner` → `charm.land/bubbles/v2/spinner`
   - No API changes required

3. **pkg/console/list.go**
   - Update imports:
     - `github.com/charmbracelet/bubbles/list` → `charm.land/bubbles/v2/list`
     - `github.com/charmbracelet/bubbletea` → `charm.land/bubbletea/v2`
   - No API changes required

4. **go.mod**
   - Update dependency: `github.com/charmbracelet/bubbles` → `charm.land/bubbles/v2`
   - Add dependency: `charm.land/bubbletea/v2`
   - May also need to update `charm.land/lipgloss/v2` for compatibility

### Migration Example

**Before (current):**
```go
// pkg/console/progress.go
package console

import (
    "fmt"
    "github.com/charmbracelet/bubbles/progress"
    "github.com/githubnext/gh-aw/pkg/styles"
)

type ProgressBar struct {
    progress progress.Model
    total    int64
    current  int64
}

func NewProgressBar(total int64) *ProgressBar {
    prog := progress.New(
        progress.WithDefaultGradient(),
        progress.WithWidth(40),
    )
    prog.FullColor = styles.ColorSuccess.Dark
    prog.EmptyColor = styles.ColorComment.Dark
    return &ProgressBar{progress: prog, total: total}
}
```

**After (v2):**
```go
// pkg/console/progress.go
package console

import (
    "fmt"
    "charm.land/bubbles/v2/progress"  // ← Only change
    "github.com/githubnext/gh-aw/pkg/styles"
)

type ProgressBar struct {
    progress progress.Model
    total    int64
    current  int64
}

func NewProgressBar(total int64) *ProgressBar {
    prog := progress.New(
        progress.WithDefaultGradient(),
        progress.WithWidth(40),
    )
    prog.FullColor = styles.ColorSuccess.Dark
    prog.EmptyColor = styles.ColorComment.Dark
    return &ProgressBar{progress: prog, total: total}
}
```

## Testing Strategy

### Pre-Migration Testing

1. **Run existing tests** to establish baseline:
   ```bash
   make test-unit
   cd pkg/console && go test -v
   ```

2. **Verify current behavior** in real usage:
   - Progress bar: `gh aw logs download <run-id>`
   - Spinner: `gh aw compile <workflow>`
   - List: `gh aw add` (interactive selection)

### Migration Testing

1. **Create a migration branch:**
   ```bash
   git checkout -b bubbles-v2-migration
   ```

2. **Update imports systematically:**
   - Use find/replace: `github.com/charmbracelet/bubbles` → `charm.land/bubbles/v2`
   - Update Bubble Tea: `github.com/charmbracelet/bubbletea` → `charm.land/bubbletea/v2`

3. **Update go.mod:**
   ```bash
   go get charm.land/bubbles/v2@v2.0.0  # When stable
   go get charm.land/bubbletea/v2@latest
   go mod tidy
   ```

4. **Run tests:**
   ```bash
   make test-unit
   cd pkg/console && go test -v
   ```

5. **Manual verification:**
   ```bash
   # Build and test each component
   make build
   
   # Test progress bar
   ./gh-aw logs download <run-id>
   
   # Test spinner
   ./gh-aw compile examples/hello-world.md
   
   # Test interactive list
   ./gh-aw add
   ./gh-aw mcp list  # Then select a workflow
   ```

6. **Visual regression testing:**
   - Verify progress bar gradient rendering
   - Check spinner animation smoothness
   - Test list navigation (arrow keys, search, selection)
   - Verify TTY detection still works in non-TTY environments

### Test Cases

#### Progress Bar Tests
- [x] Create progress bar with total size
- [x] Update progress to various percentages
- [x] Handle edge case: total = 0
- [x] TTY vs non-TTY output format
- [x] Byte formatting (B, KB, MB, GB)
- [ ] **New**: Verify improved blend algorithm rendering

#### Spinner Tests
- [x] Create spinner with message
- [x] Start/stop animation
- [x] Update message while running
- [x] Stop with final message
- [x] TTY detection (enabled/disabled)
- [ ] **New**: Verify animation smoothness

#### List Tests
- [x] Create list with multiple items
- [x] Keyboard navigation (up/down)
- [x] Selection with Enter
- [x] Cancel with Esc/Ctrl+C
- [x] Filter/search functionality
- [x] Non-TTY fallback (numbered list)
- [ ] **New**: Verify cursor position fixes (#831)
- [ ] **New**: Test page navigation improvements

## New Features to Consider

While our current usage is straightforward, v2 introduces features we could leverage:

### Progress Component

**Multiple stops for custom gradients:**
```go
// Current: Default gradient (2 colors)
prog := progress.New(progress.WithDefaultGradient())

// V2: Custom multi-stop gradient
stops := []progress.GradientStop{
    {Color: lipgloss.Color("#00FF00"), Pos: 0.0},   // Green at start
    {Color: lipgloss.Color("#FFFF00"), Pos: 0.5},   // Yellow at 50%
    {Color: lipgloss.Color("#FF0000"), Pos: 1.0},   // Red at end
}
prog := progress.New(progress.WithGradientStops(stops))
```

**Use case**: Visual feedback for different progress stages (e.g., green = starting, yellow = processing, red = finalizing)

### Textarea Component (Not currently used)

If we add text editing capabilities in the future:
- PageUp/PageDown navigation
- ScrollYOffset and ScrollPosition for large text
- MoveToBeginning/MoveToEnd methods
- Improved cursor handling with real cursor support

### Viewport Component (Not currently used)

If we add scrollable output views:
- Horizontal mouse wheel scrolling
- Better syntax highlighting integration
- Line number support
- Soft-wrapping improvements

## Timeline Recommendation

### Phase 1: Preparation (Now)

- [x] Document current usage patterns
- [x] Review v2 breaking changes
- [x] Create migration plan
- [ ] Monitor v2 stability and release status

### Phase 2: Testing (When v2.0.0 stable)

**Estimated effort**: 1-2 hours

- [ ] Wait for stable v2.0.0 release (not RC)
- [ ] Create migration branch
- [ ] Update imports (3 files)
- [ ] Run test suite
- [ ] Manual testing of all three components
- [ ] Document any issues

### Phase 3: Migration (After testing)

**Estimated effort**: 30 minutes (if testing successful)

- [ ] Merge migration PR
- [ ] Update documentation if needed
- [ ] Monitor for issues in production use

### Phase 4: Optimization (Optional, Future)

**Estimated effort**: 2-4 hours

- [ ] Evaluate new gradient features for progress bars
- [ ] Consider adding textarea/viewport if needed
- [ ] Optimize rendering performance with v2 improvements

## Risk Assessment

### Low Risk ✅

Our usage of Bubbles is:
- **Isolated**: Components are wrapped in `pkg/console/` package
- **Simple**: We use basic APIs, not advanced features
- **Tested**: Each component has tests
- **Abstracted**: Other code depends on our wrappers, not Bubbles directly

### Potential Issues

1. **Import path changes across ecosystem**
   - Risk: Other Charm libraries may need updates (Lipgloss, Bubbletea)
   - Mitigation: Update all Charm dependencies together

2. **Subtle rendering differences**
   - Risk: Progress gradient or spinner animation may look different
   - Mitigation: Visual regression testing before merge

3. **Performance changes**
   - Risk: v2 blend algorithm may be slower/faster
   - Mitigation: Monitor performance in real workflows

4. **Dependency conflicts**
   - Risk: Other dependencies may not be compatible with v2
   - Mitigation: Test full build before committing

## Rollback Plan

If migration causes issues:

1. **Revert the PR**:
   ```bash
   git revert <commit-hash>
   ```

2. **Pin to v0.21.x**:
   ```go
   // go.mod
   require github.com/charmbracelet/bubbles v0.21.1
   ```

3. **Report issues** to Charmbracelet if blocking bugs found

## References

- [Bubbles v2.0.0-rc.1 Release Notes](https://github.com/charmbracelet/bubbles/releases/tag/v2.0.0-rc.1)
- [Bubble Tea v2 Migration Guide](https://github.com/charmbracelet/bubbletea/discussions/1374)
- [Charm.land Documentation](https://charm.land/)
- [Bubbles v2 API Documentation](https://pkg.go.dev/charm.land/bubbles/v2)

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-01-03 | Wait for stable v2.0.0 | RC releases may have bugs; wait for stable |
| 2026-01-03 | Import-only migration initially | APIs are compatible; focus on import changes |
| 2026-01-03 | Defer new features | Minimize changes; evaluate features later |

## Checklist

### Documentation
- [x] Breaking changes documented
- [x] Migration steps clearly outlined
- [x] Code examples for required changes
- [x] Testing strategy defined
- [x] Timeline recommendation provided
- [x] New features documented for future consideration

### Pre-Migration
- [ ] Monitor v2.0.0 release status
- [ ] Check for v2.0.0 stable release announcement
- [ ] Review any additional changes in stable vs RC

### Migration
- [ ] Create migration branch
- [ ] Update imports in 3 files
- [ ] Update go.mod dependencies
- [ ] Run `make test-unit`
- [ ] Run component-specific tests
- [ ] Manual testing of progress bar
- [ ] Manual testing of spinner
- [ ] Manual testing of list
- [ ] Visual verification
- [ ] Performance check
- [ ] Update CHANGELOG.md
- [ ] Create PR with migration changes
- [ ] Code review
- [ ] Merge to main

### Post-Migration
- [ ] Monitor production usage
- [ ] Document any issues
- [ ] Consider new features for future work

---

**Status**: 📝 Planning complete, awaiting v2.0.0 stable release  
**Owner**: Development team  
**Last updated**: 2026-01-03
