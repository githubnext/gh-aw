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
	mu              sync.Mutex
	cache           map[[32]byte]*JavaScriptNode
}

func (p *TypeScriptParser) Parse(ctx context.Context, source string) (*JavaScriptNode, error) {
	if len(source) > 1024*1024 {
		return nil, errors.New("JavaScript source exceeds analysis size limit")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	key := sha256.Sum256([]byte(source))
	if tree := p.cache[key]; tree != nil {
		return tree, nil
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
		Source string `json:"source"`
	}{source})
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
	var tree JavaScriptNode
	if err := json.Unmarshal(output, &tree); err != nil || tree.Kind != "SourceFile" {
		return nil, errors.New("native TypeScript compiler returned an invalid AST")
	}
	if p.cache == nil {
		p.cache = make(map[[32]byte]*JavaScriptNode)
	}
	p.cache[key] = &tree
	return &tree, nil
}
