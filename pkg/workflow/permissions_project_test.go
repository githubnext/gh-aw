package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPermissionsContentsReadProjectsWrite tests that the project permissions helper
// includes the necessary permissions for project operations and campaign label management
func TestNewPermissionsContentsReadProjectsWrite(t *testing.T) {
	perms := NewPermissionsContentsReadProjectsWrite()

	require.NotNil(t, perms, "NewPermissionsContentsReadProjectsWrite() should not return nil")

	// Verify contents: read permission (required for basic access)
	contentsLevel, contentsExists := perms.Get(PermissionContents)
	assert.True(t, contentsExists, "contents permission should be set")
	assert.Equal(t, PermissionRead, contentsLevel, "contents should be read")

	// Verify organization-projects: write permission (required for project operations)
	projectsLevel, projectsExists := perms.Get(PermissionOrganizationProj)
	assert.True(t, projectsExists, "organization-projects permission should be set")
	assert.Equal(t, PermissionWrite, projectsLevel, "organization-projects should be write")

	// Verify issues: write permission (required for adding campaign labels to issues)
	issuesLevel, issuesExists := perms.Get(PermissionIssues)
	assert.True(t, issuesExists, "issues permission should be set")
	assert.Equal(t, PermissionWrite, issuesLevel, "issues should be write")

	// Verify no unexpected permissions are set
	assert.Len(t, perms.permissions, 3, "should have exactly 3 permissions")
}
