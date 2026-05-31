package console

import (
	"fmt"
	"sync"
	"time"
)

var (
	timeLocationMu sync.RWMutex
	timeLocation   *time.Location
)

// SetTimeLocation configures the location used when rendering time.Time values.
func SetTimeLocation(location *time.Location) {
	timeLocationMu.Lock()
	defer timeLocationMu.Unlock()
	timeLocation = location
}

// ResetTimeLocation clears any configured location override for rendered times.
func ResetTimeLocation() {
	SetTimeLocation(nil)
}

func currentTimeLocation() *time.Location {
	timeLocationMu.RLock()
	defer timeLocationMu.RUnlock()
	return timeLocation
}

func formatConfiguredTimeValue(timeVal time.Time) string {
	location := currentTimeLocation()
	if location == nil {
		return timeVal.Format("2006-01-02 15:04:05")
	}

	localTime := timeVal.In(location)
	_, offsetSeconds := localTime.Zone()
	return fmt.Sprintf("%s UTC%s", localTime.Format("2006-01-02 15:04:05"), formatUTCOffset(offsetSeconds))
}

func formatUTCOffset(offsetSeconds int) string {
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}

	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}
