package taint

import (
	"context"
	"crypto/sha256"
	"errors"

	"github.com/github/gh-aw/pkg/workflow/aw"
)

type javaScriptSourceRecorder struct {
	sources []string
	seen    map[[32]byte]bool
}

func (p *javaScriptSourceRecorder) Parse(ctx context.Context, source string) (*JavaScriptNode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := sha256.Sum256([]byte(source))
	if !p.seen[key] {
		p.seen[key] = true
		p.sources = append(p.sources, source)
	}
	return &JavaScriptNode{Kind: "SourceFile"}, nil
}

// PrimeAuthoredJavaScript batches the deterministically rendered entry sources
// across documents. The discovery pass does not evaluate JavaScript or modules.
// Additional sources discovered through actual data flow are parsed on demand.
func PrimeAuthoredJavaScript(ctx context.Context, documents []*aw.Document, rule Rule, options JavaScriptOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	parser, ok := options.Parser.(JavaScriptBatchParser)
	if !ok {
		return nil
	}
	recorder := &javaScriptSourceRecorder{seen: make(map[[32]byte]bool)}
	options.Parser, options.Context = recorder, ctx
	rule.JavaScript = &options
	var failures []error
	for _, document := range documents {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, err := AnalyzeAuthoredDocument(ctx, document, rule)
		failures = append(failures, err)
	}
	_, err := parser.ParseBatch(ctx, recorder.sources)
	return errors.Join(append(failures, err)...)
}
