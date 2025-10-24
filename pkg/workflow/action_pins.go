package workflow

// ActionPin represents a pinned GitHub Action with its commit SHA
type ActionPin struct {
	Repo    string // e.g., "actions/checkout"
	Version string // e.g., "v5"
	SHA     string // Full commit SHA
}

// actionPins maps action@version to their commit SHAs
// These SHAs should be updated periodically to use the latest stable versions
var actionPins = map[string]ActionPin{
	// Core actions
	"actions/checkout@v5": {
		Repo:    "actions/checkout",
		Version: "v5",
		SHA:     "08c6903cd8c0fde910a37f88322edcfb5dd907a8", // v5
	},
	"actions/github-script@v8": {
		Repo:    "actions/github-script",
		Version: "v8",
		SHA:     "ed597411d8f924073f98dfc5c65a23a2325f34cd", // v8
	},
	"actions/upload-artifact@v4": {
		Repo:    "actions/upload-artifact",
		Version: "v4",
		SHA:     "ea165f8d65b6e75b540449e92b4886f43607fa02", // v4
	},
	"actions/download-artifact@v5": {
		Repo:    "actions/download-artifact",
		Version: "v5",
		SHA:     "634f93cb2916e3fdff6788551b99b062d0335ce0", // v5
	},
	"actions/cache@v4": {
		Repo:    "actions/cache",
		Version: "v4",
		SHA:     "0057852bfaa89a56745cba8c7296529d2fc39830", // v4
	},

	// Setup actions
	"actions/setup-node@v4": {
		Repo:    "actions/setup-node",
		Version: "v4",
		SHA:     "49933ea5288caeca8642d1e84afbd3f7d6820020", // v4
	},
	"actions/setup-python@v5": {
		Repo:    "actions/setup-python",
		Version: "v5",
		SHA:     "a26af69be951a213d495a4c3e4e4022e16d87065", // v5
	},
	"actions/setup-go@v5": {
		Repo:    "actions/setup-go",
		Version: "v5",
		SHA:     "d35c59abb061a4a6fb18e82ac0862c26744d6ab5", // v5
	},
	"actions/setup-java@v4": {
		Repo:    "actions/setup-java",
		Version: "v4",
		SHA:     "c5195efecf7bdfc987ee8bae7a71cb8b11521c00", // v4
	},
	"actions/setup-dotnet@v4": {
		Repo:    "actions/setup-dotnet",
		Version: "v4",
		SHA:     "67a3573c9a986a3f9c594539f4ab511d57bb3ce9", // v4
	},

	// Third-party setup actions
	"erlef/setup-beam@v1": {
		Repo:    "erlef/setup-beam",
		Version: "v1",
		SHA:     "3559ac3b631a9560f28817e8e7fdde1638664336", // v1
	},
	"haskell-actions/setup@v2": {
		Repo:    "haskell-actions/setup",
		Version: "v2",
		SHA:     "d5d0f498b388e1a0eab1cd150202f664c5738e35", // v2
	},
	"ruby/setup-ruby@v1": {
		Repo:    "ruby/setup-ruby",
		Version: "v1",
		SHA:     "e5517072e87f198d9533967ae13d97c11b604005", // v1.99.0
	},
	"astral-sh/setup-uv@v5": {
		Repo:    "astral-sh/setup-uv",
		Version: "v5",
		SHA:     "e58605a9b6da7c637471fab8847a5e5a6b8df081", // v5
	},

	// GitHub Actions
	"github/codeql-action/upload-sarif@v3": {
		Repo:    "github/codeql-action/upload-sarif",
		Version: "v3",
		SHA:     "562257dc84ee23987d348302b161ee561898ec02", // v3
	},
}

// GetActionPin returns the pinned action reference for a given action@version
// If no pin is found, it returns the original action@version reference
func GetActionPin(actionRepo, version string) string {
	key := actionRepo + "@" + version
	if pin, exists := actionPins[key]; exists {
		return actionRepo + "@" + pin.SHA
	}
	// If no pin exists, return the version-based reference
	// This is a fallback for actions that haven't been pinned yet
	return actionRepo + "@" + version
}

// GetActionPinOrVersion is similar to GetActionPin but explicitly named to show fallback behavior
func GetActionPinOrVersion(actionRepo, version string) string {
	return GetActionPin(actionRepo, version)
}
