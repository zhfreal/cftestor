package outbound

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"cftestor/internal/config"
)

func TestResolveLocationsParallelCappedAtCandidateCount(t *testing.T) {
	ip1, ip2, ip3 := "1.1.1.1", "1.0.0.1", "1.0.0.2"
	locEmpty := ""
	locExisting := "SJC"

	results := []config.VerifyResults{
		{IP: &ip1, Loc: &locEmpty},
		{IP: &ip2, Loc: &locExisting}, // already resolved, must be skipped
		{IP: &ip3, Loc: &locEmpty},
	}

	var activeThreads int32
	var maxObservedThreads int32

	// Request concurrency 10, but only 2 candidates need resolution
	concurrency := 10
	mockResolver := func(ip *string) string {
		cur := atomic.AddInt32(&activeThreads, 1)
		for {
			oldMax := atomic.LoadInt32(&maxObservedThreads)
			if cur <= oldMax || atomic.CompareAndSwapInt32(&maxObservedThreads, oldMax, cur) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&activeThreads, -1)
		return "TEST_" + *ip
	}

	ResolveLocationsParallelWithResolver(results, concurrency, mockResolver)

	if maxObservedThreads > 2 {
		t.Errorf("expected max threads capped at candidate count (2), got %d", maxObservedThreads)
	}
	if *results[0].Loc != "TEST_1.1.1.1" {
		t.Errorf("expected results[0].Loc to be TEST_1.1.1.1, got %s", *results[0].Loc)
	}
	if *results[1].Loc != "SJC" {
		t.Errorf("expected results[1].Loc to remain SJC, got %s", *results[1].Loc)
	}
	if *results[2].Loc != "TEST_1.0.0.2" {
		t.Errorf("expected results[2].Loc to be TEST_1.0.0.2, got %s", *results[2].Loc)
	}
}

func TestResolveLocationsParallelRespectsConcurrencyLimit(t *testing.T) {
	// Pre-create distinct IP strings to avoid closure issues
	ips := make([]string, 8)
	locs := make([]string, 8)
	var results []config.VerifyResults
	for i := 0; i < 8; i++ {
		ips[i] = fmt.Sprintf("198.51.100.%d", i)
		locs[i] = ""
		results = append(results, config.VerifyResults{
			IP:  &ips[i],
			Loc: &locs[i],
		})
	}

	var activeThreads int32
	var maxObservedThreads int32

	concurrency := 3
	mockResolver := func(ip *string) string {
		cur := atomic.AddInt32(&activeThreads, 1)
		for {
			oldMax := atomic.LoadInt32(&maxObservedThreads)
			if cur <= oldMax || atomic.CompareAndSwapInt32(&maxObservedThreads, oldMax, cur) {
				break
			}
		}
		time.Sleep(15 * time.Millisecond)
		atomic.AddInt32(&activeThreads, -1)
		return "LOC_" + *ip
	}

	ResolveLocationsParallelWithResolver(results, concurrency, mockResolver)

	if maxObservedThreads > 3 {
		t.Errorf("expected max threads not to exceed concurrency limit (3), got %d", maxObservedThreads)
	}
	for i := range results {
		want := fmt.Sprintf("LOC_198.51.100.%d", i)
		if results[i].Loc == nil || *results[i].Loc != want {
			t.Errorf("results[%d].Loc = %v, want %s", i, results[i].Loc, want)
		}
	}
}

func TestResolveLocationsParallelEmptyAndNil(t *testing.T) {
	// Should not panic on empty slice
	ResolveLocationsParallelWithResolver([]config.VerifyResults{}, 5, nil)

	// Should not panic on nil IP or nil Loc
	results := []config.VerifyResults{
		{IP: nil, Loc: nil},
	}
	ResolveLocationsParallelWithResolver(results, 5, nil)
}
