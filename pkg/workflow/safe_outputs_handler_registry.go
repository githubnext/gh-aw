package workflow

import "maps"

var handlerRegistryCategoryBuilders = []func() map[string]handlerBuilder{
	buildIssueAndDiscussionHandlerRegistry,
	buildPullRequestHandlerRegistry,
	buildRepositoryAutomationHandlerRegistry,
	buildWorkflowDispatchAndReportingHandlerRegistry,
	buildArtifactAndProjectHandlerRegistry,
}

// handlerRegistry maps handler names to their builder functions.
// Each entry is keyed by the handler name used in GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG
// and returns a config map (nil means the handler is disabled).
var handlerRegistry = buildHandlerRegistry()

func buildHandlerRegistry() map[string]handlerBuilder {
	registry := make(map[string]handlerBuilder)
	for _, buildCategory := range handlerRegistryCategoryBuilders {
		maps.Copy(registry, buildCategory())
	}
	return registry
}
