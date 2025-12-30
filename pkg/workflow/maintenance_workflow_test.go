package workflow

import (
	"testing"
)

func TestGenerateMaintenanceCron(t *testing.T) {
	tests := []struct {
		name             string
		minExpiresDays   int
		expectedCron     string
		expectedDesc     string
	}{
		{
			name:             "1 day or less - every 2 hours",
			minExpiresDays:   1,
			expectedCron:     "37 */2 * * *",
			expectedDesc:     "Every 2 hours",
		},
		{
			name:             "2 days - every 6 hours",
			minExpiresDays:   2,
			expectedCron:     "37 */6 * * *",
			expectedDesc:     "Every 6 hours",
		},
		{
			name:             "3 days - every 12 hours",
			minExpiresDays:   3,
			expectedCron:     "37 */12 * * *",
			expectedDesc:     "Every 12 hours",
		},
		{
			name:             "4 days - every 12 hours",
			minExpiresDays:   4,
			expectedCron:     "37 */12 * * *",
			expectedDesc:     "Every 12 hours",
		},
		{
			name:             "5 days - daily",
			minExpiresDays:   5,
			expectedCron:     "37 0 * * *",
			expectedDesc:     "Daily",
		},
		{
			name:             "7 days - daily",
			minExpiresDays:   7,
			expectedCron:     "37 0 * * *",
			expectedDesc:     "Daily",
		},
		{
			name:             "30 days - daily",
			minExpiresDays:   30,
			expectedCron:     "37 0 * * *",
			expectedDesc:     "Daily",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cron, desc := generateMaintenanceCron(tt.minExpiresDays)
			if cron != tt.expectedCron {
				t.Errorf("generateMaintenanceCron(%d) cron = %q, expected %q", tt.minExpiresDays, cron, tt.expectedCron)
			}
			if desc != tt.expectedDesc {
				t.Errorf("generateMaintenanceCron(%d) desc = %q, expected %q", tt.minExpiresDays, desc, tt.expectedDesc)
			}
		})
	}
}
