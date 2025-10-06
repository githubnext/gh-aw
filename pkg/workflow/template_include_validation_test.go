package workflow

import (
	"strings"
	"testing"
)

func TestValidateNoIncludesInTemplateRegions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid - include outside template region",
			input: `# Test Workflow

@include shared/tools.md

{{#if github.event.issue.number}}
This is inside a template.
{{/if}}`,
			wantErr: false,
		},
		{
			name: "invalid - include inside template region",
			input: `# Test Workflow

{{#if github.event.issue.number}}
@include shared/tools.md
Some content here.
{{/if}}`,
			wantErr: true,
			errMsg:  "@include/@import directives cannot be used inside template regions",
		},
		{
			name: "invalid - import inside template region",
			input: `# Test Workflow

{{#if github.actor}}
@import shared/config.md
{{/if}}`,
			wantErr: true,
			errMsg:  "@include/@import directives cannot be used inside template regions",
		},
		{
			name: "invalid - optional include inside template region",
			input: `# Test Workflow

{{#if github.repository}}
@include? shared/optional.md
{{/if}}`,
			wantErr: true,
			errMsg:  "@include/@import directives cannot be used inside template regions",
		},
		{
			name: "valid - multiple includes outside templates",
			input: `# Test Workflow

@include shared/tools.md
@import shared/config.md

{{#if github.event.issue.number}}
Content here.
{{/if}}

@include shared/footer.md`,
			wantErr: false,
		},
		{
			name: "valid - no templates, only includes",
			input: `# Test Workflow

@include shared/tools.md
@import shared/config.md

Regular content without templates.`,
			wantErr: false,
		},
		{
			name: "valid - no includes, only templates",
			input: `# Test Workflow

{{#if github.event.issue.number}}
Content inside template.
{{/if}}

Regular content outside template.`,
			wantErr: false,
		},
		{
			name: "invalid - multiple templates with include in one",
			input: `# Test Workflow

{{#if github.event.issue.number}}
First template - no include.
{{/if}}

{{#if github.actor}}
@include shared/tools.md
Second template - has include.
{{/if}}`,
			wantErr: true,
			errMsg:  "@include/@import directives cannot be used inside template regions",
		},
		{
			name: "valid - nested content but include outside",
			input: `# Test Workflow

@include shared/header.md

{{#if github.event.issue.number}}
Some content.
{{/if}}

@include shared/footer.md`,
			wantErr: false,
		},
		{
			name: "invalid - include with section reference inside template",
			input: `# Test Workflow

{{#if github.event.pull_request.number}}
@include shared/tools.md#Security
{{/if}}`,
			wantErr: true,
			errMsg:  "@include/@import directives cannot be used inside template regions",
		},
		{
			name: "valid - include with section reference outside template",
			input: `# Test Workflow

@include shared/tools.md#Security

{{#if github.event.pull_request.number}}
Content here.
{{/if}}`,
			wantErr: false,
		},
		{
			name: "invalid - include in multiline template content",
			input: `# Test Workflow

{{#if github.event.issue.number}}
This is a longer template block
with multiple lines of content.

@include shared/tools.md

More content after the include.
{{/if}}`,
			wantErr: true,
			errMsg:  "@include/@import directives cannot be used inside template regions",
		},
		{
			name: "valid - template inside template outside (complex nesting)",
			input: `# Test Workflow

@include shared/header.md

{{#if github.event.issue.number}}
Content 1
{{/if}}

Some text between templates.

{{#if github.actor}}
Content 2
{{/if}}

@include shared/footer.md`,
			wantErr: false,
		},
		{
			name: "valid - empty template",
			input: `# Test Workflow

{{#if github.event.issue.number}}{{/if}}

@include shared/tools.md`,
			wantErr: false,
		},
		{
			name: "invalid - include in template with wrapped expression",
			input: `# Test Workflow

{{#if ${{ github.event.issue.number }} }}
@include shared/tools.md
{{/if}}`,
			wantErr: true,
			errMsg:  "@include/@import directives cannot be used inside template regions",
		},
		{
			name: "valid - no templates or includes",
			input: `# Test Workflow

Just regular markdown content.
No templates or includes here.`,
			wantErr: false,
		},
		{
			name: "invalid - indented include inside template",
			input: `# Test Workflow

{{#if github.event.issue.number}}
  @include shared/tools.md
{{/if}}`,
			wantErr: true,
			errMsg:  "@include/@import directives cannot be used inside template regions",
		},
		{
			name: "invalid - nested template with include in inner block",
			input: `# Test Workflow

{{#if github.event.issue.number}}
First level template.

  {{#if github.actor}}
  @include shared/nested-tools.md
  {{/if}}

End of first level.
{{/if}}`,
			wantErr: true,
			errMsg:  "@include/@import directives cannot be used inside template regions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoIncludesInTemplateRegions(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateNoIncludesInTemplateRegions() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateNoIncludesInTemplateRegions() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateNoIncludesInTemplateRegions() unexpected error = %v", err)
				}
			}
		})
	}
}
