// Package workflow provides GitHub Actions expression parsing and manipulation.
//
// # Expression Safety
//
// This file contains expression parsing utilities for GitHub Actions expressions.
// For expression security validation, see expression_validation.go.
//
// # Expression Parsing
//
// This file provides functions to parse and manipulate GitHub Actions expressions:
//   - ParseExpression() - Parses expression strings into expression trees
//   - VisitExpressionTree() - Traverses expression trees for validation
//
// # When to Add Code Here
//
// Add code to this file when:
//   - It parses GitHub Actions expression syntax
//   - It manipulates expression trees
//   - It extracts or transforms expression components
//
// For expression validation, see expression_validation.go.
// For general validation, see validation.go.
package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var expressionSafetyLog = logger.New("workflow:expression_safety")
