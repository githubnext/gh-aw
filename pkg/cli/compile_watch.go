package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var compileWatchLog = logger.New("cli:compile_watch")

// watchAndCompileWorkflows watches for changes to workflow files and recompiles them automatically
func watchAndCompileWorkflows(ctx context.Context, markdownFile string, compiler *workflow.Compiler, verbose bool) error {
	_, workflowsDir, markdownFile, err := watchAndCompileWorkflowsInputs(markdownFile)
	if err != nil {
		return err
	}

	depGraph := watchAndCompileWorkflowsBuildDependencyGraph(workflowsDir, compiler, verbose)
	watcher, err := watchAndCompileWorkflowsCreateWatcher(workflowsDir)
	if err != nil {
		return err
	}
	defer watcher.Close()

	watchAndCompileWorkflowsPrintBegin(markdownFile, workflowsDir, verbose)
	watchAndCompileWorkflowsInitial(ctx, markdownFile, compiler, workflowsDir, verbose)
	return watchAndCompileWorkflowsLoop(ctx, markdownFile, compiler, verbose, watcher, depGraph)
}

func watchAndCompileWorkflowsInputs(markdownFile string) (string, string, string, error) {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return "", "", "", fmt.Errorf("watch mode requires being in a git repository: %w", err)
	}
	workflowsDir := filepath.Join(gitRoot, constants.GetWorkflowDir())
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return "", "", "", fmt.Errorf("the %s directory does not exist in git root (%s)", constants.GetWorkflowDir(), gitRoot)
	}
	if markdownFile != "" {
		if !filepath.IsAbs(markdownFile) {
			markdownFile = filepath.Join(workflowsDir, markdownFile)
		}
		if _, err := os.Stat(markdownFile); os.IsNotExist(err) {
			return "", "", "", fmt.Errorf("specified markdown file does not exist: %s", markdownFile)
		}
	}
	return gitRoot, workflowsDir, markdownFile, nil
}

func watchAndCompileWorkflowsBuildDependencyGraph(workflowsDir string, compiler *workflow.Compiler, verbose bool) *DependencyGraph {
	depGraph := NewDependencyGraph(workflowsDir)
	compileWatchLog.Print("Building dependency graph for watch mode...")
	if err := depGraph.BuildGraph(compiler); err != nil {
		compileWatchLog.Printf("Warning: failed to build dependency graph: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to build dependency graph: %v", err)))
	} else {
		compileWatchLog.Printf("Dependency graph built successfully: %d workflows", len(depGraph.nodes))
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Dependency graph built: %d workflows", len(depGraph.nodes))))
		}
	}
	return depGraph
}

func watchAndCompileWorkflowsCreateWatcher(workflowsDir string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewBufferedWatcher(100)
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}
	if err := watchAndCompileWorkflowsAddWatchPath(watcher, workflowsDir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch directory %s: %w", workflowsDir, err)
	}
	if err := watchAndCompileWorkflowsWatchSubdirectories(watcher, workflowsDir); err != nil {
		compileWatchLog.Printf("Failed to walk subdirectories: %v", err)
	}
	return watcher, nil
}

func watchAndCompileWorkflowsAddWatchPath(watcher *fsnotify.Watcher, path string) error {
	if runtime.GOOS == "windows" {
		return watcher.AddWith(path, fsnotify.WithBufferSize(64*1024))
	}
	return watcher.Add(path)
}

func watchAndCompileWorkflowsWatchSubdirectories(watcher *fsnotify.Watcher, workflowsDir string) error {
	return filepath.Walk(workflowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && path != workflowsDir {
			if err := watchAndCompileWorkflowsAddWatchPath(watcher, path); err != nil {
				compileWatchLog.Printf("Failed to watch subdirectory %s: %v", path, err)
			} else {
				compileWatchLog.Printf("Watching subdirectory: %s", path)
			}
		}
		return nil
	})
}

func watchAndCompileWorkflowsPrintBegin(markdownFile string, workflowsDir string, verbose bool) {
	if markdownFile != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Watching for file changes to %s...", markdownFile)))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Watching for file changes in %s...", workflowsDir)))
	}
	if verbose {
		fmt.Fprintln(os.Stderr, "Press Ctrl+C to stop watching.")
	}
}

func watchAndCompileWorkflowsInitial(ctx context.Context, markdownFile string, compiler *workflow.Compiler, workflowsDir string, verbose bool) {
	if markdownFile == "" {
		fmt.Fprintln(os.Stderr, "Watching for file changes")
		if verbose {
			fmt.Fprintln(os.Stderr, "🔨 Initial compilation of all workflow files...")
		}
		stats, err := compileAllWorkflowFiles(ctx, compiler, workflowsDir, verbose)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Initial compilation failed: %v", err)))
		}
		printCompilationSummary(stats, false)
		return
	}
	watchAndCompileWorkflowsInitialSingle(ctx, markdownFile, compiler, verbose)
}

func watchAndCompileWorkflowsInitialSingle(ctx context.Context, markdownFile string, compiler *workflow.Compiler, verbose bool) {
	compiler.ResetWarningCount()
	stats := &CompilationStats{}
	fmt.Fprintln(os.Stderr, "Watching for file changes")
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Initial compilation of %s...", markdownFile)))
	}
	compileSingleFile(ctx, compiler, markdownFile, stats, verbose, false)
	stats.Warnings = compiler.GetWarningCount()
	printCompilationSummary(stats, false)
}

