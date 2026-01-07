package workflow

import (
	"reflect"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var safeOutputConfigBuilderLog = logger.New("workflow:safe_output_config_builder")

// fieldMapping defines how struct field names map to handler config keys
var fieldMapping = map[string]string{
	"Max":                      "max",
	"GitHubToken":              "github-token",
	"TitlePrefix":              "title_prefix",
	"Labels":                   "labels",
	"AllowedLabels":            "allowed_labels",
	"Assignees":                "assignees",
	"TargetRepoSlug":           "target-repo",
	"AllowedRepos":             "allowed_repos",
	"Expires":                  "expires",
	"Target":                   "target",
	"HideOlderComments":        "hide_older_comments",
	"Category":                 "category",
	"CloseOlderDiscussions":    "close_older_discussions",
	"RequiredLabels":           "required_labels",
	"RequiredTitlePrefix":      "required_title_prefix",
	"RequiredCategory":         "required_category",
	"Allowed":                  "allowed",
	"Status":                   "allow_status",
	"Title":                    "allow_title",
	"Body":                     "allow_body",
	"ParentRequiredLabels":     "parent_required_labels",
	"ParentTitlePrefix":        "parent_title_prefix",
	"SubRequiredLabels":        "sub_required_labels",
	"SubTitlePrefix":           "sub_title_prefix",
	"Side":                     "side",
	"Draft":                    "draft",
	"IfNoChanges":              "if_no_changes",
	"AllowEmpty":               "allow_empty",
	"CommitTitleSuffix":        "commit_title_suffix",
	"AllowedReasons":           "allowed_reasons",
	"Workflows":                "workflows",
	"Reviewers":                "reviewers",
	"DefaultAgent":             "default_agent",
	"Discussion":               "discussion",
}

// buildHandlerConfig builds a handler configuration map from a config struct using reflection.
// It automatically handles common patterns like:
// - Skipping zero values (empty strings, 0, nil slices, nil pointers)
// - Converting boolean pointers to their presence (for allow_* fields)
// - Mapping Go struct field names to handler config keys
// - Processing embedded structs recursively
func buildHandlerConfig(configStruct any) map[string]any {
	handlerConfig := make(map[string]any)

	v := reflect.ValueOf(configStruct)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return handlerConfig
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		safeOutputConfigBuilderLog.Printf("Expected struct, got %v", v.Kind())
		return handlerConfig
	}

	buildHandlerConfigRecursive(v, handlerConfig)
	return handlerConfig
}

// buildHandlerConfigRecursive processes struct fields recursively, handling embedded structs
func buildHandlerConfigRecursive(v reflect.Value, handlerConfig map[string]any) {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}

		// Handle embedded structs by recursing into them
		if fieldType.Anonymous {
			if field.Kind() == reflect.Struct {
				buildHandlerConfigRecursive(field, handlerConfig)
			}
			continue
		}

		// Get the field name and look up the mapping
		fieldName := fieldType.Name
		configKey, exists := fieldMapping[fieldName]
		if !exists {
			// If no mapping exists, skip this field
			continue
		}

		// Handle different field types
		switch field.Kind() {
		case reflect.String:
			str := field.String()
			if str != "" {
				handlerConfig[configKey] = str
			}

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal := field.Int()
			if intVal > 0 {
				handlerConfig[configKey] = intVal
			}

		case reflect.Bool:
			boolVal := field.Bool()
			if boolVal {
				handlerConfig[configKey] = true
			}

		case reflect.Slice:
			if !field.IsNil() && field.Len() > 0 {
				// Convert slice to []any for JSON marshaling
				slice := make([]any, field.Len())
				for j := 0; j < field.Len(); j++ {
					slice[j] = field.Index(j).Interface()
				}
				handlerConfig[configKey] = slice
			}

		case reflect.Ptr:
			if !field.IsNil() {
				// For pointer fields, check the type they point to
				elem := field.Elem()
				switch elem.Kind() {
				case reflect.Bool:
					// Boolean pointers indicate "allow_*" fields
					// For these, we just set the key to true if the pointer exists
					handlerConfig[configKey] = true
				default:
					// For other pointer types, use their underlying value
					handlerConfig[configKey] = elem.Interface()
				}
			}

		default:
			safeOutputConfigBuilderLog.Printf("Unhandled field type %v for field %s", field.Kind(), fieldName)
		}
	}
}

