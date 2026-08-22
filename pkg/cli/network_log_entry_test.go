package cli

import (
	"encoding/json"
	"testing"
)

func TestNetworkStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		wantCode   int
		wantParsed bool
	}{
		{name: "plain numeric status", status: "200", wantCode: 200, wantParsed: true},
		{name: "numeric status with leading zero", status: "0", wantCode: 0, wantParsed: true},
		{name: "empty status", status: "", wantCode: 0, wantParsed: false},
		{name: "non-numeric status", status: "TCP_MISS", wantCode: 0, wantParsed: false},
		{name: "squid combined decision/status is not parsed", status: "TCP_MISS/200", wantCode: 0, wantParsed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCode, gotParsed := networkStatusCode(tt.status)
			if gotParsed != tt.wantParsed {
				t.Fatalf("networkStatusCode(%q) parsed = %v, want %v", tt.status, gotParsed, tt.wantParsed)
			}
			if gotParsed && gotCode != tt.wantCode {
				t.Fatalf("networkStatusCode(%q) = %d, want %d", tt.status, gotCode, tt.wantCode)
			}
		})
	}
}

func TestNetworkStatusCodeOrZero(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   int
	}{
		{name: "valid numeric status", status: "404", want: 404},
		{name: "invalid status falls back to zero", status: "TCP_MISS", want: 0},
		{name: "empty status falls back to zero", status: "", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := networkStatusCodeOrZero(tt.status); got != tt.want {
				t.Fatalf("networkStatusCodeOrZero(%q) = %d, want %d", tt.status, got, tt.want)
			}
		})
	}
}

func TestNetworkStatusFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "empty raw message", raw: "", want: ""},
		{name: "JSON null", raw: "null", want: ""},
		{name: "plain numeric JSON status", raw: "200", want: "200"},
		{name: "quoted string status", raw: `"407"`, want: "407"},
		{name: "quoted non-numeric string status", raw: `"TCP_MISS"`, want: "TCP_MISS"},
		{name: "negative numeric status is rejected", raw: "-1", wantErr: true},
		{name: "boolean status is rejected", raw: "true", wantErr: true},
		{name: "float status is rejected", raw: "1.2", wantErr: true},
		{name: "object status is rejected", raw: `{"code":200}`, wantErr: true},
		{name: "array status is rejected", raw: `[200]`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := networkStatusFromJSON(json.RawMessage(tt.raw))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("networkStatusFromJSON(%q) expected error, got nil (status=%q)", tt.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("networkStatusFromJSON(%q) unexpected error: %v", tt.raw, err)
			}
			if got != tt.want {
				t.Fatalf("networkStatusFromJSON(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
