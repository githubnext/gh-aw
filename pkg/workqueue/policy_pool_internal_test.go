package workqueue

import (
	"strings"
	"testing"
)

func validPoolPolicy() PoolPolicy {
	return PoolPolicy{
		DefaultProfile:      "default",
		LogicalLimit:        1,
		NativeLimit:         1,
		PerAccountLimit:     0,
		MaxObservationAgeMS: 1,
		Retry:               RetryPolicy{MaxAttempts: 1, BackoffMS: 1},
		Reconciliation:      ReconciliationPolicy{MaxAttempts: 1, DeadlineMS: 1},
		Profiles: map[string]WorkerProfile{
			"default": {
				Workflow:        ".github/workflows/worker.lock.yml",
				Ref:             strings.Repeat("a", 40),
				Principal:       "1001",
				TrustDomain:     "default",
				CredentialScope: "repository",
				EffectScope:     "owner/repo",
				MaxClaims:       1,
			},
		},
	}
}

func TestValidatePoolPolicy(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		limits        Limits
		update        func(*PoolPolicy)
		valid         bool
	}{
		{name: "valid", limits: Limits{Operations: 3}, valid: true},
		{name: "missing default profile", limits: Limits{Operations: 3}, update: func(pool *PoolPolicy) { pool.DefaultProfile = "missing" }},
		{name: "invalid logical contract", limits: Limits{Operations: 3}, update: func(pool *PoolPolicy) {
			profile := pool.Profiles["default"]
			profile.LogicalContract = "not-a-digest"
			pool.Profiles["default"] = profile
		}},
		{name: "non-decimal principal", limits: Limits{Operations: 3}, update: func(pool *PoolPolicy) {
			profile := pool.Profiles["default"]
			profile.Principal = "operator"
			pool.Profiles["default"] = profile
		}},
		{name: "aw requires derived routing", authorization: "aw", limits: Limits{Operations: 3}, update: func(pool *PoolPolicy) {
			profile := pool.Profiles["default"]
			profile.TrustDomain = "foreign"
			pool.Profiles["default"] = profile
		}},
		{name: "invalid claims", limits: Limits{Operations: 3}, update: func(pool *PoolPolicy) {
			profile := pool.Profiles["default"]
			profile.MaxClaims = MaxClaimsPerDispatch + 1
			pool.Profiles["default"] = profile
		}},
		{name: "mutable revision", limits: Limits{Operations: 3}, update: func(pool *PoolPolicy) {
			profile := pool.Profiles["default"]
			profile.Ref = "main"
			pool.Profiles["default"] = profile
		}},
		{name: "unapproved workflow", limits: Limits{Operations: 3}, update: func(pool *PoolPolicy) {
			profile := pool.Profiles["default"]
			profile.Workflow = "workflow.lock.yml"
			pool.Profiles["default"] = profile
		}},
		{name: "insufficient operations", limits: Limits{Operations: 2}},
		{name: "invalid pool limits", limits: Limits{Operations: 3}, update: func(pool *PoolPolicy) { pool.LogicalLimit = 0 }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := validPoolPolicy()
			if test.update != nil {
				test.update(&pool)
			}

			err := validatePoolPolicy(pool, test.limits, test.authorization)
			if test.valid && err != nil {
				t.Fatalf("validatePoolPolicy() error = %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("validatePoolPolicy() accepted invalid pool policy")
			}
		})
	}
}
