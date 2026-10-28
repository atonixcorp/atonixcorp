package middleware

import (
	"testing"
	"time"
)

func TestPrincipalLimiterEnforcesWindowAndExpires(t *testing.T) {
	limiter := &principalLimiter{
		buckets: make(map[uint]rateBucket),
		max:     2,
		window:  time.Minute,
	}
	start := time.Date(2026, 10, 28, 8, 0, 0, 0, time.UTC)

	if _, limited := limiter.take(7, start); limited {
		t.Fatal("first request should pass")
	}
	if _, limited := limiter.take(7, start.Add(time.Second)); limited {
		t.Fatal("second request should pass")
	}
	if retry, limited := limiter.take(7, start.Add(2*time.Second)); !limited || retry <= 0 {
		t.Fatalf("third request should be rate limited with retry delay, got %v, %v", retry, limited)
	}
	if _, limited := limiter.take(7, start.Add(time.Minute)); limited {
		t.Fatal("request after the fixed window should pass")
	}
	if _, limited := limiter.take(8, start.Add(time.Minute)); limited {
		t.Fatal("rate limits should be isolated by authenticated subject")
	}
}

func TestPrincipalLimiterBoundsSubjectTable(t *testing.T) {
	limiter := &principalLimiter{
		buckets: make(map[uint]rateBucket, 10000),
		max:     1,
		window:  time.Minute,
	}
	now := time.Now()
	for id := uint(1); id <= 10000; id++ {
		limiter.buckets[id] = rateBucket{started: now, count: 1}
	}
	if retry, limited := limiter.take(10001, now); !limited || retry != time.Minute {
		t.Fatalf("new subject at capacity should be rejected, got %v, %v", retry, limited)
	}
}
