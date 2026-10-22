package cli

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

type migrateEngineFieldToTopLevelOptions struct {
	engineField            string
	targetTopLevelField    string
	preserveTopLevelFields []string
	log                    *logger.Logger
	skipInlineMessage      string
	removedMessage         string
	migratedMessage        string
}

func migrateEngineFieldToTopLevel(content string, _ map[string]any, opts migrateEngineFieldToTopLevelOptions) (string, bool, error) {
	return codemod.Apply(content, func(f *codemod.Frontmatter) error {
		path := []string{"engine", opts.engineField}
		if _, exists := f.Lookup(path...); !exists {
			return nil
		}
		preserve := false
		for _, key := range opts.preserveTopLevelFields {
			if _, exists := f.Lookup(key); exists {
				preserve = true
				break
			}
		}
		if preserve {
			if err := f.Remove(path...); err != nil {
				return err
			}
		} else if err := f.Move(path, []string{opts.targetTopLevelField}); err != nil {
			return err
		}
		if err := f.PruneEmpty("engine"); err != nil {
			return err
		}
		if opts.log != nil {
			if preserve {
				opts.log.Print(opts.removedMessage)
			} else {
				opts.log.Print(opts.migratedMessage)
			}
		}
		return nil
	})
}
