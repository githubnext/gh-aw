package workflow

// GitHubMCPFeatureFieldsParam is the feature flag name that enables the optional `fields`
// response-filtering parameter on selected GitHub MCP read tools. Introduced in v1.6.0.
// When enabled, tools such as list_pull_requests, search_code, and search_issues
// advertise a `fields` array parameter that lets agents restrict which fields are
// returned, reducing response size and context-window consumption.
//
// See: https://github.com/github/github-mcp-server/releases/tag/v1.6.0
const GitHubMCPFeatureFieldsParam = "fields_param"

// GitHubMCPDefaultFields maps each fields-enabled GitHub MCP tool to its recommended
// "slim" field set. These sets are sized to include only the fields that most
// workflows actually need, omitting the heaviest fields (e.g. full body text,
// reactions, nested repository objects) that dominate response size.
//
// Agents should pass one of these field lists when they do not need the full
// response from a list or search operation.
//
// Field guidance (per tool source in github/github-mcp-server v1.6.0):
//   - list_pull_requests / search_pull_requests: omit "body" (largest field)
//   - search_issues / list_issues:               omit "body", "reactions"
//   - search_code:                               omit "repository", "text_matches"
//   - list_commits:                              omit "parents", "stats", "files"
//   - get_file_contents (directory listing):     limit to "name", "type"
//   - list_releases:                             omit "body", "assets", "author"
var GitHubMCPDefaultFields = map[string][]string{
	"list_pull_requests": {
		"number",
		"title",
		"state",
		"draft",
		"created_at",
		"updated_at",
		"user",
		"base",
		"head",
		"labels",
	},
	"search_pull_requests": {
		"number",
		"title",
		"state",
		"draft",
		"created_at",
		"updated_at",
		"user",
		"labels",
	},
	"list_issues": {
		"number",
		"title",
		"state",
		"created_at",
		"updated_at",
		"user",
		"labels",
	},
	"search_issues": {
		"number",
		"title",
		"state",
		"created_at",
		"updated_at",
		"user",
		"labels",
		"assignees",
	},
	"search_code": {
		"name",
		"path",
		"sha",
	},
	"list_commits": {
		"sha",
		"html_url",
		"commit",
		"author",
	},
	"get_file_contents": {
		"name",
		"type",
	},
	"list_releases": {
		"id",
		"tag_name",
		"name",
		"draft",
		"prerelease",
		"published_at",
		"html_url",
	},
}
