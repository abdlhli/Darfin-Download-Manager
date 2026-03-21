package downloader

import (
	"io"
	"sync"
	"time"
)

// SpeedLimiter provides global download speed limiting
type SpeedLimiter struct {
	mu           sync.Mutex
	bytesPerSec  int64
	enabled      bool
	bucket       int64
	lastRefill   time.Time
}

// NewSpeedLimiter creates a new speed limiter
func NewSpeedLimiter(bytesPerSec int64) *SpeedLimiter {
	return &SpeedLimiter{
		bytesPerSec: bytesPerSec,
		enabled:     bytesPerSec > 0,
		bucket:      bytesPerSec,
		lastRefill:  time.Now(),
	}
}

// SetLimit updates the speed limit
func (sl *SpeedLimiter) SetLimit(bytesPerSec int64) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.bytesPerSec = bytesPerSec
	sl.enabled = bytesPerSec > 0
	sl.bucket = bytesPerSec
	sl.lastRefill = time.Now()
}

// IsEnabled returns whether the limiter is active
func (sl *SpeedLimiter) IsEnabled() bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.enabled
}

// Allow waits until n bytes can be consumed, returns how many bytes are allowed
func (sl *SpeedLimiter) Allow(n int) int {
	if !sl.IsEnabled() {
		return n
	}

	sl.mu.Lock()
	defer sl.mu.Unlock()

	// Refill the bucket based on elapsed time
	now := time.Now()
	elapsed := now.Sub(sl.lastRefill).Seconds()
	if elapsed > 0 {
		refill := int64(elapsed * float64(sl.bytesPerSec))
		sl.bucket += refill
		if sl.bucket > sl.bytesPerSec {
			sl.bucket = sl.bytesPerSec
		}
		sl.lastRefill = now
	}

	// If bucket is empty, wait a bit
	if sl.bucket <= 0 {
		sl.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		sl.mu.Lock()
		// Refill after sleep
		now = time.Now()
		elapsed = now.Sub(sl.lastRefill).Seconds()
		refill := int64(elapsed * float64(sl.bytesPerSec))
		sl.bucket += refill
		sl.lastRefill = now
	}

	allowed := int64(n)
	if allowed > sl.bucket {
		allowed = sl.bucket
	}
	if allowed <= 0 {
		allowed = 1
	}

	sl.bucket -= allowed
	return int(allowed)
}

// Reader wraps an io.Reader with speed limiting
func (sl *SpeedLimiter) Reader(r io.Reader) io.Reader {
	if !sl.IsEnabled() {
		return r
	}
	return &limitedReader{reader: r, limiter: sl}
}

type limitedReader struct {
	reader  io.Reader
	limiter *SpeedLimiter
}

func (lr *limitedReader) Read(p []byte) (int, error) {
	allowed := lr.limiter.Allow(len(p))
	return lr.reader.Read(p[:allowed])
}
