package cli

import (
	"reflect"
	"testing"
)

func TestDomainBucketsAccessors(t *testing.T) {
	buckets := &DomainBuckets{}

	// Test initial state
	if buckets.GetAllowedDomains() != nil {
		t.Errorf("Expected nil allowed domains initially, got %v", buckets.GetAllowedDomains())
	}
	if buckets.GetDeniedDomains() != nil {
		t.Errorf("Expected nil denied domains initially, got %v", buckets.GetDeniedDomains())
	}

	// Test SetAllowedDomains
	allowedDomains := []string{"example.com", "github.com"}
	buckets.SetAllowedDomains(allowedDomains)
	if !reflect.DeepEqual(buckets.GetAllowedDomains(), allowedDomains) {
		t.Errorf("Expected allowed domains %v, got %v", allowedDomains, buckets.GetAllowedDomains())
	}

	// Test SetDeniedDomains
	deniedDomains := []string{"blocked.com", "malicious.com"}
	buckets.SetDeniedDomains(deniedDomains)
	if !reflect.DeepEqual(buckets.GetDeniedDomains(), deniedDomains) {
		t.Errorf("Expected denied domains %v, got %v", deniedDomains, buckets.GetDeniedDomains())
	}
}

func TestDomainBucketsWithEmbedding(t *testing.T) {
	// This test verifies that types embedding DomainBuckets can access
	// the domain accessor methods.
	type TestAnalysis struct {
		DomainBuckets
		TotalRequests int
	}

	analysis := &TestAnalysis{}

	// Test that we can call accessor methods through embedding
	analysis.SetAllowedDomains([]string{"test.com"})
	analysis.SetDeniedDomains([]string{"bad.com"})

	if len(analysis.GetAllowedDomains()) != 1 {
		t.Errorf("Expected 1 allowed domain, got %d", len(analysis.GetAllowedDomains()))
	}
	if len(analysis.GetDeniedDomains()) != 1 {
		t.Errorf("Expected 1 denied domain, got %d", len(analysis.GetDeniedDomains()))
	}
}
