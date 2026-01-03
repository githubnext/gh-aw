package cli

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPagerModel(t *testing.T) {
	content := "Test content\nLine 2\nLine 3"
	model := newPagerModel(content)

	assert.Equal(t, content, model.content, "Content should be set")
	assert.False(t, model.ready, "Model should not be ready initially")
	assert.False(t, model.showHelp, "Help should not be shown initially")
	assert.False(t, model.searchMode, "Search mode should be off initially")
	assert.Equal(t, "", model.searchTerm, "Search term should be empty initially")
	assert.Equal(t, -1, model.searchIndex, "Search index should be -1 initially")
	assert.NotEmpty(t, model.helpContent, "Help content should be generated")
}

func TestPagerModelInit(t *testing.T) {
	model := newPagerModel("test")
	cmd := model.Init()
	assert.Nil(t, cmd, "Init should return nil command")
}

func TestPagerModelWindowSizeUpdate(t *testing.T) {
	model := newPagerModel("test content")

	// Send window size message
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(pagerModel)

	assert.True(t, model.ready, "Model should be ready after window size message")
	assert.Equal(t, 80, model.viewport.Width, "Viewport width should match window width")
	// Height should be window height minus header and footer (2 + 2 = 4)
	assert.Equal(t, 20, model.viewport.Height, "Viewport height should be window height minus margins")
}

func TestPagerModelQuitKeys(t *testing.T) {
	model := newPagerModel("test")
	model.ready = true

	quitKeys := []string{"q", "esc", "ctrl+c"}

	for _, keyStr := range quitKeys {
		t.Run(keyStr, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(keyStr)}
			if keyStr == "esc" {
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			} else if keyStr == "ctrl+c" {
				msg = tea.KeyMsg{Type: tea.KeyCtrlC}
			}

			_, cmd := model.Update(msg)
			assert.NotNil(t, cmd, "Quit keys should return a command")
		})
	}
}

func TestPagerModelHelpToggle(t *testing.T) {
	model := newPagerModel("test")
	model.ready = true

	// Toggle help on with '?'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(pagerModel)
	assert.True(t, model.showHelp, "Help should be shown after pressing ?")

	// Toggle help off with 'h'
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")}
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(pagerModel)
	assert.False(t, model.showHelp, "Help should be hidden after pressing h")
}

func TestPagerModelSearchMode(t *testing.T) {
	model := newPagerModel("test content\nwith multiple lines\ncontaining test")
	model.ready = true

	// Enter search mode with '/'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(pagerModel)
	assert.True(t, model.searchMode, "Search mode should be active after pressing /")

	// Type search term
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")}
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(pagerModel)
	assert.Equal(t, "t", model.searchTerm, "Search term should be updated")

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")}
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(pagerModel)
	assert.Equal(t, "te", model.searchTerm, "Search term should be updated")

	// Test backspace
	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(pagerModel)
	assert.Equal(t, "t", model.searchTerm, "Backspace should remove last character")

	// Complete search term
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("est")}
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(pagerModel)

	// Execute search with Enter
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(pagerModel)
	assert.False(t, model.searchMode, "Search mode should be off after pressing Enter")
	assert.Greater(t, len(model.searchLines), 0, "Search should find matches")
}

func TestPagerModelSearchCancel(t *testing.T) {
	model := newPagerModel("test content")
	model.ready = true

	// Enter search mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(pagerModel)

	// Type search term
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test")}
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(pagerModel)

	// Cancel with Esc
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(pagerModel)

	assert.False(t, model.searchMode, "Search mode should be off after Esc")
	assert.Equal(t, "", model.searchTerm, "Search term should be cleared after canceling")
}

func TestPerformSearch(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		searchTerm      string
		expectedMatches int
	}{
		{
			name:            "single match",
			content:         "line 1\nline 2 with test\nline 3",
			searchTerm:      "test",
			expectedMatches: 1,
		},
		{
			name:            "multiple matches",
			content:         "test line 1\nline 2\ntest line 3",
			searchTerm:      "test",
			expectedMatches: 2,
		},
		{
			name:            "case insensitive",
			content:         "TEST line 1\nline 2\nTest line 3",
			searchTerm:      "test",
			expectedMatches: 2,
		},
		{
			name:            "no matches",
			content:         "line 1\nline 2\nline 3",
			searchTerm:      "nomatch",
			expectedMatches: 0,
		},
		{
			name:            "empty search term",
			content:         "line 1\nline 2",
			searchTerm:      "",
			expectedMatches: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newPagerModel(tt.content)
			model.searchTerm = tt.searchTerm
			model.performSearch()

			assert.Equal(t, tt.expectedMatches, len(model.searchLines),
				"Expected %d matches for search term '%s'", tt.expectedMatches, tt.searchTerm)
		})
	}
}

