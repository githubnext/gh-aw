package cli

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var compileInfrastructureLog = logger.New("cli:compile_infrastructure")

// updateGitAttributes ensures .gitattributes marks .lock.yml files as generated
func updateGitAttributes(successCount int, actionCache *workflow.ActionCache, verbose bool) error {
	compileInfrastructureLog.Printf("Updating .gitattributes (compiled=%d, actionCache=%v)", successCount, actionCache != nil)

	hasActionCacheEntries := actionCache != nil && len(actionCache.Entries) > 0

	// Only update if we successfully compiled workflows or have action cache entries
	if successCount > 0 || hasActionCacheEntries {
		compileInfrastructureLog.Printf("Updating .gitattributes (compiled=%d, actionCache=%v)", successCount, hasActionCacheEntries)
		updated, err := ensureGitAttributes()
		if err != nil {
			compileInfrastructureLog.Printf("Failed to update .gitattributes: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update .gitattributes: %v", err)))
			}
			return err
		}
		if updated {
			compileInfrastructureLog.Printf("Successfully updated .gitattributes")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated .gitattributes to mark .lock.yml files as generated"))
			}
		} else {
			compileInfrastructureLog.Print(".gitattributes already up to date")
		}
	} else {
		compileInfrastructureLog.Print("Skipping .gitattributes update (no compiled workflows and no action cache entries)")
	}

	return nil
}

// saveActionCache saves the action cache after all compilations
func saveActionCache(actionCache *workflow.ActionCache, verbose bool) error {
	if actionCache == nil {
		return nil
	}

	compileInfrastructureLog.Print("Saving action cache")

	if err := actionCache.Save(); err != nil {
		compileInfrastructureLog.Printf("Failed to save action cache: %v", err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save action cache: %v", err)))
		}
		return err
	}

	compileInfrastructureLog.Print("Action cache saved successfully")
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Action cache saved to "+actionCache.GetCachePath()))
	}

	return nil
}
