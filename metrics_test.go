package main

import (
	"sync"
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()

	if m.TotalRequests() != 0 {
		t.Errorf("expected 0 total requests, got %d", m.TotalRequests())
	}
	if m.SuccessRequests() != 0 {
		t.Errorf("expected 0 success requests, got %d", m.SuccessRequests())
	}
}

func TestMetricsRecordSuccess(t *testing.T) {
	m := NewMetrics()

	m.RecordSuccess(10 * time.Millisecond)
	m.RecordSuccess(20 * time.Millisecond)
	m.RecordSuccess(30 * time.Millisecond)

	if m.TotalRequests() != 3 {
		t.Errorf("expected 3 total requests, got %d", m.TotalRequests())
	}
	if m.SuccessRequests() != 3 {
		t.Errorf("expected 3 success requests, got %d", m.SuccessRequests())
	}
	if m.FailedRequests() != 0 {
		t.Errorf("expected 0 failed requests, got %d", m.FailedRequests())
	}
}

func TestMetricsRecordFailure(t *testing.T) {
	m := NewMetrics()

	m.RecordFailure(10 * time.Millisecond)
	m.RecordSuccess(20 * time.Millisecond)

	if m.TotalRequests() != 2 {
		t.Errorf("expected 2 total requests, got %d", m.TotalRequests())
	}
	if m.FailedRequests() != 1 {
		t.Errorf("expected 1 failed request, got %d", m.FailedRequests())
	}
}

func TestMetricsAverageLatency(t *testing.T) {
	m := NewMetrics()

	m.RecordSuccess(10 * time.Millisecond)
	m.RecordSuccess(20 * time.Millisecond)
	m.RecordSuccess(30 * time.Millisecond)

	avg := m.AverageLatency()
	expected := 20 * time.Millisecond

	if avg != expected {
		t.Errorf("expected average latency %v, got %v", expected, avg)
	}
}

func TestMetricsErrorRate(t *testing.T) {
	m := NewMetrics()

	m.RecordSuccess(10 * time.Millisecond)
	m.RecordFailure(10 * time.Millisecond)

	rate := m.ErrorRate()
	if rate != 0.5 {
		t.Errorf("expected error rate 0.5, got %f", rate)
	}
}

func TestMetricsP99Latency(t *testing.T) {
	m := NewMetrics()

	for i := 1; i <= 100; i++ {
		m.RecordSuccess(time.Duration(i) * time.Millisecond)
	}

	p99 := m.P99Latency()
	// P99 should be around 99ms or 100ms
	if p99 < 99*time.Millisecond || p99 > 100*time.Millisecond {
		t.Errorf("expected P99 around 99-100ms, got %v", p99)
	}
}

func TestMetricsReset(t *testing.T) {
	m := NewMetrics()

	m.RecordSuccess(10 * time.Millisecond)
	m.RecordSuccess(20 * time.Millisecond)

	m.Reset()

	// Window metrics should be reset
	if m.RPS() != 0 {
		t.Errorf("expected RPS 0 after reset, got %f", m.RPS())
	}

	// But total should remain
	if m.TotalRequests() != 2 {
		t.Errorf("expected total 2 after reset, got %d", m.TotalRequests())
	}
}

func TestMetricsConcurrent(t *testing.T) {
	m := NewMetrics()
	var wg sync.WaitGroup

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				m.RecordSuccess(time.Millisecond)
			}
		}()
	}

	wg.Wait()

	if m.TotalRequests() != 10000 {
		t.Errorf("expected 10000 requests, got %d", m.TotalRequests())
	}
}

func TestMetricsSnapshot(t *testing.T) {
	m := NewMetrics()

	m.RecordSuccess(10 * time.Millisecond)
	m.RecordFailure(20 * time.Millisecond)

	snap := m.Snapshot()

	if snap.TotalRequests != 2 {
		t.Errorf("expected 2 total, got %d", snap.TotalRequests)
	}
	if snap.SuccessRequests != 1 {
		t.Errorf("expected 1 success, got %d", snap.SuccessRequests)
	}
	if snap.FailedRequests != 1 {
		t.Errorf("expected 1 failed, got %d", snap.FailedRequests)
	}
}