func TestSearchNavigation(t *testing.T) {
	content := "line 1 test\nline 2\nline 3 test\nline 4\nline 5 test"
	model := newPagerModel(content)
	model.searchTerm = "test"
	model.performSearch()

	require.Equal(t, 3, len(model.searchLines), "Should find 3 matches")
	require.Equal(t, -1, model.searchIndex, "Initial search index should be -1")

	// Navigate to next match
	model.gotoNextSearchMatch()
	assert.Equal(t, 0, model.searchIndex, "Should be at first match")

	model.gotoNextSearchMatch()
	assert.Equal(t, 1, model.searchIndex, "Should be at second match")

	model.gotoNextSearchMatch()
	assert.Equal(t, 2, model.searchIndex, "Should be at third match")

	// Wrap around to first
	model.gotoNextSearchMatch()
	assert.Equal(t, 0, model.searchIndex, "Should wrap to first match")

	// Navigate backwards
	model.gotoPrevSearchMatch()
	assert.Equal(t, 2, model.searchIndex, "Should go to last match")

	model.gotoPrevSearchMatch()
	assert.Equal(t, 1, model.searchIndex, "Should go to second match")
}

func TestPagerModeString(t *testing.T) {
	assert.Equal(t, "auto", string(PagerModeAuto))
	assert.Equal(t, "always", string(PagerModeAlways))
	assert.Equal(t, "never", string(PagerModeNever))
}

func TestShouldAutoEnablePager(t *testing.T) {
	tests := []struct {
		name           string
		numRuns        int
		expectedEnable bool
	}{
		{
			name:           "small output - no pager",
			numRuns:        10,
			expectedEnable: false,
		},
		{
			name:           "medium output - no pager",
			numRuns:        50,
			expectedEnable: false,
		},
		{
			name:           "large output - enable pager",
			numRuns:        101,
			expectedEnable: true,
		},
		{
			name:           "very large output - enable pager",
			numRuns:        200,
			expectedEnable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := LogsData{
				Runs: make([]RunData, tt.numRuns),
			}

			result := shouldAutoEnablePager(data)
			assert.Equal(t, tt.expectedEnable, result,
				"shouldAutoEnablePager with %d runs should return %v", tt.numRuns, tt.expectedEnable)
		})
	}
}

func TestPagerModelView(t *testing.T) {
	model := newPagerModel("test content\nline 2\nline 3")

	// Before ready
	view := model.View()
	assert.Contains(t, view, "Initializing", "View should show initializing message before ready")

	// After window size (ready state)
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(pagerModel)

	view = model.View()
	assert.NotEmpty(t, view, "View should not be empty when ready")
	assert.Contains(t, view, "Workflow Logs Viewer", "View should contain title")

	// With help shown
	model.showHelp = true
	view = model.View()
	assert.Contains(t, view, "Keyboard Shortcuts", "Help view should contain shortcuts title")
}

func TestBuildHelpContent(t *testing.T) {
	keys := defaultKeyMap()
	help := buildHelpContent(keys)

	assert.NotEmpty(t, help, "Help content should not be empty")
	assert.Contains(t, help, "Keyboard Shortcuts", "Help should contain title")
	assert.Contains(t, help, "up", "Help should contain up key")
	assert.Contains(t, help, "down", "Help should contain down key")
	assert.Contains(t, help, "search", "Help should contain search key")
	assert.Contains(t, help, "quit", "Help should contain quit key")
}

func TestPagerRenderHeader(t *testing.T) {
	model := newPagerModel("test")
	model.ready = true

	// Normal mode
	header := model.renderHeader()
	assert.Contains(t, header, "Workflow Logs Viewer", "Header should contain title")

	// Search mode
	model.searchMode = true
	model.searchTerm = "test"
	header = model.renderHeader()
	assert.Contains(t, header, "Search:", "Header should show search prompt in search mode")
	assert.Contains(t, header, "test", "Header should show search term")

	// With search results
	model.searchMode = false
	model.searchLines = []int{1, 3, 5}
	model.searchIndex = 1
	header = model.renderHeader()
	assert.Contains(t, header, "Match 2/3", "Header should show match count")
}

