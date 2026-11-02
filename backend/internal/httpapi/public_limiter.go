package httpapi

import (
	"net"
	"sync"
	"time"
)

type publicBucket struct {
	until time.Time
	used  int
}
type publicLimiter struct {
	mu              sync.Mutex
	buckets         map[string]publicBucket
	limit, capacity int
	window          time.Duration
}

func newPublicLimiter(limit int, window time.Duration, capacity int) *publicLimiter {
	return &publicLimiter{buckets: map[string]publicBucket{}, limit: limit, capacity: capacity, window: window}
}
func (l *publicLimiter) allow(remote string, now time.Time) bool {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		host = remote
	}
	ip := net.ParseIP(host)
	key := "unknown"
	if ip != nil {
		key = ip.String()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	bucket, exists := l.buckets[key]
	if !exists && len(l.buckets) >= l.capacity {
		for peer, b := range l.buckets {
			if !b.until.After(now) {
				delete(l.buckets, peer)
			}
		}
		if len(l.buckets) >= l.capacity {
			return false
		}
	}
	if !exists || !bucket.until.After(now) {
		bucket = publicBucket{until: now.Add(l.window)}
	}
	if bucket.used >= l.limit {
		return false
	}
	bucket.used++
	l.buckets[key] = bucket
	return true
}
