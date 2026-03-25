//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestValidateEventFilters(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantErr     bool
		errContains string
	}{
		// Valid configurations
		{
			name: "valid branches only",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"main"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid branches-ignore only",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches-ignore": []string{"dev"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid paths only",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"paths": []string{"src/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid paths-ignore only",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"paths-ignore": []string{"docs/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid both branches and paths",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"main"},
						"paths":    []string{"src/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid pull_request with branches",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"branches": []string{"main"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid pull_request with paths-ignore",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"paths-ignore": []string{"docs/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "valid no on section",
			frontmatter: map[string]any{},
			wantErr:     false,
		},
		{
			name: "valid on section with string value",
			frontmatter: map[string]any{
				"on": "push",
			},
			wantErr: false,
		},
		{
			name: "valid push event with empty map",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{},
				},
			},
			wantErr: false,
		},
		{
			name: "valid push event with null value",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": nil,
				},
			},
			wantErr: false,
		},

		// Invalid configurations - branches/branches-ignore
		{
			name: "invalid both branches and branches-ignore on push",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches":        []string{"main"},
						"branches-ignore": []string{"dev"},
					},
				},
			},
			wantErr:     true,
			errContains: "push event cannot specify both 'branches' and 'branches-ignore'",
		},
		{
			name: "invalid both branches and branches-ignore on pull_request",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"branches":        []string{"main"},
						"branches-ignore": []string{"dev"},
					},
				},
			},
			wantErr:     true,
			errContains: "pull_request event cannot specify both 'branches' and 'branches-ignore'",
		},

		// Invalid configurations - paths/paths-ignore
		{
			name: "invalid both paths and paths-ignore on push",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"paths":        []string{"src/**"},
						"paths-ignore": []string{"docs/**"},
					},
				},
			},
			wantErr:     true,
			errContains: "push event cannot specify both 'paths' and 'paths-ignore'",
		},
		{
			name: "invalid both paths and paths-ignore on pull_request",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"paths":        []string{"src/**"},
						"paths-ignore": []string{"docs/**"},
					},
				},
			},
			wantErr:     true,
			errContains: "pull_request event cannot specify both 'paths' and 'paths-ignore'",
		},

		// Complex cases
		{
			name: "invalid multiple violations on same event",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches":        []string{"main"},
						"branches-ignore": []string{"dev"},
						"paths":           []string{"src/**"},
						"paths-ignore":    []string{"docs/**"},
					},
				},
			},
			wantErr:     true,
			errContains: "branches", // Should catch the first violation
		},
		{
			name: "valid one event, invalid another",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"main"},
					},
					"pull_request": map[string]any{
						"branches":        []string{"main"},
						"branches-ignore": []string{"dev"},
					},
				},
			},
			wantErr:     true,
			errContains: "pull_request",
		},
		{
			name: "valid both push and pull_request without conflicts",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"main"},
						"paths":    []string{"src/**"},
					},
					"pull_request": map[string]any{
						"branches-ignore": []string{"dev"},
						"paths-ignore":    []string{"docs/**"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEventFilters(tt.frontmatter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEventFilters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateEventFilters() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidateFilterExclusivity(t *testing.T) {
	tests := []struct {
		name        string
		eventVal    any
		eventName   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid nil event",
			eventVal:  nil,
			eventName: "push",
			wantErr:   false,
		},
		{
			name:      "valid string event",
			eventVal:  "some-string",
			eventName: "push",
			wantErr:   false,
		},
		{
			name:      "valid empty map",
			eventVal:  map[string]any{},
			eventName: "push",
			wantErr:   false,
		},
		{
			name: "valid single filter",
			eventVal: map[string]any{
				"branches": []string{"main"},
			},
			eventName: "push",
			wantErr:   false,
		},
		{
			name: "invalid branches conflict",
			eventVal: map[string]any{
				"branches":        []string{"main"},
				"branches-ignore": []string{"dev"},
			},
			eventName:   "push",
			wantErr:     true,
			errContains: "branches",
		},
		{
			name: "invalid paths conflict",
			eventVal: map[string]any{
				"paths":        []string{"src/**"},
				"paths-ignore": []string{"docs/**"},
			},
			eventName:   "pull_request",
			wantErr:     true,
			errContains: "paths",
		},
		{
			name: "valid with other fields present",
			eventVal: map[string]any{
				"branches": []string{"main"},
				"types":    []string{"opened"},
				"paths":    []string{"src/**"},
			},
			eventName: "pull_request",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilterExclusivity(tt.eventVal, tt.eventName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFilterExclusivity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateFilterExclusivity() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidateGlobPatterns(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantErr     bool
		errContains string
	}{
		// ---- valid ref globs ----
		{
			name: "valid branch pattern main",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"main"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid branch wildcard",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"release/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid tag pattern v*",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"tags": []string{"v*"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid semver tag pattern",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"tags": []string{"v[0-9]+.[0-9]+.[0-9]+"},
					},
				},
			},
			wantErr: false,
		},
		// ---- valid path globs ----
		{
			name: "valid path src/**",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"paths": []string{"src/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid paths-ignore docs/**",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"paths-ignore": []string{"docs/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid negated path glob",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"paths": []string{"!docs/**"},
					},
				},
			},
			wantErr: false,
		},
		// ---- no 'on' section ----
		{
			name:        "no on section",
			frontmatter: map[string]any{},
			wantErr:     false,
		},
		// ---- invalid ref glob ----
		{
			name: "invalid branch pattern with space",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"main branch"},
					},
				},
			},
			wantErr:     true,
			errContains: "on.push.branches",
		},
		{
			name: "invalid tag pattern with colon",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"tags": []string{"v1:0"},
					},
				},
			},
			wantErr:     true,
			errContains: "on.push.tags",
		},
		// ---- ./ prefix path glob (always invalid in GitHub Actions) ----
		{
			name: "invalid path glob with ./ prefix",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"paths": []string{"./src/**/*.go"},
					},
				},
			},
			wantErr:     true,
			errContains: "on.push.paths",
		},
		{
			name: "invalid paths-ignore with ./ prefix",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"paths-ignore": []string{"./docs/**"},
					},
				},
			},
			wantErr:     true,
			errContains: "on.pull_request.paths-ignore",
		},
		// ---- pull_request event ----
		{
			name: "valid pull_request branch pattern",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"branches": []string{"main", "release/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid pull_request branch with tilde",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"branches": []string{"~invalid"},
					},
				},
			},
			wantErr:     true,
			errContains: "on.pull_request.branches",
		},
		// ---- non-glob on section ----
		{
			name: "on section is a string (not a map)",
			frontmatter: map[string]any{
				"on": "push",
			},
			wantErr: false,
		},
		// ---- []any pattern list ----
		{
			name: "valid branch list as []any",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []any{"main", "develop"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid path in []any list",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"paths": []any{"./bad/**"},
					},
				},
			},
			wantErr:     true,
			errContains: "on.push.paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGlobPatterns(tt.frontmatter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGlobPatterns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateGlobPatterns() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

// TestValidateGlobPatternsExtendedEvents verifies that glob validation is applied to all
// supported GitHub Actions events (pull_request_target, workflow_run) and all filter keys
// (branches-ignore, tags, tags-ignore, paths-ignore).
func TestValidateGlobPatternsExtendedEvents(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantErr     bool
		errContains string
	}{
		// ---- pull_request_target event ----
		{
			name: "valid pull_request_target branches",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request_target": map[string]any{
						"branches": []string{"main", "release/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid pull_request_target branch with space",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request_target": map[string]any{
						"branches": []string{"main branch"},
					},
				},
			},
			wantErr:     true,
			errContains: "on.pull_request_target.branches",
		},
		{
			name: "invalid pull_request_target path with ./ prefix",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request_target": map[string]any{
						"paths": []string{"./src/**"},
					},
				},
			},
			wantErr:     true,
			errContains: "on.pull_request_target.paths",
		},
		// ---- workflow_run event ----
		{
			name: "valid workflow_run branches",
			frontmatter: map[string]any{
				"on": map[string]any{
					"workflow_run": map[string]any{
						"branches": []string{"main"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid workflow_run branch with tilde",
			frontmatter: map[string]any{
				"on": map[string]any{
					"workflow_run": map[string]any{
						"branches": []string{"~bad"},
					},
				},
			},
			wantErr:     true,
			errContains: "on.workflow_run.branches",
		},
		// ---- branches-ignore filter key ----
		{
			name: "valid branches-ignore pattern",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches-ignore": []string{"dependabot/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid branches-ignore pattern with colon",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches-ignore": []string{"feat:bad"},
					},
				},
			},
			wantErr:     true,
			errContains: "on.push.branches-ignore",
		},
		// ---- tags filter key ----
		{
			name: "valid tags-ignore pattern",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"tags-ignore": []string{"v*-beta"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid tags-ignore pattern with space",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"tags-ignore": []string{"bad tag"},
					},
				},
			},
			wantErr:     true,
			errContains: "on.push.tags-ignore",
		},
		// ---- second pattern in a list is invalid ----
		{
			name: "second branch in list is invalid",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"main", "bad branch"},
					},
				},
			},
			wantErr:     true,
			errContains: "on.push.branches",
		},
		{
			name: "second path in list is invalid",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"paths": []string{"src/**", "./bad"},
					},
				},
			},
			wantErr:     true,
			errContains: "on.push.paths",
		},
		// ---- non-map event value is gracefully skipped ----
		{
			name: "push event with null value is skipped",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": nil,
				},
			},
			wantErr: false,
		},
		{
			name: "push event with string value is skipped",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": "simple-string",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGlobPatterns(tt.frontmatter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGlobPatterns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateGlobPatterns() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

// TestValidateRefGlob exercises the low-level validateRefGlob helper directly.
func TestValidateRefGlob(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		// valid patterns
		{name: "simple branch name", pattern: "main", wantErr: false},
		{name: "wildcard branch", pattern: "release/*", wantErr: false},
		{name: "double wildcard", pattern: "feature/**", wantErr: false},
		{name: "negated pattern", pattern: "!dependabot/**", wantErr: false},
		{name: "version tag", pattern: "v1.*", wantErr: false},
		{name: "character class", pattern: "release/v[0-9]*", wantErr: false},
		// invalid patterns
		{name: "empty string", pattern: "", wantErr: true},
		{name: "contains space", pattern: "feature branch", wantErr: true},
		{name: "contains tilde", pattern: "~bad", wantErr: true},
		{name: "contains caret", pattern: "bad^name", wantErr: true},
		{name: "contains colon", pattern: "bad:name", wantErr: true},
		{name: "starts with slash", pattern: "/branch", wantErr: true},
		{name: "ends with slash", pattern: "branch/", wantErr: true},
		{name: "ends with dot", pattern: "branch.", wantErr: true},
		{name: "empty character class", pattern: "feat/[]bad", wantErr: true},
		{name: "unclosed character class", pattern: "feat/[a-z", wantErr: true},
		{name: "bare exclamation", pattern: "!", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateRefGlob(tt.pattern)
			gotErr := len(errs) > 0
			if gotErr != tt.wantErr {
				t.Errorf("validateRefGlob(%q): got errors=%v, wantErr=%v", tt.pattern, errs, tt.wantErr)
			}
		})
	}
}

// TestValidatePathGlob exercises the low-level validatePathGlob helper directly.
func TestValidatePathGlob(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		// valid patterns
		{name: "simple filename", pattern: "README.md", wantErr: false},
		{name: "subdirectory wildcard", pattern: "src/**/*.go", wantErr: false},
		{name: "negated path", pattern: "!docs/**", wantErr: false},
		{name: "root wildcard", pattern: "*.go", wantErr: false},
		{name: "deep path", pattern: "a/b/c/**", wantErr: false},
		{name: "negated dot-path is valid", pattern: "!./ignored", wantErr: true}, // ./ after ! still invalid
		// invalid: ./ and ../ prefixes
		{name: "./ prefix", pattern: "./src/**", wantErr: true},
		{name: "../ prefix", pattern: "../other", wantErr: true},
		{name: "bare dot", pattern: ".", wantErr: true},
		{name: "bare double-dot", pattern: "..", wantErr: true},
		{name: "negated ./ prefix", pattern: "!./bad", wantErr: true},
		// invalid: leading/trailing spaces
		{name: "leading space", pattern: " src/**", wantErr: true},
		{name: "trailing space", pattern: "src/** ", wantErr: true},
		// invalid: empty and bare exclamation
		{name: "empty string", pattern: "", wantErr: true},
		{name: "bare exclamation", pattern: "!", wantErr: true},
		// invalid: unclosed bracket
		{name: "unclosed bracket", pattern: "src/[a-z", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validatePathGlob(tt.pattern)
			gotErr := len(errs) > 0
			if gotErr != tt.wantErr {
				t.Errorf("validatePathGlob(%q): got errors=%v, wantErr=%v", tt.pattern, errs, tt.wantErr)
			}
		})
	}
}

// TestToStringSlice tests the toStringSlice conversion helper.
func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    []string
		wantErr bool
	}{
		{name: "[]string", input: []string{"a", "b"}, want: []string{"a", "b"}, wantErr: false},
		{name: "[]any strings", input: []any{"a", "b"}, want: []string{"a", "b"}, wantErr: false},
		{name: "single string", input: "hello", want: []string{"hello"}, wantErr: false},
		{name: "[]any with int", input: []any{"a", 42}, wantErr: true},
		{name: "integer type", input: 123, wantErr: true},
		{name: "nil", input: nil, wantErr: true},
		{name: "empty []string", input: []string{}, want: []string{}, wantErr: false},
		{name: "empty []any", input: []any{}, want: []string{}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toStringSlice(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("toStringSlice(%v): err = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("toStringSlice(%v): got %v, want %v", tt.input, got, tt.want)
					return
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("toStringSlice(%v)[%d]: got %q, want %q", tt.input, i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}
