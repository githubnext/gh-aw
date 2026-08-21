package workflow

import (
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var artifactManagerLog = logger.New("workflow:artifact_manager")

// ArtifactManager tracks artifact uploads and downloads during compilation.
type ArtifactManager struct {
	uploads   map[string][]*ArtifactUpload
	downloads map[string][]*ArtifactDownload
}

// ArtifactUpload represents an artifact upload operation.
type ArtifactUpload struct {
	Name               string
	Paths              []string
	IfNoFilesFound     string
	IncludeHiddenFiles bool
	JobName            string
}

// ArtifactDownload represents an artifact download operation.
type ArtifactDownload struct {
	Name          string
	Pattern       string
	Path          string
	MergeMultiple bool
	JobName       string
	DependsOn     []string
}

// ArtifactInventoryEntry summarizes how an artifact is used by the workflow.
type ArtifactInventoryEntry struct {
	Name         string   `json:"name"`
	Created      bool     `json:"created"`
	Downloaded   bool     `json:"downloaded"`
	CreatedIn    []string `json:"created_in,omitempty"`
	DownloadedIn []string `json:"downloaded_in,omitempty"`
}

// NewArtifactManager creates a new artifact manager.
func NewArtifactManager() *ArtifactManager {
	return &ArtifactManager{
		uploads:   make(map[string][]*ArtifactUpload),
		downloads: make(map[string][]*ArtifactDownload),
	}
}

type artifactActionStep struct {
	Uses string         `yaml:"uses"`
	With map[string]any `yaml:"with"`
}

// AnalyzeJobs discovers artifact actions in generated jobs and validates uploads.
func (am *ArtifactManager) AnalyzeJobs(jobs map[string]*Job) error {
	am.Reset()

	jobNames := make([]string, 0, len(jobs))
	for jobName := range jobs {
		jobNames = append(jobNames, jobName)
	}
	sort.Strings(jobNames)

	for _, jobName := range jobNames {
		job := jobs[jobName]
		if job == nil || len(job.Steps) == 0 {
			continue
		}

		for _, step := range parseArtifactActionSteps(strings.Join(job.Steps, "")) {
			action := strings.TrimSpace(step.Uses)
			switch {
			case strings.HasPrefix(action, "actions/upload-artifact@"):
				if err := am.recordUploadStep(jobName, step.With); err != nil {
					return err
				}
			case strings.HasPrefix(action, "actions/download-artifact@"):
				am.recordDownloadStep(jobName, job, step.With)
			}
		}
	}

	return am.validateUploadNames()
}

func (am *ArtifactManager) recordUploadStep(jobName string, with map[string]any) error {
	name := artifactInputString(with, "name")
	if name == "" {
		name = "artifact"
	}
	if !strings.Contains(name, "${{") && strings.ContainsAny(name, "\"':<>|*?\r\n\\/") {
		return fmt.Errorf("artifact %q created by job %q has an invalid name", name, jobName)
	}

	paths := splitArtifactPaths(artifactInputString(with, "path"))
	if len(paths) == 0 {
		return fmt.Errorf("artifact %q created by job %q must have at least one path", name, jobName)
	}

	am.uploads[jobName] = append(am.uploads[jobName], &ArtifactUpload{
		Name:               name,
		Paths:              paths,
		IfNoFilesFound:     artifactInputString(with, "if-no-files-found"),
		IncludeHiddenFiles: artifactInputString(with, "include-hidden-files") == "true",
		JobName:            jobName,
	})
	return nil
}

func (am *ArtifactManager) recordDownloadStep(jobName string, job *Job, with map[string]any) {
	name := artifactInputString(with, "name")
	pattern := artifactInputString(with, "pattern")
	if name == "" && pattern == "" {
		pattern = "*"
	}
	downloadPath := artifactInputString(with, "path")
	if downloadPath == "" {
		downloadPath = "${{ github.workspace }}"
	}

	am.downloads[jobName] = append(am.downloads[jobName], &ArtifactDownload{
		Name:          name,
		Pattern:       pattern,
		Path:          downloadPath,
		MergeMultiple: artifactInputString(with, "merge-multiple") == "true",
		JobName:       jobName,
		DependsOn:     append([]string(nil), job.Needs...),
	})
}

func (am *ArtifactManager) validateUploadNames() error {
	createdBy := make(map[string]string)
	jobNames := make([]string, 0, len(am.uploads))
	for jobName := range am.uploads {
		jobNames = append(jobNames, jobName)
	}
	sort.Strings(jobNames)

	for _, jobName := range jobNames {
		for _, upload := range am.uploads[jobName] {
			if previousJob, exists := createdBy[upload.Name]; exists {
				if previousJob == jobName {
					return fmt.Errorf("artifact name clash: %q is created more than once by job %q", upload.Name, jobName)
				}
				return fmt.Errorf("artifact name clash: %q is created by jobs %q and %q", upload.Name, previousJob, jobName)
			}
			createdBy[upload.Name] = jobName
		}
	}
	return nil
}

// Inventory returns a stable summary of every artifact created or downloaded.
func (am *ArtifactManager) Inventory() []ArtifactInventoryEntry {
	entries := make(map[string]*ArtifactInventoryEntry)
	entryFor := func(name string) *ArtifactInventoryEntry {
		entry, ok := entries[name]
		if !ok {
			entry = &ArtifactInventoryEntry{Name: name}
			entries[name] = entry
		}
		return entry
	}

	for jobName, uploads := range am.uploads {
		for _, upload := range uploads {
			entry := entryFor(upload.Name)
			entry.Created = true
			entry.CreatedIn = appendUnique(entry.CreatedIn, jobName)
		}
	}

	for jobName, downloads := range am.downloads {
		for _, download := range downloads {
			matches := am.matchingArtifactNames(download)
			if len(matches) == 0 {
				if download.Name != "" {
					matches = []string{download.Name}
				}
			}
			for _, name := range matches {
				entry := entryFor(name)
				entry.Downloaded = true
				entry.DownloadedIn = appendUnique(entry.DownloadedIn, jobName)
			}
		}
	}

	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)

	inventory := make([]ArtifactInventoryEntry, 0, len(names))
	for _, name := range names {
		entry := entries[name]
		sort.Strings(entry.CreatedIn)
		sort.Strings(entry.DownloadedIn)
		inventory = append(inventory, *entry)
	}
	return inventory
}

