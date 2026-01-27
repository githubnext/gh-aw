package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLibraryMountArgs(t *testing.T) {
	args := GetLibraryMountArgs()

	// Should return mount arguments for library directories
	require.NotEmpty(t, args, "Should return mount arguments")

	// Should return pairs of --mount and path
	require.Equal(t, 0, len(args)%2, "Arguments should come in pairs (--mount and path)")

	// Check that we have the expected library directory mounts
	argStr := strings.Join(args, " ")

	// Primary x86_64 library directory
	assert.Contains(t, argStr, "/usr/lib/x86_64-linux-gnu:/usr/lib/x86_64-linux-gnu:ro",
		"Should mount /usr/lib/x86_64-linux-gnu as read-only")

	// Alternative library location
	assert.Contains(t, argStr, "/lib/x86_64-linux-gnu:/lib/x86_64-linux-gnu:ro",
		"Should mount /lib/x86_64-linux-gnu as read-only")
}

func TestGetLibraryMountArgs_ReadOnlyAccess(t *testing.T) {
	args := GetLibraryMountArgs()

	// All library mounts should be read-only for security
	for i := 1; i < len(args); i += 2 {
		mountSpec := args[i]
		assert.True(t, strings.HasSuffix(mountSpec, ":ro"),
			"Library mount %s should be read-only", mountSpec)
	}
}

func TestGetBinaryMountArgs(t *testing.T) {
	args := GetBinaryMountArgs()

	// Should return mount arguments for binaries
	require.NotEmpty(t, args, "Should return mount arguments")

	// Should return pairs of --mount and path
	require.Equal(t, 0, len(args)%2, "Arguments should come in pairs (--mount and path)")

	// Check essential utilities are present
	argStr := strings.Join(args, " ")

	essentialUtils := []string{"cat", "curl", "date", "find", "gh", "grep", "jq", "yq"}
	for _, util := range essentialUtils {
		expected := "/usr/bin/" + util + ":/usr/bin/" + util + ":ro"
		assert.Contains(t, argStr, expected, "Should mount essential utility %s", util)
	}

	// Check common utilities are present
	commonUtils := []string{"cp", "cut", "diff", "head", "ls", "mkdir", "rm", "sed", "sort", "tail", "wc", "which"}
	for _, util := range commonUtils {
		expected := "/usr/bin/" + util + ":/usr/bin/" + util + ":ro"
		assert.Contains(t, argStr, expected, "Should mount common utility %s", util)
	}
}

func TestGetBinaryMountArgs_ReadOnlyAccess(t *testing.T) {
	args := GetBinaryMountArgs()

	// All binary mounts should be read-only for security
	for i := 1; i < len(args); i += 2 {
		mountSpec := args[i]
		assert.True(t, strings.HasSuffix(mountSpec, ":ro"),
			"Binary mount %s should be read-only", mountSpec)
	}
}

func TestGetAllUtilityMountArgs(t *testing.T) {
	args := GetAllUtilityMountArgs()

	// Should include both binary and library mounts
	binaryArgs := GetBinaryMountArgs()
	libraryArgs := GetLibraryMountArgs()

	expectedLen := len(binaryArgs) + len(libraryArgs)
	assert.Len(t, args, expectedLen,
		"Should combine binary and library mount arguments")

	// Verify binaries come first
	argStr := strings.Join(args, " ")
	binPos := strings.Index(argStr, "/usr/bin/cat")
	libPos := strings.Index(argStr, "/usr/lib/x86_64-linux-gnu")

	assert.Less(t, binPos, libPos,
		"Binary mounts should come before library mounts")
}

func TestGetAllUtilityMountArgs_AllReadOnly(t *testing.T) {
	args := GetAllUtilityMountArgs()

	// All mounts should be read-only for security
	for i := 1; i < len(args); i += 2 {
		mountSpec := args[i]
		assert.True(t, strings.HasSuffix(mountSpec, ":ro"),
			"Mount %s should be read-only", mountSpec)
	}
}

func TestLibraryMountArgs_FormatConsistency(t *testing.T) {
	args := GetLibraryMountArgs()

	// Every odd index should be "--mount"
	for i := 0; i < len(args); i += 2 {
		assert.Equal(t, "--mount", args[i],
			"Even index %d should be --mount flag", i)
	}

	// Every even index (mount spec) should follow the format "src:dst:mode"
	for i := 1; i < len(args); i += 2 {
		parts := strings.Split(args[i], ":")
		require.Len(t, parts, 3,
			"Mount spec at index %d should have 3 parts (src:dst:mode)", i)

		assert.True(t, strings.HasPrefix(parts[0], "/"),
			"Source path should be absolute")
		assert.True(t, strings.HasPrefix(parts[1], "/"),
			"Destination path should be absolute")
		assert.Equal(t, "ro", parts[2],
			"Mode should be 'ro' for read-only")
	}
}
