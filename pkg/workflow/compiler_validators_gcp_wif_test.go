package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateGCPWIFEngineAuth(t *testing.T) {
	t.Parallel()

	completeGCPAuth := &EngineAuthConfig{
		Type:                           "github-oidc",
		Provider:                       "gcp",
		GoogleWorkloadIdentityProvider: "projects/123/locations/global/workloadIdentityPools/pool/providers/provider",
		GoogleServiceAccount:           "sa@project.iam.gserviceaccount.com",
		GoogleProject:                  "my-project",
	}

	tests := []struct {
		name        string
		workflow    *WorkflowData
		wantErr     bool
		errContains string
	}{
		{
			name:     "nil workflow data",
			workflow: nil,
			wantErr:  false,
		},
		{
			name:     "nil engine config",
			workflow: &WorkflowData{EngineConfig: nil},
			wantErr:  false,
		},
		{
			name:     "nil auth config",
			workflow: &WorkflowData{EngineConfig: &EngineConfig{Auth: nil}},
			wantErr:  false,
		},
		{
			name: "auth type not github-oidc",
			workflow: &WorkflowData{EngineConfig: &EngineConfig{Auth: &EngineAuthConfig{
				Type:     "other",
				Provider: "gcp",
			}}},
			wantErr: false,
		},
		{
			name: "auth provider not gcp",
			workflow: &WorkflowData{EngineConfig: &EngineConfig{Auth: &EngineAuthConfig{
				Type:     "github-oidc",
				Provider: "azure",
			}}},
			wantErr: false,
		},
		{
			name:     "complete gcp auth",
			workflow: &WorkflowData{EngineConfig: &EngineConfig{Auth: completeGCPAuth}},
			wantErr:  false,
		},
		{
			name: "missing workload-identity-provider only",
			workflow: &WorkflowData{EngineConfig: &EngineConfig{Auth: &EngineAuthConfig{
				Type:                 "github-oidc",
				Provider:             "gcp",
				GoogleServiceAccount: "sa@project.iam.gserviceaccount.com",
				GoogleProject:        "my-project",
			}}},
			wantErr:     true,
			errContains: "workload-identity-provider",
		},
		{
			name: "missing service-account only",
			workflow: &WorkflowData{EngineConfig: &EngineConfig{Auth: &EngineAuthConfig{
				Type:                           "github-oidc",
				Provider:                       "gcp",
				GoogleWorkloadIdentityProvider: "provider",
				GoogleProject:                  "my-project",
			}}},
			wantErr:     true,
			errContains: "service-account",
		},
		{
			name: "missing project only",
			workflow: &WorkflowData{EngineConfig: &EngineConfig{Auth: &EngineAuthConfig{
				Type:                           "github-oidc",
				Provider:                       "gcp",
				GoogleWorkloadIdentityProvider: "provider",
				GoogleServiceAccount:           "sa@project.iam.gserviceaccount.com",
			}}},
			wantErr:     true,
			errContains: "project",
		},
		{
			name: "missing all three fields",
			workflow: &WorkflowData{EngineConfig: &EngineConfig{Auth: &EngineAuthConfig{
				Type:     "github-oidc",
				Provider: "gcp",
			}}},
			wantErr:     true,
			errContains: "workload-identity-provider, service-account, project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateGCPWIFEngineAuth(tt.workflow)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Contains(t, err.Error(), "engine.auth with provider=gcp requires the following fields")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