func parseArtifactActionSteps(content string) []artifactActionStep {
	var steps []artifactActionStep
	var current *artifactActionStep
	var withIndent int
	var blockKey string
	var blockIndent int

	flush := func() {
		if current != nil && current.Uses != "" {
			steps = append(steps, *current)
		}
		current = nil
		withIndent = 0
		blockKey = ""
		blockIndent = 0
	}

	for line := range strings.SplitSeq(content, "\n") {
		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "      - ") {
			flush()
			current = &artifactActionStep{}
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
		}
		if current == nil || trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if blockKey != "" && indent > blockIndent {
			current.With[blockKey] = fmt.Sprint(current.With[blockKey]) + strings.TrimSpace(line) + "\n"
			continue
		}
		blockKey = ""

		if value, ok := strings.CutPrefix(trimmed, "uses:"); ok {
			current.Uses = strings.Trim(strings.TrimSpace(value), `"'`)
			continue
		}
		if trimmed == "with:" {
			withIndent = indent
			if current.With == nil {
				current.With = make(map[string]any)
			}
			continue
		}
		if withIndent == 0 || indent <= withIndent {
			continue
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if value == "|" || value == ">" {
			blockKey = key
			blockIndent = indent
			current.With[key] = ""
			continue
		}
		current.With[key] = value
	}
	flush()
	return steps
}

func (am *ArtifactManager) matchingArtifactNames(download *ArtifactDownload) []string {
	var matches []string
	for _, uploads := range am.uploads {
		for _, upload := range uploads {
			if download.Name == upload.Name || (download.Pattern != "" && matchesArtifactPattern(upload.Name, download.Pattern)) {
				matches = appendUnique(matches, upload.Name)
			}
		}
	}
	sort.Strings(matches)
	return matches
}

func matchesArtifactPattern(name, pattern string) bool {
	for alternative := range strings.SplitSeq(strings.Trim(pattern, "{}"), ",") {
		if matched, err := path.Match(strings.TrimSpace(alternative), name); err == nil && matched {
			return true
		}
	}
	return false
}

func artifactInputString(with map[string]any, key string) string {
	if with == nil {
		return ""
	}
	value, exists := with[key]
	if !exists || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func splitArtifactPaths(value string) []string {
	var paths []string
	for line := range strings.SplitSeq(value, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			paths = append(paths, line)
		}
	}
	return paths
}

func appendUnique(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

// Reset clears all tracked uploads and downloads.
func (am *ArtifactManager) Reset() {
	am.uploads = make(map[string][]*ArtifactUpload)
	am.downloads = make(map[string][]*ArtifactDownload)
	artifactManagerLog.Print("Reset artifact manager")
}