// safeOutputRegistry maps safe output types to their handler names
type safeOutputHandler struct {
	fieldName   string
	handlerName string
	customizer  func(config any, handlerConfig map[string]any) // Optional customizer for special cases
}

var safeOutputHandlers = []safeOutputHandler{
	{fieldName: "CreateIssues", handlerName: "create_issue"},
	{fieldName: "AddComments", handlerName: "add_comment"},
	{fieldName: "CreateDiscussions", handlerName: "create_discussion"},
	{fieldName: "CloseIssues", handlerName: "close_issue"},
	{fieldName: "CloseDiscussions", handlerName: "close_discussion"},
	{fieldName: "AddLabels", handlerName: "add_labels"},
	{fieldName: "UpdateIssues", handlerName: "update_issue"},
	{fieldName: "UpdateDiscussions", handlerName: "update_discussion"},
	{fieldName: "LinkSubIssue", handlerName: "link_sub_issue"},
	{fieldName: "UpdateRelease", handlerName: "update_release"},
	{fieldName: "CreatePullRequestReviewComments", handlerName: "create_pull_request_review_comment"},
	{
		fieldName:   "CreatePullRequests",
		handlerName: "create_pull_request",
		customizer: func(config any, handlerConfig map[string]any) {
			// Add base branch for git operations
			handlerConfig["base_branch"] = "${{ github.ref_name }}"
		},
	},
	{
		fieldName:   "PushToPullRequestBranch",
		handlerName: "push_to_pull_request_branch",
		customizer: func(config any, handlerConfig map[string]any) {
			// Add base branch for git operations
			handlerConfig["base_branch"] = "${{ github.ref_name }}"
		},
	},
	{fieldName: "UpdatePullRequests", handlerName: "update_pull_request", customizer: updatePullRequestCustomizer},
	{fieldName: "ClosePullRequests", handlerName: "close_pull_request"},
	{fieldName: "HideComment", handlerName: "hide_comment"},
	{fieldName: "DispatchWorkflow", handlerName: "dispatch_workflow"},
	{fieldName: "CreateProjectStatusUpdates", handlerName: "create_project_status_update"},
}

// updatePullRequestCustomizer handles special defaulting logic for update_pull_request
func updatePullRequestCustomizer(config any, handlerConfig map[string]any) {
	// For update_pull_request, default to true if not specified
	if _, exists := handlerConfig["allow_title"]; !exists {
		handlerConfig["allow_title"] = true
	}
	if _, exists := handlerConfig["allow_body"]; !exists {
		handlerConfig["allow_body"] = true
	}
}

// buildSafeOutputConfigs builds handler configuration for all safe output types using reflection.
// This replaces the massive duplication in addHandlerManagerConfigEnvVar.
func buildSafeOutputConfigs(safeOutputs *SafeOutputsConfig) map[string]map[string]any {
	config := make(map[string]map[string]any)

	if safeOutputs == nil {
		return config
	}

	v := reflect.ValueOf(safeOutputs).Elem()

	// Process each registered handler
	for _, handler := range safeOutputHandlers {
		field := v.FieldByName(handler.fieldName)
		if !field.IsValid() || field.IsNil() {
			continue
		}

		// Build the handler config using reflection
		handlerConfig := buildHandlerConfig(field.Interface())

		// Apply any custom logic
		if handler.customizer != nil {
			handler.customizer(field.Interface(), handlerConfig)
		}

		// Add even if empty - the presence of the config pointer indicates the handler is enabled
		config[handler.handlerName] = handlerConfig
	}

	// Handle max_patch_size for PR-related handlers
	if safeOutputs.MaximumPatchSize > 0 {
		for _, handlerName := range []string{"create_pull_request", "push_to_pull_request_branch"} {
			if cfg, exists := config[handlerName]; exists {
				cfg["max_patch_size"] = safeOutputs.MaximumPatchSize
			}
		}
	} else if safeOutputs.CreatePullRequests != nil || safeOutputs.PushToPullRequestBranch != nil {
		// Add default max_patch_size
		defaultSize := 1024
		for _, handlerName := range []string{"create_pull_request", "push_to_pull_request_branch"} {
			if cfg, exists := config[handlerName]; exists {
				cfg["max_patch_size"] = defaultSize
			}
		}
	}

	return config
}
