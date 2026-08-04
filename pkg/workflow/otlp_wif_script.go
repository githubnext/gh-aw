package workflow

import (
	_ "embed"
	"strings"
)

// exchangeOTLPWorkloadIdentityScript is the runtime implementation of the Google
// Workload Identity Federation token exchange. The source of truth lives in
// actions/setup/js/exchange_otlp_workload_identity.cjs (with vitest coverage);
// this copy is embedded because the exchange step runs before the setup step
// that copies the .cjs files onto the runner.
//
//go:embed js/exchange_otlp_workload_identity.cjs
var exchangeOTLPWorkloadIdentityScript string

// getExchangeOTLPWorkloadIdentityScript returns the inline script body for the
// workload identity exchange github-script step.
func getExchangeOTLPWorkloadIdentityScript() string {
	return strings.TrimSpace(exchangeOTLPWorkloadIdentityScript) + "\nawait main();"
}
