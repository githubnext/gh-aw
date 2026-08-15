package setupjs

import "embed"

// AgenticCommandsScripts contains the JavaScript modules used by the generated
// agentic commands router.
//
//go:embed route_slash_command.cjs add_reaction.cjs add_workflow_run_comment.cjs
//go:embed aw_context.cjs error_codes.cjs error_helpers.cjs experiment_helpers.cjs
//go:embed generate_footer.cjs github_api_helpers.cjs glob_pattern_helpers.cjs
//go:embed invocation_context_helpers.cjs markdown_code_region_balancer.cjs
//go:embed messages_core.cjs messages_run_status.cjs repo_helpers.cjs
//go:embed sanitize_content.cjs sanitize_content_core.cjs slash_command_matcher.cjs
//go:embed templatable.cjs threat_detection_warning.cjs workflow_metadata_helpers.cjs
var AgenticCommandsScripts embed.FS