func watchAndCompileWorkflowsLoop(
	ctx context.Context,
	markdownFile string,
	compiler *workflow.Compiler,
	verbose bool,
	watcher *fsnotify.Watcher,
	depGraph *DependencyGraph,
) error {
	const debounceDelay = 300 * time.Millisecond
	var debounceTimer *time.Timer
	var debounceMu sync.Mutex
	modifiedFiles := make(map[string]struct{})

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return errors.New("watcher channel closed")
			}
			watchAndCompileWorkflowsHandleEvent(watchAndCompileWorkflowsHandleEventParams{
				Ctx:           ctx,
				MarkdownFile:  markdownFile,
				Compiler:      compiler,
				Verbose:       verbose,
				DepGraph:      depGraph,
				Event:         event,
				DebounceTimer: &debounceTimer,
				DebounceMu:    &debounceMu,
				ModifiedFiles: &modifiedFiles,
				DebounceDelay: debounceDelay,
			})
		case err, ok := <-watcher.Errors:
			if !ok {
				return errors.New("watcher error channel closed")
			}
			watchAndCompileWorkflowsHandleError(err, verbose)
		case <-ctx.Done():
			watchAndCompileWorkflowsStop(debounceTimer, verbose)
			return nil
		}
	}
}

type watchAndCompileWorkflowsHandleEventParams struct {
	Ctx           context.Context
	MarkdownFile  string
	Compiler      *workflow.Compiler
	Verbose       bool
	DepGraph      *DependencyGraph
	Event         fsnotify.Event
	DebounceTimer **time.Timer
	DebounceMu    *sync.Mutex
	ModifiedFiles *map[string]struct{}
	DebounceDelay time.Duration
}

func watchAndCompileWorkflowsHandleEvent(p watchAndCompileWorkflowsHandleEventParams) {
	if p.Event.Has(fsnotify.Chmod) || !strings.HasSuffix(p.Event.Name, ".md") {
		return
	}
	if p.MarkdownFile != "" && p.Event.Name != p.MarkdownFile {
		return
	}
	compileWatchLog.Printf("Detected change: %s (%s)", p.Event.Name, p.Event.Op.String())
	if p.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Detected change: %s (%s)", p.Event.Name, p.Event.Op.String())))
	}
	if p.Event.Has(fsnotify.Remove) {
		handleFileDeleted(p.Event.Name, p.Verbose)
		p.DepGraph.RemoveWorkflow(p.Event.Name)
		return
	}
	if p.Event.Has(fsnotify.Write) || p.Event.Has(fsnotify.Create) {
		watchAndCompileWorkflowsDebounce(watchAndCompileWorkflowsDebounceParams{
			Ctx:           p.Ctx,
			Compiler:      p.Compiler,
			Verbose:       p.Verbose,
			DepGraph:      p.DepGraph,
			FileName:      p.Event.Name,
			DebounceTimer: p.DebounceTimer,
			DebounceMu:    p.DebounceMu,
			ModifiedFiles: p.ModifiedFiles,
			DebounceDelay: p.DebounceDelay,
		})
	}
}

type watchAndCompileWorkflowsDebounceParams struct {
	Ctx           context.Context
	Compiler      *workflow.Compiler
	Verbose       bool
	DepGraph      *DependencyGraph
	FileName      string
	DebounceTimer **time.Timer
	DebounceMu    *sync.Mutex
	ModifiedFiles *map[string]struct{}
	DebounceDelay time.Duration
}

func watchAndCompileWorkflowsDebounce(p watchAndCompileWorkflowsDebounceParams) {
	p.DebounceMu.Lock()
	defer p.DebounceMu.Unlock()
	(*p.ModifiedFiles)[p.FileName] = struct{}{}
	if *p.DebounceTimer != nil {
		(*p.DebounceTimer).Stop()
	}
	*p.DebounceTimer = time.AfterFunc(p.DebounceDelay, func() {
		filesToCompile := watchAndCompileWorkflowsTakeModifiedFiles(p.DebounceMu, p.ModifiedFiles)
		compileModifiedFilesWithDependencies(p.Ctx, p.Compiler, p.DepGraph, filesToCompile, p.Verbose)
	})
}

func watchAndCompileWorkflowsTakeModifiedFiles(debounceMu *sync.Mutex, modifiedFiles *map[string]struct{}) []string {
	debounceMu.Lock()
	defer debounceMu.Unlock()
	files := make([]string, 0, len(*modifiedFiles))
	for file := range *modifiedFiles {
		files = append(files, file)
	}
	*modifiedFiles = make(map[string]struct{})
	return files
}

func watchAndCompileWorkflowsHandleError(err error, verbose bool) {
	compileWatchLog.Printf("Watcher error: %v", err)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Watcher error: %v", err)))
	}
}

func watchAndCompileWorkflowsStop(debounceTimer *time.Timer, verbose bool) {
	if verbose {
		fmt.Fprintln(os.Stderr, "\n🛑 Stopping watch mode...")
	}
	if debounceTimer != nil {
		debounceTimer.Stop()
	}
}
