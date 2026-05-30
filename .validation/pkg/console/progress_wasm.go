//go:build js || wasm

package console

import "fmt"

type ProgressBar struct {
	total         int64
	current       int64
	indeterminate bool
	updateCount   int64
}

func NewProgressBar(total int64) *ProgressBar {
	return &ProgressBar{total: total}
}

func NewIndeterminateProgressBar() *ProgressBar {
	return &ProgressBar{indeterminate: true}
}

func (p *ProgressBar) Update(current int64) string {
	p.current = current
	p.updateCount++
	if p.indeterminate {
		if current == 0 {
			return "Processing..."
		}
		return fmt.Sprintf("Processing... (%s)", formatBytes(current))
	}
	if p.total == 0 {
		return "100% (0B/0B)"
	}
	percent := float64(current) / float64(p.total)
	return fmt.Sprintf("%d%% (%s/%s)", int(percent*100), formatBytes(current), formatBytes(p.total))
}
