// Package cli provides command-line interface functionality for gh-aw.
// This file (logs_pager.go) implements an interactive pager for viewing workflow logs
// using the Bubble Tea viewport component.
//
// Key responsibilities:
//   - Providing interactive navigation through large log outputs
//   - Implementing keyboard controls for scrolling and searching
//   - TTY detection and fallback to non-interactive mode
package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/styles"
	"github.com/githubnext/gh-aw/pkg/tty"
)

var logsPagerLog = logger.New("cli:logs_pager")

// PagerMode controls when the interactive pager is enabled
type PagerMode string

const (
	// PagerModeAuto automatically enables pager for large outputs in TTY
	PagerModeAuto PagerMode = "auto"
	// PagerModeAlways always enables pager (fails if not in TTY)
	PagerModeAlways PagerMode = "always"
	// PagerModeNever disables pager and uses standard output
	PagerModeNever PagerMode = "never"
)

// pagerKeyMap defines keyboard shortcuts for the pager
type pagerKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	PageUp    key.Binding
	PageDown  key.Binding
	HalfUp    key.Binding
	HalfDown  key.Binding
	Top       key.Binding
	Bottom    key.Binding
	Search    key.Binding
	NextMatch key.Binding
	PrevMatch key.Binding
	Help      key.Binding
	Quit      key.Binding
}

// defaultKeyMap returns the default key bindings
func defaultKeyMap() pagerKeyMap {
	return pagerKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("b", "pgup"),
			key.WithHelp("b/pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("space", "pgdown"),
			key.WithHelp("space/pgdn", "page down"),
		),
		HalfUp: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "half page up"),
		),
		HalfDown: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "half page down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g/home", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G/end", "bottom"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		NextMatch: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next match"),
		),
		PrevMatch: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "prev match"),
		),
		Help: key.NewBinding(
			key.WithKeys("?", "h"),
			key.WithHelp("?/h", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q/esc", "quit"),
		),
	}
}

// pagerModel is the Bubble Tea model for the interactive pager
type pagerModel struct {
	viewport    viewport.Model
	content     string
	keys        pagerKeyMap
	ready       bool
	showHelp    bool
	searchMode  bool
	searchTerm  string
	searchIndex int
	searchLines []int // Line numbers where search term appears
	helpContent string
}

// newPagerModel creates a new pager model with the given content
func newPagerModel(content string) pagerModel {
	keys := defaultKeyMap()

	// Build help content
	helpContent := buildHelpContent(keys)

	return pagerModel{
		content:     content,
		keys:        keys,
		searchIndex: -1,
		helpContent: helpContent,
	}
}

// buildHelpContent creates the help text display
func buildHelpContent(keys pagerKeyMap) string {
	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(styles.ColorInfo).
		Bold(true).
		Padding(0, 0, 1, 0)

	bindingStyle := lipgloss.NewStyle().
		Foreground(styles.ColorSuccess).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(styles.ColorForeground)

	sb.WriteString(titleStyle.Render("Keyboard Shortcuts"))
	sb.WriteString("\n\n")

	shortcuts := []struct {
		binding string
		desc    string
	}{
		{keys.Up.Help().Key, keys.Up.Help().Desc},
		{keys.Down.Help().Key, keys.Down.Help().Desc},
		{keys.PageUp.Help().Key, keys.PageUp.Help().Desc},
		{keys.PageDown.Help().Key, keys.PageDown.Help().Desc},
		{keys.HalfUp.Help().Key, keys.HalfUp.Help().Desc},
		{keys.HalfDown.Help().Key, keys.HalfDown.Help().Desc},
		{keys.Top.Help().Key, keys.Top.Help().Desc},
		{keys.Bottom.Help().Key, keys.Bottom.Help().Desc},
		{keys.Search.Help().Key, keys.Search.Help().Desc},
		{keys.NextMatch.Help().Key, keys.NextMatch.Help().Desc},
		{keys.PrevMatch.Help().Key, keys.PrevMatch.Help().Desc},
		{keys.Help.Help().Key, "toggle help"},
		{keys.Quit.Help().Key, keys.Quit.Help().Desc},
	}

	for _, sc := range shortcuts {
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			bindingStyle.Render(sc.binding),
			descStyle.Render(sc.desc)))
	}

	return sb.String()
}