func TestPagerRenderFooter(t *testing.T) {
	model := newPagerModel("test")
	model.ready = true

	// Initialize viewport
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(pagerModel)

	footer := model.renderFooter()
	assert.NotEmpty(t, footer, "Footer should not be empty")
	assert.Contains(t, footer, "%", "Footer should show percentage")
	assert.Contains(t, footer, "help", "Footer should mention help")
	assert.Contains(t, footer, "quit", "Footer should mention quit")
}

func TestDefaultKeyMap(t *testing.T) {
	keys := defaultKeyMap()

	// Verify all key bindings are defined
	assert.NotEmpty(t, keys.Up.Keys(), "Up key should be defined")
	assert.NotEmpty(t, keys.Down.Keys(), "Down key should be defined")
	assert.NotEmpty(t, keys.PageUp.Keys(), "PageUp key should be defined")
	assert.NotEmpty(t, keys.PageDown.Keys(), "PageDown key should be defined")
	assert.NotEmpty(t, keys.HalfUp.Keys(), "HalfUp key should be defined")
	assert.NotEmpty(t, keys.HalfDown.Keys(), "HalfDown key should be defined")
	assert.NotEmpty(t, keys.Top.Keys(), "Top key should be defined")
	assert.NotEmpty(t, keys.Bottom.Keys(), "Bottom key should be defined")
	assert.NotEmpty(t, keys.Search.Keys(), "Search key should be defined")
	assert.NotEmpty(t, keys.NextMatch.Keys(), "NextMatch key should be defined")
	assert.NotEmpty(t, keys.PrevMatch.Keys(), "PrevMatch key should be defined")
	assert.NotEmpty(t, keys.Help.Keys(), "Help key should be defined")
	assert.NotEmpty(t, keys.Quit.Keys(), "Quit key should be defined")

	// Verify some specific key mappings
	assert.Contains(t, keys.Up.Keys(), "k", "Up should include 'k' key")
	assert.Contains(t, keys.Down.Keys(), "j", "Down should include 'j' key")
	assert.Contains(t, keys.Search.Keys(), "/", "Search should include '/' key")
	assert.Contains(t, keys.Quit.Keys(), "q", "Quit should include 'q' key")
}

func TestPagerModelNavigationKeys(t *testing.T) {
	// Create content with many lines to ensure scrolling is possible
	model := newPagerModel(strings.Repeat("line\n", 200))
	model.ready = true

	// Initialize viewport with a small height to ensure content is scrollable
	msg := tea.WindowSizeMsg{Width: 80, Height: 10}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(pagerModel)

	initialYOffset := model.viewport.YOffset

	// Test down navigation
	msg2 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	updatedModel, _ = model.Update(msg2)
	model = updatedModel.(pagerModel)
	// Viewport should have moved down (increased Y offset)
	// Note: Exact value depends on viewport implementation

	// Test up navigation
	msg3 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	updatedModel, _ = model.Update(msg3)
	model = updatedModel.(pagerModel)
	// Should move back up

	// Test goto top
	msg4 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")}
	updatedModel, _ = model.Update(msg4)
	model = updatedModel.(pagerModel)
	assert.Equal(t, 0, model.viewport.YOffset, "Goto top should set Y offset to 0")

	// Test goto bottom - navigate down first to have content to scroll
	for i := 0; i < 5; i++ {
		msgDown := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
		updatedModel, _ = model.Update(msgDown)
		model = updatedModel.(pagerModel)
	}

	beforeBottomOffset := model.viewport.YOffset
	msg5 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")}
	updatedModel, _ = model.Update(msg5)
	model = updatedModel.(pagerModel)

	// After goto bottom, should be at a higher Y offset (scrolled down more)
	// The actual value depends on content size and viewport height
	assert.GreaterOrEqual(t, model.viewport.YOffset, beforeBottomOffset,
		"Goto bottom should move to end or stay at same position if already at bottom")

	// Verify initialYOffset was 0 (sanity check)
	assert.Equal(t, 0, initialYOffset, "Initial Y offset should be 0")
}
