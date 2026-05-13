package cli

import (
	"context"
	"fmt"
	"github.com/github/gh-aw/pkg/constants"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/tty"
	"github.com/github/gh-aw/pkg/workflow"
)

var compileUpdateCheckLog = logger.New("cli:update_check")

const (
	compileUpdateCheckDisableEnv = "GH_AW_DISABLE_UPDATE_CHECK"
	compileUpdateCheckFileName   = "gh-aw-last-compile-update-check"
	compileUpdateCheckInterval   = 24 * time.Hour
	compileUpdateCheckTimeout    = 3 * time.Second
	compileUpdateCheckNoWait     = 0
)

var (
	compileUpdateCheckLatestReleaseURL = "https://github.com/github/gh-aw/releases/latest"
	compileUpdateCheckProbeURLFunc     = func(tag string) string {
		return fmt.Sprintf("https://raw.githubusercontent.com/github/gh-aw/refs/tags/%s/go.mod", tag)
	}
	compileUpdateCheckHTTPClientFactory = func() *http.Client {
		return &http.Client{Timeout: compileUpdateCheckTimeout}
	}
	compileUpdateCheckIsTerminalFunc  = tty.IsStderrTerminal
	getCompileUpdateCheckFilePathFunc = getCompileUpdateCheckFilePathImpl
)

type compileUpdateNotificationKind string

const (
	compileUpdateNotificationMinorBehind compileUpdateNotificationKind = "minor_behind"
	compileUpdateNotificationRemovedTag  compileUpdateNotificationKind = "removed_tag"
)

type compileUpdateNotification struct {
	Kind           compileUpdateNotificationKind
	CurrentVersion string
	LatestVersion  string
}

// StartCompileUpdateCheck begins a best-effort update check for the compile command.
// The returned function should be called once before the command exits to print any
// ready notification without blocking compilation for long.
func StartCompileUpdateCheck(ctx context.Context, noCheckUpdate bool, verbose bool) func() {
	if !shouldRunCompileUpdateCheck(noCheckUpdate) {
		return func() {}
	}
	updateCompileUpdateCheckTime()

	results := make(chan *compileUpdateNotification, 1) // buffered channel closed by sender goroutine via defer

	go func() {
		defer close(results)
		defer func() {
			if r := recover(); r != nil {
				compileUpdateCheckLog.Printf("Panic in compile update check (recovered): %v", r)
			}
		}()

		if ctx.Err() != nil {
			compileUpdateCheckLog.Printf("Compile update check cancelled before starting: %v", ctx.Err())
			return
		}

		result, err := runCompileUpdateCheck(ctx, compileUpdateCheckHTTPClientFactory())
		if err != nil {
			compileUpdateCheckLog.Printf("Compile update check failed (ignoring): %v", err)
			return
		}
		if result == nil {
			if verbose {
				compileUpdateCheckLog.Print("No compile update notification needed")
			}
			return
		}

		select {
		case results <- result:
		case <-ctx.Done():
		}
	}()

	return func() {
		result := waitForCompileUpdateNotification(ctx, results, compileUpdateCheckNoWait)
		if result != nil {
			printCompileUpdateNotification(result)
		}
	}
}

func shouldRunCompileUpdateCheck(noCheckUpdate bool) bool {
	if noCheckUpdate {
		compileUpdateCheckLog.Print("Update check disabled via --no-check-update flag")
		return false
	}
	if os.Getenv(compileUpdateCheckDisableEnv) != "" {
		compileUpdateCheckLog.Printf("Update check disabled via %s", compileUpdateCheckDisableEnv)
		return false
	}
	if IsRunningInCI() {
		compileUpdateCheckLog.Print("Update check disabled in CI environment")
		return false
	}
	if isRunningAsMCPServer() {
		compileUpdateCheckLog.Print("Update check disabled in MCP server mode")
		return false
	}
	if !compileUpdateCheckIsTerminalFunc() {
		compileUpdateCheckLog.Print("Update check disabled when stderr is not a terminal")
		return false
	}

	lastCheckFile := getCompileUpdateCheckFilePath()
	if lastCheckFile == "" {
		compileUpdateCheckLog.Print("Could not determine compile update check file path")
		return false
	}

	data, err := os.ReadFile(lastCheckFile)
	if err != nil {
		if !os.IsNotExist(err) {
			compileUpdateCheckLog.Printf("Error reading compile update check file: %v", err)
		}
		return true
	}

	lastCheck, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		compileUpdateCheckLog.Printf("Error parsing compile update check time: %v", err)
		return true
	}

	elapsed := time.Since(lastCheck)
	if elapsed < compileUpdateCheckInterval {
		compileUpdateCheckLog.Printf("Last compile update check was %v ago, skipping", elapsed)
		return false
	}
	return true
}

