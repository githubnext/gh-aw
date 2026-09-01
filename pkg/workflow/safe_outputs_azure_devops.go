package workflow

import "github.com/github/gh-aw/pkg/logger"

var azureDevOpsSafeOutputsLog = logger.New("workflow:safe_outputs_azure_devops")

type AzureDevOpsArtifactLinkConfig struct {
	Enabled    bool   `yaml:"enabled,omitempty"`
	Repository string `yaml:"repository,omitempty"`
	Branch     string `yaml:"branch,omitempty"`
}

type CreateWorkItemConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	WorkItemType         string                        `yaml:"work-item-type,omitempty"`
	DescriptionField     string                        `yaml:"description-field,omitempty"`
	AreaPath             string                        `yaml:"area-path,omitempty"`
	IterationPath        string                        `yaml:"iteration-path,omitempty"`
	Assignee             string                        `yaml:"assignee,omitempty"`
	Tags                 []string                      `yaml:"tags,omitempty"`
	AllowedTags          []string                      `yaml:"allowed-tags,omitempty"`
	CustomFields         map[string]string             `yaml:"custom-fields,omitempty"`
	ArtifactLink         AzureDevOpsArtifactLinkConfig `yaml:"artifact-link,omitempty"`
	IncludeStats         bool                          `yaml:"include-stats,omitempty"`
}

type UpdateWorkItemConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Status               bool     `yaml:"status,omitempty"`
	Title                bool     `yaml:"title,omitempty"`
	Body                 bool     `yaml:"body,omitempty"`
	MarkdownBody         bool     `yaml:"markdown-body,omitempty"`
	TitlePrefix          string   `yaml:"title-prefix,omitempty"`
	TagPrefix            string   `yaml:"tag-prefix,omitempty"`
	Target               any      `yaml:"target,omitempty"`
	AreaPath             bool     `yaml:"area-path,omitempty"`
	IterationPath        bool     `yaml:"iteration-path,omitempty"`
	Assignee             bool     `yaml:"assignee,omitempty"`
	Tags                 bool     `yaml:"tags,omitempty"`
	AllowedTags          []string `yaml:"allowed-tags,omitempty"`
}

type CommentOnWorkItemConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Target               any  `yaml:"target,omitempty"`
	IncludeStats         bool `yaml:"include-stats,omitempty"`
}

type AssignWorkItemConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Target               any      `yaml:"target,omitempty"`
	Allowed              []string `yaml:"allowed,omitempty"`
	Blocked              []string `yaml:"blocked,omitempty"`
}

type LinkWorkItemsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Target               any      `yaml:"target,omitempty"`
	AllowedLinkTypes     []string `yaml:"allowed-link-types,omitempty"`
}

type UploadWorkItemAttachmentConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	MaxFileSize          int64    `yaml:"max-file-size,omitempty"`
	AllowedExtensions    []string `yaml:"allowed-extensions,omitempty"`
	CommentPrefix        string   `yaml:"comment-prefix,omitempty"`
}

func parseAzureDevOpsConfig[T any](c *Compiler, outputMap map[string]any, key string, defaultMax int, postProcess func(*T)) *T {
	config := parseConfigScaffold(outputMap, key, azureDevOpsSafeOutputsLog, func(err error) *T {
		azureDevOpsSafeOutputsLog.Printf("Failed to parse %s configuration: %v", key, err)
		return nil
	})
	if config == nil {
		return nil
	}
	if configMap, ok := outputMap[key].(map[string]any); ok {
		switch typed := any(config).(type) {
		case *CreateWorkItemConfig:
			c.parseBaseSafeOutputConfig(configMap, &typed.BaseSafeOutputConfig, defaultMax)
		case *UpdateWorkItemConfig:
			c.parseBaseSafeOutputConfig(configMap, &typed.BaseSafeOutputConfig, defaultMax)
		case *CommentOnWorkItemConfig:
			c.parseBaseSafeOutputConfig(configMap, &typed.BaseSafeOutputConfig, defaultMax)
		case *AssignWorkItemConfig:
			c.parseBaseSafeOutputConfig(configMap, &typed.BaseSafeOutputConfig, defaultMax)
		case *LinkWorkItemsConfig:
			c.parseBaseSafeOutputConfig(configMap, &typed.BaseSafeOutputConfig, defaultMax)
		case *UploadWorkItemAttachmentConfig:
			c.parseBaseSafeOutputConfig(configMap, &typed.BaseSafeOutputConfig, defaultMax)
		}
	}
	if postProcess != nil {
		postProcess(config)
	}
	return config
}

