//go:build !integration

package console

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewListItem(t *testing.T) {
	item := NewListItem("Title", "Description", "value")

	assert.Equal(t, "Title", item.title)
	assert.Equal(t, "Description", item.description)
}

func TestShowInteractiveList_EmptyItems(t *testing.T) {
	items := []ListItem{}
	_, err := ShowInteractiveList("Test", items)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no items to display")
}

// Note: Full interactive list testing requires TTY and cannot be automated.
// Manual testing should be performed to verify:
// - Arrow key navigation works
// - Selection with Enter key
// - Quit with Esc/Ctrl+C
// - Non-TTY fallback to text list
