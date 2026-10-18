package taint

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

//go:embed javascript/native.cjs
var nativeTypeScriptBridge string

// TypeScriptParser uses only the upstream-exported native-preview API.
// ModuleDirectory must contain an explicitly installed, exactly pinned
// @typescript/native-preview@7.0.0-dev.20260707.2. Nothing is downloaded at runtime.
type TypeScriptParser struct {
	ModuleDirectory string
	NodeExecutable  string
	Environment     []string
	once            sync.Once
	gate            chan struct{}
	cache           map[[32]byte]*JavaScriptNode
	failures        map[[32]byte]error
}

func (p *TypeScriptParser) Parse(ctx context.Context, source string) (*JavaScriptNode, error) {
	results, err := p.ParseBatch(ctx, []string{source})
	if err != nil {
		return nil, err
	}
	result, exists := sliceAt(results, 0)
	if !exists {
		return nil, errors.New("native TypeScript compiler returned an invalid batch")
	}
	return result.Node, result.Err
}

func (p *TypeScriptParser) ParseBatch(ctx context.Context, sources []string) ([]JavaScriptParseResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.once.Do(func() { p.gate = make(chan struct{}, 1) })
	select {
	case p.gate <- struct{}{}:
		defer func() { <-p.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var pending []string
	size := 0
	seen := make(map[[32]byte]bool)
	for _, source := range sources {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		key := sha256.Sum256([]byte(source))
		if len(source) > 1024*1024 || p.cache[key] != nil || p.failures[key] != nil || seen[key] {
			continue
		}
		if len(pending) == 64 || size+len(source) > 2*1024*1024 {
			if err := p.cacheNativeBatch(ctx, pending); err != nil {
				return p.failedBatch(sources, err)
			}
			pending, size = nil, 0
			seen = make(map[[32]byte]bool)
		}
		pending = append(pending, source)
		size += len(source)
		seen[key] = true
	}
	if err := p.cacheNativeBatch(ctx, pending); err != nil {
		return p.failedBatch(sources, err)
	}
	return p.batchResults(sources), nil
}

func (p *TypeScriptParser) cacheNativeBatch(ctx context.Context, sources []string) error {
	if len(sources) == 0 {
		return nil
	}
	results, err := p.parseNativeBatch(ctx, sources)
	if err != nil {
		return err
	}
	if p.cache == nil {
		p.cache = make(map[[32]byte]*JavaScriptNode)
	}
	if p.failures == nil {
		p.failures = make(map[[32]byte]error)
	}
	for i, result := range results {
		source, exists := sliceAt(sources, i)
		if !exists {
			return errors.New("native TypeScript compiler returned an invalid batch")
		}
		key := sha256.Sum256([]byte(source))
		if result.Err != nil {
			p.failures[key] = result.Err
		} else {
			p.cache[key] = result.Node
		}
	}
	return nil
}

func (p *TypeScriptParser) failedBatch(sources []string, err error) ([]JavaScriptParseResult, error) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}
	if p.failures == nil {
		p.failures = make(map[[32]byte]error)
	}
	for _, source := range sources {
		key := sha256.Sum256([]byte(source))
		if p.cache[key] == nil && p.failures[key] == nil {
			p.failures[key] = err
		}
	}
	return p.batchResults(sources), nil
}

func (p *TypeScriptParser) batchResults(sources []string) []JavaScriptParseResult {
	results := make([]JavaScriptParseResult, 0, len(sources))
	for _, source := range sources {
		key := sha256.Sum256([]byte(source))
		result := JavaScriptParseResult{Node: p.cache[key], Err: p.failures[key]}
		if len(source) > 1024*1024 {
			result.Err = errors.New("JavaScript source exceeds analysis size limit")
		}
		results = append(results, result)
	}
	return results
}

func (p *TypeScriptParser) parseNativeBatch(ctx context.Context, sources []string) ([]JavaScriptParseResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.ModuleDirectory == "" {
		return nil, errors.New("JavaScript analysis requires an explicit native compiler module directory")
	}
	directory, err := filepath.Abs(p.ModuleDirectory)
	if err != nil {
		return nil, errors.New("could not resolve native compiler module directory")
	}
	node := p.NodeExecutable
	if node == "" {
		node = "node"
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", nativeTypeScriptBridge, directory)
	// Do not pass credentials, NODE_OPTIONS, or inherited preload hooks into
	// the parser process. Only platform executable lookup variables are needed.
	cmd.Env = append([]string{}, p.Environment...)
	request, err := json.Marshal(struct {
		Sources []string `json:"sources"`
	}{sources})
	if err != nil {
		return nil, errors.New("could not encode native compiler request")
	}
	cmd.Stdin = bytes.NewReader(request)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("native TypeScript AST unavailable or source could not be parsed; install the pinned native-preview module and Node.js")
	}
	var response []struct {
		Tree  *JavaScriptNode `json:"tree"`
		Error string          `json:"error"`
	}
	if err := json.Unmarshal(output, &response); err != nil || len(response) != len(sources) {
		return nil, errors.New("native TypeScript compiler returned an invalid AST")
	}
	results := make([]JavaScriptParseResult, 0, len(sources))
	for _, item := range response {
		var result JavaScriptParseResult
		if item.Error != "" {
			result.Err = errors.New("native TypeScript AST unavailable or source could not be parsed")
		} else if item.Tree == nil || item.Tree.Kind != "SourceFile" {
			return nil, errors.New("native TypeScript compiler returned an invalid AST")
		} else {
			result.Node = item.Tree
		}
		results = append(results, result)
	}
	return results, nil
}
