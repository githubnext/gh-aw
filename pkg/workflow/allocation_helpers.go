package workflow

import (
	"math"

	"github.com/github/gh-aw/pkg/logger"
)

var allocationLog = logger.New("workflow:allocation_helpers")

// safeAllocationCapacity returns the summed capacity hint when it fits in int.
// When the total would overflow, it falls back to 0 so callers can skip
// preallocation without changing correctness.
func safeAllocationCapacity(parts ...int) int {
	total := 0
	for _, part := range parts {
		if part < 0 || total > math.MaxInt-part {
			allocationLog.Printf("Capacity hint overflow or negative part (part=%d, running total=%d), falling back to 0", part, total)
			return 0
		}
		total += part
	}
	return total
}