func waitForCompileUpdateNotification(ctx context.Context, results <-chan *compileUpdateNotification, timeout time.Duration) *compileUpdateNotification {
	if results == nil {
		return nil
	}

	if timeout <= 0 {
		select {
		case result, ok := <-results:
			if !ok {
				return nil
			}
			return result
		default:
			return nil
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result, ok := <-results:
		if !ok {
			return nil
		}
		return result
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return nil
	}
}

func runCompileUpdateCheck(ctx context.Context, client *http.Client) (*compileUpdateNotification, error) {
	currentVersion := GetVersion()
	if !workflow.IsReleasedVersion(currentVersion) {
		compileUpdateCheckLog.Print("Not a released version, skipping update check")
		return nil, nil
	}

	latestVersion, err := fetchLatestReleaseTag(ctx, client)
	if err != nil {
		return nil, err
	}
	if latestVersion == "" {
		return nil, nil
	}

	latestTagExists, err := downloadReleaseProbeFile(ctx, client, latestVersion)
	if err != nil {
		return nil, err
	}
	if !latestTagExists {
		compileUpdateCheckLog.Printf("Latest release tag %s did not expose the probe file; skipping", latestVersion)
		return nil, nil
	}

	currentTagExists, err := downloadReleaseProbeFile(ctx, client, currentVersion)
	if err != nil {
		return nil, err
	}
	if !currentTagExists {
		return &compileUpdateNotification{
			Kind:           compileUpdateNotificationRemovedTag,
			CurrentVersion: currentVersion,
			LatestVersion:  latestVersion,
		}, nil
	}

	if !isMinorVersionBehind(currentVersion, latestVersion) {
		return nil, nil
	}

	return &compileUpdateNotification{
		Kind:           compileUpdateNotificationMinorBehind,
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
	}, nil
}

func fetchLatestReleaseTag(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, compileUpdateCheckLatestReleaseURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create latest release request: %w", err)
	}
	req.Header.Set("User-Agent", "gh-aw/"+GetVersion())

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("latest release request returned status %d", resp.StatusCode)
	}

	finalPath := resp.Request.URL.Path
	if !strings.Contains(finalPath, "/releases/tag/") {
		return "", fmt.Errorf("unexpected latest release path %q", finalPath)
	}

	tag := path.Base(finalPath)
	if tag == "" || tag == "." || tag == "latest" {
		return "", fmt.Errorf("could not determine latest release tag from %q", finalPath)
	}

	return tag, nil
}

func downloadReleaseProbeFile(ctx context.Context, client *http.Client, tag string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, compileUpdateCheckProbeURLFunc(tag), nil)
	if err != nil {
		return false, fmt.Errorf("failed to create probe request for %s: %w", tag, err)
	}
	req.Header.Set("User-Agent", "gh-aw/"+GetVersion())

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to download probe file for %s: %w", tag, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("probe download for %s returned status %d", tag, resp.StatusCode)
	}
}

func getCompileUpdateCheckFilePath() string {
	return getCompileUpdateCheckFilePathFunc()
}

func getCompileUpdateCheckFilePathImpl() string {
	return getLastCheckFilePathFor(compileUpdateCheckFileName)
}

func updateCompileUpdateCheckTime() {
	lastCheckFile := getCompileUpdateCheckFilePath()
	if lastCheckFile == "" {
		return
	}

	timestamp := time.Now().Format(time.RFC3339)
	if err := os.WriteFile(lastCheckFile, []byte(timestamp), constants.FilePermSensitive); err != nil {
		compileUpdateCheckLog.Printf("Error writing compile update check time: %v", err)
	}
}

func isMinorVersionBehind(currentVersion string, latestVersion string) bool {
	currentSV := ensureSemverPrefix(currentVersion)
	latestSV := ensureSemverPrefix(latestVersion)

	if !semver.IsValid(currentSV) || !semver.IsValid(latestSV) {
		return false
	}
	if semver.Compare(currentSV, latestSV) >= 0 {
		return false
	}

	currentMajor, currentMinor, ok := semverMajorMinorParts(currentSV)
	if !ok {
		return false
	}
	latestMajor, latestMinor, ok := semverMajorMinorParts(latestSV)
	if !ok {
		return false
	}

	return currentMajor == latestMajor && latestMinor > currentMinor
}

func semverMajorMinorParts(version string) (int, int, bool) {
	trimmed := strings.TrimPrefix(version, "v")
	trimmed = strings.SplitN(trimmed, "-", 2)[0]
	trimmed = strings.SplitN(trimmed, "+", 2)[0]

	parts := strings.Split(trimmed, ".")
	if len(parts) < 2 {
		return 0, 0, false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}

	return major, minor, true
}

func ensureSemverPrefix(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func printCompileUpdateNotification(notification *compileUpdateNotification) {
	if notification == nil {
		return
	}

	fmt.Fprintln(os.Stderr)

	switch notification.Kind {
	case compileUpdateNotificationRemovedTag:
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf(
			"The installed gh-aw compiler version %s is no longer available as a repository tag.", notification.CurrentVersion,
		)))
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf(
			"Update the compiler before recompiling workflows (latest release: %s).", notification.LatestVersion,
		)))
	default:
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf(
			"Compiler upgrade recommended: gh-aw %s is behind the latest release %s.", notification.CurrentVersion, notification.LatestVersion,
		)))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Hint: upgrade the compiler with: gh extension upgrade github/gh-aw"))
	}

	fmt.Fprintln(os.Stderr)
}
