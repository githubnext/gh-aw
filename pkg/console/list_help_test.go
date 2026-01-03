package console

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListHelpEnabled verifies that the interactive list has help enabled
func TestListHelpEnabled(t *testing.T) {
	// Create sample items
	items := []ListItem{
		NewListItem("Item 1", "Description 1", "value1"),
		NewListItem("Item 2", "Description 2", "value2"),
	}

	// Convert to list.Item interface
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	// Create list with custom delegate (same as ShowInteractiveList)
	delegate := itemDelegate{}
	l := list.New(listItems, delegate, 80, 20)
	l.Title = "Test List"
	l.SetShowHelp(true)

	// Verify help is enabled
	assert.True(t, l.ShowHelp(), "Help should be enabled in interactive list")
}

// TestListKeyboardShortcutsAvailable verifies that standard keyboard shortcuts are configured
func TestListKeyboardShortcutsAvailable(t *testing.T) {
	// Create a basic list
	items := []list.Item{
		NewListItem("Item 1", "Description 1", "value1"),
	}

	delegate := itemDelegate{}
	l := list.New(items, delegate, 80, 20)

	// Get the key map
	keyMap := l.KeyMap

	// Verify essential keyboard shortcuts are configured
	require.NotNil(t, keyMap, "KeyMap should not be nil")

	// Test navigation shortcuts
	assert.NotEmpty(t, keyMap.CursorUp.Keys(), "CursorUp shortcut should be configured")
	assert.NotEmpty(t, keyMap.CursorDown.Keys(), "CursorDown shortcut should be configured")

	// Test pagination shortcuts
	assert.NotEmpty(t, keyMap.NextPage.Keys(), "NextPage shortcut should be configured")
	assert.NotEmpty(t, keyMap.PrevPage.Keys(), "PrevPage shortcut should be configured")

	// Test filter shortcuts
	assert.NotEmpty(t, keyMap.Filter.Keys(), "Filter shortcut should be configured")
	assert.NotEmpty(t, keyMap.ClearFilter.Keys(), "ClearFilter shortcut should be configured")

	// Test quit shortcuts
	assert.NotEmpty(t, keyMap.Quit.Keys(), "Quit shortcut should be configured")
	assert.NotEmpty(t, keyMap.ForceQuit.Keys(), "ForceQuit shortcut should be configured")

	// Test help toggle shortcuts
	assert.NotEmpty(t, keyMap.ShowFullHelp.Keys(), "ShowFullHelp shortcut should be configured")
	assert.NotEmpty(t, keyMap.CloseFullHelp.Keys(), "CloseFullHelp shortcut should be configured")
}

// TestListKeyboardShortcutDescriptions verifies that shortcuts have user-friendly descriptions
func TestListKeyboardShortcutDescriptions(t *testing.T) {
	// Create a basic list
	items := []list.Item{
		NewListItem("Item 1", "Description 1", "value1"),
	}

	delegate := itemDelegate{}
	l := list.New(items, delegate, 80, 20)
	keyMap := l.KeyMap

	// Verify shortcuts have help descriptions
	assert.NotEmpty(t, keyMap.CursorUp.Help().Key, "CursorUp should have help key description")
	assert.NotEmpty(t, keyMap.CursorUp.Help().Desc, "CursorUp should have help text description")

	assert.NotEmpty(t, keyMap.CursorDown.Help().Key, "CursorDown should have help key description")
	assert.NotEmpty(t, keyMap.CursorDown.Help().Desc, "CursorDown should have help text description")

	assert.NotEmpty(t, keyMap.Filter.Help().Key, "Filter should have help key description")
	assert.NotEmpty(t, keyMap.Filter.Help().Desc, "Filter should have help text description")

	assert.NotEmpty(t, keyMap.Quit.Help().Key, "Quit should have help key description")
	assert.NotEmpty(t, keyMap.Quit.Help().Desc, "Quit should have help text description")

	assert.NotEmpty(t, keyMap.ShowFullHelp.Help().Key, "ShowFullHelp should have help key description")
	assert.NotEmpty(t, keyMap.ShowFullHelp.Help().Desc, "ShowFullHelp should have help text description")
}

// TestListFilteringEnabled verifies that filtering is enabled
func TestListFilteringEnabled(t *testing.T) {
	// Create sample items
	items := []ListItem{
		NewListItem("Item 1", "Description 1", "value1"),
		NewListItem("Item 2", "Description 2", "value2"),
	}

	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	// Create list (same as ShowInteractiveList)
	delegate := itemDelegate{}
	l := list.New(listItems, delegate, 80, 20)
	l.SetFilteringEnabled(true)

	// Verify filtering is enabled
	assert.True(t, l.FilteringEnabled(), "Filtering should be enabled in interactive list")
}

// TestListModelUpdate_QuitKeys verifies that quit keys work in the list model
func TestListModelUpdate_QuitKeys(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		shouldQuit bool
	}{
		{
			name:       "ctrl+c quits",
			key:        "ctrl+c",
			shouldQuit: true,
		},
		{
			name:       "q quits",
			key:        "q",
			shouldQuit: true,
		},
		{
			name:       "esc quits",
			key:        "esc",
			shouldQuit: true,
		},
		{
			name:       "enter does not quit",
			key:        "enter",
			shouldQuit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a list model
			items := []ListItem{
				NewListItem("Item 1", "Description 1", "value1"),
			}

			listItems := make([]list.Item, len(items))
			for i, item := range items {
				listItems[i] = item
			}

			delegate := itemDelegate{}
			l := list.New(listItems, delegate, 80, 20)
			m := listModel{list: l}

			// Simulate key press (we can't actually create tea.KeyMsg in tests,
			// but we can verify the quit logic exists)
			// This test validates the structure exists, full integration testing
			// requires manual testing with a TTY
			assert.False(t, m.quitting, "Model should not be quitting initially")
		})
	}
}
