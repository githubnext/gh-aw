//go:build !integration

package console

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewListItem(t *testing.T) {
	item := NewListItem("Title", "Description", "value")

	assert.Equal(t, "Title", item.Title())
	assert.Equal(t, "Description", item.Description())
}

func TestListItem_Title(t *testing.T) {
	item := ListItem{title: "My Title", description: "desc", value: "val"}
	assert.Equal(t, "My Title", item.Title())
}

func TestListItem_Description(t *testing.T) {
	item := ListItem{title: "title", description: "My Description", value: "val"}
	assert.Equal(t, "My Description", item.Description())
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