func (c *Compiler) parseCreateWorkItemConfig(outputMap map[string]any) *CreateWorkItemConfig {
	return parseAzureDevOpsConfig(c, outputMap, "create-work-item", 1, func(config *CreateWorkItemConfig) {
		if config.WorkItemType == "" {
			config.WorkItemType = "Task"
		}
		if config.ArtifactLink.Branch == "" {
			config.ArtifactLink.Branch = "main"
		}
	})
}

func (c *Compiler) parseUpdateWorkItemConfig(outputMap map[string]any) *UpdateWorkItemConfig {
	return parseAzureDevOpsConfig[UpdateWorkItemConfig](c, outputMap, "update-work-item", 1, nil)
}

func (c *Compiler) parseCommentOnWorkItemConfig(outputMap map[string]any) *CommentOnWorkItemConfig {
	return parseAzureDevOpsConfig[CommentOnWorkItemConfig](c, outputMap, "comment-on-work-item", 1, nil)
}

func (c *Compiler) parseAssignWorkItemConfig(outputMap map[string]any) *AssignWorkItemConfig {
	return parseAzureDevOpsConfig[AssignWorkItemConfig](c, outputMap, "assign-work-item", 1, nil)
}

func (c *Compiler) parseLinkWorkItemsConfig(outputMap map[string]any) *LinkWorkItemsConfig {
	return parseAzureDevOpsConfig[LinkWorkItemsConfig](c, outputMap, "link-work-items", 5, nil)
}

func (c *Compiler) parseUploadWorkItemAttachmentConfig(outputMap map[string]any) *UploadWorkItemAttachmentConfig {
	return parseAzureDevOpsConfig(c, outputMap, "upload-workitem-attachment", 1, func(config *UploadWorkItemAttachmentConfig) {
		if config.MaxFileSize == 0 {
			config.MaxFileSize = 5 * 1024 * 1024
		}
	})
}

func addAzureDevOpsTarget(builder *handlerConfigBuilder, target any) *handlerConfigBuilder {
	if target != nil {
		builder.AddDefault("target", target)
	}
	return builder
}

var azureDevOpsWorkItemHandlerRegistry = map[string]handlerBuilder{
	"create_work_item": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateWorkItems == nil {
			return nil
		}
		c := cfg.CreateWorkItems
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("work_item_type", c.WorkItemType).
			AddIfNotEmpty("description_field", c.DescriptionField).
			AddIfNotEmpty("area_path", c.AreaPath).
			AddIfNotEmpty("iteration_path", c.IterationPath).
			AddIfNotEmpty("assignee", c.Assignee).
			AddStringSlice("tags", c.Tags).
			AddStringSlice("allowed_tags", c.AllowedTags).
			AddDefault("custom_fields", c.CustomFields).
			AddDefault("artifact_link", c.ArtifactLink).
			AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
			Build()
	},
	"update_work_item": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UpdateWorkItems == nil {
			return nil
		}
		c := cfg.UpdateWorkItems
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfTrue("status", c.Status).
			AddIfTrue("title", c.Title).
			AddIfTrue("body", c.Body).
			AddIfTrue("markdown_body", c.MarkdownBody).
			AddIfNotEmpty("title_prefix", c.TitlePrefix).
			AddIfNotEmpty("tag_prefix", c.TagPrefix).
			AddIfTrue("area_path", c.AreaPath).
			AddIfTrue("iteration_path", c.IterationPath).
			AddIfTrue("assignee", c.Assignee).
			AddIfTrue("tags", c.Tags).
			AddStringSlice("allowed_tags", c.AllowedTags).
			AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged))
		return addAzureDevOpsTarget(builder, c.Target).Build()
	},
	"comment_on_work_item": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CommentOnWorkItems == nil {
			return nil
		}
		c := cfg.CommentOnWorkItems
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged))
		return addAzureDevOpsTarget(builder, c.Target).Build()
	},
	"assign_work_item": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AssignWorkItems == nil {
			return nil
		}
		c := cfg.AssignWorkItems
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Allowed).
			AddStringSlice("blocked", c.Blocked).
			AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged))
		return addAzureDevOpsTarget(builder, c.Target).Build()
	},
	"link_work_items": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.LinkWorkItems == nil {
			return nil
		}
		c := cfg.LinkWorkItems
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed_link_types", c.AllowedLinkTypes).
			AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged))
		return addAzureDevOpsTarget(builder, c.Target).Build()
	},
	"upload_workitem_attachment": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UploadWorkItemAttachments == nil {
			return nil
		}
		c := cfg.UploadWorkItemAttachments
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddDefault("max_file_size", c.MaxFileSize).
			AddStringSlice("allowed_extensions", c.AllowedExtensions).
			AddIfNotEmpty("comment_prefix", c.CommentPrefix).
			AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
			Build()
	},
}
