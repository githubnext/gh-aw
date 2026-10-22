package cli

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

type postFieldRemovalEdit func(*codemod.Frontmatter, map[string]any, any) error

type fieldRemovalCodemodConfig struct {
	ID           string
	Name         string
	Description  string
	IntroducedIn string
	ParentKey    string
	FieldKey     string
	LogMsg       string
	Log          *logger.Logger
	PostEdit     postFieldRemovalEdit
}

func newFieldRemovalCodemod(cfg fieldRemovalCodemodConfig) Codemod {
	return Codemod{
		ID:           cfg.ID,
		Name:         cfg.Name,
		Description:  cfg.Description,
		IntroducedIn: cfg.IntroducedIn,
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			parent, ok := frontmatter[cfg.ParentKey].(map[string]any)
			if !ok {
				return content, false, nil
			}
			value, found := parent[cfg.FieldKey]
			if !found {
				return content, false, nil
			}
			updated, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				if err := f.Remove(cfg.ParentKey, cfg.FieldKey); err != nil {
					return err
				}
				if err := f.PruneEmpty(cfg.ParentKey); err != nil {
					return err
				}
				if cfg.PostEdit != nil {
					return cfg.PostEdit(f, frontmatter, value)
				}
				return nil
			})
			if err != nil || !applied {
				return content, false, err
			}
			cfg.Log.Print(cfg.LogMsg)
			return updated, true, nil
		},
	}
}

type moveToOnBlockConfig struct {
	ID           string
	Name         string
	Description  string
	IntroducedIn string
	FieldKey     string
	Log          *logger.Logger
}

func newMoveTopLevelKeyToOnBlockCodemod(cfg moveToOnBlockConfig) Codemod {
	return Codemod{
		ID:           cfg.ID,
		Name:         cfg.Name,
		Description:  cfg.Description,
		IntroducedIn: cfg.IntroducedIn,
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			_, exists := frontmatter[cfg.FieldKey]
			if !exists {
				return content, false, nil
			}
			on, hasOn := frontmatter["on"]
			if hasOn {
				mapping, ok := on.(map[string]any)
				if !ok {
					// Trigger shorthand must not be overwritten by a mapping.
					return content, false, nil
				}
				if _, exists := mapping[cfg.FieldKey]; exists {
					return content, false, nil
				}
			}
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Move([]string{cfg.FieldKey}, []string{"on", cfg.FieldKey})
			})
		},
	}
}
