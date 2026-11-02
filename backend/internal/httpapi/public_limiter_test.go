package httpapi

import (
	"testing"
	"time"
)

func TestPublicLimiterBoundsRequestRateAndMemory(t *testing.T) {
	limiter := newPublicLimiter(2, time.Minute, 2)
	now := time.Now()
	if !limiter.allow("127.0.0.1:1", now) || !limiter.allow("127.0.0.1:2", now) || limiter.allow("127.0.0.1:3", now) {
		t.Fatal("changing source port bypassed IP limit")
	}
	if !limiter.allow("127.0.0.2:1", now) || limiter.allow("127.0.0.3:1", now) {
		t.Fatal("bucket memory not bounded")
	}
	if !limiter.allow("127.0.0.3:1", now.Add(time.Minute)) {
		t.Fatal("expired buckets not reclaimed")
	}
}