// Init initializes the pager model
func (m pagerModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard input and updates the model
func (m pagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle search mode separately
		if m.searchMode {
			return m.handleSearchInput(msg)
		}

		// Handle help toggle
		if key.Matches(msg, m.keys.Help) {
			m.showHelp = !m.showHelp
			return m, nil
		}

		// Handle quit
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

		// Handle navigation
		if key.Matches(msg, m.keys.Up) {
			m.viewport.LineUp(1)
		} else if key.Matches(msg, m.keys.Down) {
			m.viewport.LineDown(1)
		} else if key.Matches(msg, m.keys.PageUp) {
			m.viewport.ViewUp()
		} else if key.Matches(msg, m.keys.PageDown) {
			m.viewport.ViewDown()
		} else if key.Matches(msg, m.keys.HalfUp) {
			m.viewport.HalfViewUp()
		} else if key.Matches(msg, m.keys.HalfDown) {
			m.viewport.HalfViewDown()
		} else if key.Matches(msg, m.keys.Top) {
			m.viewport.GotoTop()
		} else if key.Matches(msg, m.keys.Bottom) {
			m.viewport.GotoBottom()
		} else if key.Matches(msg, m.keys.Search) {
			m.searchMode = true
			m.searchTerm = ""
			return m, nil
		} else if key.Matches(msg, m.keys.NextMatch) {
			m.gotoNextSearchMatch()
		} else if key.Matches(msg, m.keys.PrevMatch) {
			m.gotoPrevSearchMatch()
		}

	case tea.WindowSizeMsg:
		headerHeight := 2
		footerHeight := 2
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Initialize viewport on first window size message
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// handleSearchInput processes input during search mode
func (m pagerModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Execute search
		m.searchMode = false
		if m.searchTerm != "" {
			m.performSearch()
			if len(m.searchLines) > 0 {
				m.searchIndex = 0
				m.gotoSearchLine(m.searchLines[0])
			}
		}
		return m, nil

	case tea.KeyEsc, tea.KeyCtrlC:
		// Cancel search
		m.searchMode = false
		m.searchTerm = ""
		return m, nil

	case tea.KeyBackspace:
		// Remove last character
		if len(m.searchTerm) > 0 {
			m.searchTerm = m.searchTerm[:len(m.searchTerm)-1]
		}
		return m, nil

	default:
		// Append character to search term
		if msg.Type == tea.KeyRunes {
			m.searchTerm += string(msg.Runes)
		}
		return m, nil
	}
}

// performSearch searches for the search term in the content
func (m *pagerModel) performSearch() {
	m.searchLines = []int{}
	m.searchIndex = -1

	if m.searchTerm == "" {
		return
	}

	lines := strings.Split(m.content, "\n")
	searchLower := strings.ToLower(m.searchTerm)

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), searchLower) {
			m.searchLines = append(m.searchLines, i)
		}
	}

	logsPagerLog.Printf("Search for '%s' found %d matches", m.searchTerm, len(m.searchLines))
}

// gotoNextSearchMatch navigates to the next search match
func (m *pagerModel) gotoNextSearchMatch() {
	if len(m.searchLines) == 0 {
		return
	}

	m.searchIndex = (m.searchIndex + 1) % len(m.searchLines)
	m.gotoSearchLine(m.searchLines[m.searchIndex])
}

// gotoPrevSearchMatch navigates to the previous search match
func (m *pagerModel) gotoPrevSearchMatch() {
	if len(m.searchLines) == 0 {
		return
	}

	m.searchIndex--
	if m.searchIndex < 0 {
		m.searchIndex = len(m.searchLines) - 1
	}
	m.gotoSearchLine(m.searchLines[m.searchIndex])
}

// gotoSearchLine scrolls the viewport to show the given line
func (m *pagerModel) gotoSearchLine(lineNum int) {
	// Calculate the Y offset for this line
	// Each line is 1 unit in the viewport
	m.viewport.SetYOffset(lineNum)
}

// View renders the pager UI
func (m pagerModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Show help overlay if requested
	if m.showHelp {
		return m.renderHelp()
	}

	// Render header
	header := m.renderHeader()

	// Render footer
	footer := m.renderFooter()

	// Combine header + viewport + footer
	return fmt.Sprintf("%s\n%s\n%s", header, m.viewport.View(), footer)
}

// renderHeader renders the top bar
func (m pagerModel) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.ColorInfo).
		Bold(true)

	title := "Workflow Logs Viewer"
	if m.searchMode {
		title = fmt.Sprintf("Search: %s_", m.searchTerm)
	} else if len(m.searchLines) > 0 {
		title = fmt.Sprintf("Workflow Logs Viewer - Match %d/%d", m.searchIndex+1, len(m.searchLines))
	}

	return titleStyle.Render(title)
}

// renderFooter renders the bottom status bar
func (m pagerModel) renderFooter() string {
	infoStyle := lipgloss.NewStyle().
		Foreground(styles.ColorComment)

	info := fmt.Sprintf("%.0f%% • Press ? for help • q to quit",
		m.viewport.ScrollPercent()*100)

	return infoStyle.Render(info)
}

// renderHelp renders the help overlay
func (m pagerModel) renderHelp() string {
	// Create a bordered box for the help content
	helpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorInfo).
		Padding(1, 2).
		Width(60)

	helpBox := helpStyle.Render(m.helpContent)

	// Center the help box in the viewport
	return lipgloss.Place(
		m.viewport.Width,
		m.viewport.Height,
		lipgloss.Center,
		lipgloss.Center,
		helpBox,
	)
}

// showInPager displays content in an interactive pager
// Returns error if pager cannot be started (e.g., not in TTY when mode is always)
func showInPager(content string, mode PagerMode) error {
	logsPagerLog.Printf("Showing content in pager: mode=%s, length=%d", mode, len(content))

	// Check TTY status
	isTTY := tty.IsStderrTerminal()

	// Determine if pager should be enabled
	shouldEnablePager := false
	switch mode {
	case PagerModeAlways:
		shouldEnablePager = true
		if !isTTY {
			return fmt.Errorf("pager mode 'always' requires a TTY environment")
		}
	case PagerModeNever:
		shouldEnablePager = false
	case PagerModeAuto:
		shouldEnablePager = isTTY
	}

	if !shouldEnablePager {
		// Fall back to standard output
		logsPagerLog.Print("Pager not enabled, using standard output")
		fmt.Print(content)
		return nil
	}

	// Create and run the pager
	model := newPagerModel(content)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run pager: %w", err)
	}

	return nil
}

// shouldAutoEnablePager determines if pager should be auto-enabled based on output size
// Pager is enabled for outputs with more than 100 runs or more than 1000 lines
func shouldAutoEnablePager(data LogsData) bool {
	// Enable pager for large number of runs
	if len(data.Runs) > 100 {
		return true
	}

	// Enable pager for large amount of content
	// (estimating ~20 lines per run minimum)
	estimatedLines := len(data.Runs) * 20
	if estimatedLines > 1000 {
		return true
	}

	return false
}
