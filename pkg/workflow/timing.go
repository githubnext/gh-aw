package workflow

import (
	"fmt"
	"os"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var timingLog = logger.New("workflow:timing")

// TimingTracker tracks the duration of compilation steps for performance analysis
type TimingTracker struct {
	verbose     bool
	startTime   time.Time
	steps       []TimingStep
	currentStep *TimingStep
}

// TimingStep represents a single timed step in the compilation process
type TimingStep struct {
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	SubSteps   []TimingStep
	currentSub *TimingStep
}

// NewTimingTracker creates a new timing tracker
func NewTimingTracker(verbose bool) *TimingTracker {
	return &TimingTracker{
		verbose:   verbose,
		startTime: time.Now(),
		steps:     make([]TimingStep, 0),
	}
}

// StartStep starts timing a new compilation step
func (t *TimingTracker) StartStep(stepName string) {
	if !t.verbose {
		return
	}

	now := time.Now()
	step := TimingStep{
		Name:      stepName,
		StartTime: now,
		SubSteps:  make([]TimingStep, 0),
	}

	timingLog.Printf("Starting step: %s", stepName)

	// End the previous step if there is one
	if t.currentStep != nil {
		t.EndStep()
	}

	t.currentStep = &step
}

// StartSubStep starts timing a sub-step within the current step
func (t *TimingTracker) StartSubStep(subStepName string) {
	if !t.verbose || t.currentStep == nil {
		return
	}

	now := time.Now()
	subStep := TimingStep{
		Name:      subStepName,
		StartTime: now,
		SubSteps:  make([]TimingStep, 0),
	}

	timingLog.Printf("Starting sub-step: %s", subStepName)

	// End the previous sub-step if there is one
	if t.currentStep.currentSub != nil {
		t.EndSubStep()
	}

	t.currentStep.currentSub = &subStep
}

// EndSubStep ends timing the current sub-step
func (t *TimingTracker) EndSubStep() {
	if !t.verbose || t.currentStep == nil || t.currentStep.currentSub == nil {
		return
	}

	now := time.Now()
	t.currentStep.currentSub.EndTime = now
	t.currentStep.currentSub.Duration = now.Sub(t.currentStep.currentSub.StartTime)

	timingLog.Printf("Completed sub-step: %s (took %v)",
		t.currentStep.currentSub.Name, t.currentStep.currentSub.Duration)

	// Add the completed sub-step to the current step
	t.currentStep.SubSteps = append(t.currentStep.SubSteps, *t.currentStep.currentSub)
	t.currentStep.currentSub = nil
}

// EndStep ends timing the current step
func (t *TimingTracker) EndStep() {
	if !t.verbose || t.currentStep == nil {
		return
	}

	now := time.Now()

	// End any pending sub-step
	if t.currentStep.currentSub != nil {
		t.EndSubStep()
	}

	t.currentStep.EndTime = now
	t.currentStep.Duration = now.Sub(t.currentStep.StartTime)

	timingLog.Printf("Completed step: %s (took %v)", t.currentStep.Name, t.currentStep.Duration)

	// Add the completed step to the list
	t.steps = append(t.steps, *t.currentStep)
	t.currentStep = nil
}

// PrintSummary displays a summary of all timing information
func (t *TimingTracker) PrintSummary() {
	if !t.verbose {
		return
	}

	// End any pending step
	if t.currentStep != nil {
		t.EndStep()
	}

	totalDuration := time.Since(t.startTime)

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(""))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("=== Compilation Timing Summary ==="))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Total compilation time: %v", formatDuration(totalDuration))))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(""))

	if len(t.steps) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No timing steps recorded"))
		return
	}

	for i, step := range t.steps {
		percentage := float64(step.Duration) / float64(totalDuration) * 100
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
			fmt.Sprintf("%d. %s: %v (%.1f%%)",
				i+1, step.Name, formatDuration(step.Duration), percentage)))

		// Print sub-steps if any
		for j, subStep := range step.SubSteps {
			subPercentage := float64(subStep.Duration) / float64(step.Duration) * 100
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
				fmt.Sprintf("   %d.%d %s: %v (%.1f%% of step)",
					i+1, j+1, subStep.Name, formatDuration(subStep.Duration), subPercentage)))
		}
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(""))
}

// GetTotalDuration returns the total time elapsed since the tracker was created
func (t *TimingTracker) GetTotalDuration() time.Duration {
	return time.Since(t.startTime)
}

// formatDuration formats a duration for human-readable display
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dμs", d.Microseconds())
	} else if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1_000_000)
	} else if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	} else {
		minutes := int(d.Minutes())
		seconds := d.Seconds() - float64(minutes*60)
		return fmt.Sprintf("%dm %.1fs", minutes, seconds)
	}
}

// TimeStep is a helper function to measure and log the duration of a function call
func (t *TimingTracker) TimeStep(stepName string, fn func() error) error {
	t.StartStep(stepName)
	defer t.EndStep()
	return fn()
}

// TimeSubStep is a helper function to measure and log the duration of a sub-step function call
func (t *TimingTracker) TimeSubStep(subStepName string, fn func() error) error {
	t.StartSubStep(subStepName)
	defer t.EndSubStep()
	return fn()
}
