package typeutil

import "math"

// SafeAllocationCapacity returns the summed capacity hint when it fits in int.
// When the total would overflow, it falls back to 0 so callers can skip
// preallocation without changing correctness. The helper is intentionally
// side-effect free so utility callers do not inherit logging dependencies.
func SafeAllocationCapacity(parts ...int) int {
	total := 0
	for _, part := range parts {
		if part < 0 || total > math.MaxInt-part {
			return 0
		}
		total += part
	}
	return total
}
