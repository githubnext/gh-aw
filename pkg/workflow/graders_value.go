package workflow

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/gitutil"
)

const maxValueFunctionSize = 64 * 1024

func (c *Compiler) prepareValueGrader(data *WorkflowData, markdownPath string) error {
	if data == nil || data.Graders == nil {
		return nil
	}
	grader, ok := data.Graders.Graders["value"]
	if !ok || (grader.Enabled != nil && !*grader.Enabled) {
		return nil
	}
	if grader.Function == "" {
		return errors.New("graders.value requires a 'function' field")
	}

	repoRoot, err := gitutil.FindGitRootFrom(filepath.Dir(markdownPath))
	if err != nil {
		return fmt.Errorf("cannot resolve graders.value.function %q: workflow is not inside a Git repository", grader.Function)
	}
	functionPath := filepath.Join(repoRoot, filepath.FromSlash(grader.Function))
	if err := fileutil.ValidatePathWithinBase(repoRoot, functionPath); err != nil {
		return fmt.Errorf("graders.value.function %q escapes the Git repository", grader.Function)
	}

	file, err := os.Open(functionPath)
	if err != nil {
		return fmt.Errorf("cannot read graders.value.function %q: %w", grader.Function, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("cannot inspect graders.value.function %q: %w", grader.Function, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("graders.value.function %q must be a regular file", grader.Function)
	}
	if info.Size() > maxValueFunctionSize {
		return fmt.Errorf("graders.value.function %q exceeds the %d-byte limit", grader.Function, maxValueFunctionSize)
	}

	content, err := io.ReadAll(io.LimitReader(file, maxValueFunctionSize+1))
	if err != nil {
		return fmt.Errorf("cannot read graders.value.function %q: %w", grader.Function, err)
	}
	if len(content) > maxValueFunctionSize {
		return fmt.Errorf("graders.value.function %q exceeds the %d-byte limit", grader.Function, maxValueFunctionSize)
	}
	if !utf8.Valid(content) {
		return fmt.Errorf("graders.value.function %q must be valid UTF-8", grader.Function)
	}
	functionContent := string(content)
	if !strings.HasPrefix(functionContent, "#!/usr/bin/env bash\n") && !strings.HasPrefix(functionContent, "#!/bin/bash\n") {
		return fmt.Errorf("graders.value.function %q must start with a Bash shebang", grader.Function)
	}

	grader.functionContent = functionContent
	return nil
}
