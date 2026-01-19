package rolekit

import (
	"sync"
	"sync/atomic"
	"time"
)

// TransactionMetrics provides transaction performance and failure statistics.
type TransactionMetrics struct {
	TotalTransactions      int64         `json:"total_transactions"`
	SuccessfulTransactions int64         `json:"successful_transactions"`
	FailedTransactions     int64         `json:"failed_transactions"`
	AverageDuration        time.Duration `json:"average_duration"`
	MaxDuration            time.Duration `json:"max_duration"`
	MinDuration            time.Duration `json:"min_duration"`
	LastReset              time.Time     `json:"last_reset"`
}

// transactionMonitor holds the internal transaction monitoring state
type transactionMonitor struct {
	totalCount    int64
	successCount  int64
	failureCount  int64
	totalDuration int64 // nanoseconds
	maxDuration   int64 // nanoseconds
	minDuration   int64 // nanoseconds
	lastReset     time.Time
	mu            sync.RWMutex
}

// newTransactionMonitor creates a new transaction monitor
func newTransactionMonitor() *transactionMonitor {
	return &transactionMonitor{
		minDuration: int64(time.Hour), // Initialize to a large value
		lastReset:   time.Now(),
	}
}

// recordTransaction records a transaction completion with its duration and success status
func (tm *transactionMonitor) recordTransaction(duration time.Duration, success bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	atomic.AddInt64(&tm.totalCount, 1)
	atomic.AddInt64(&tm.totalDuration, int64(duration))

	if success {
		atomic.AddInt64(&tm.successCount, 1)
	} else {
		atomic.AddInt64(&tm.failureCount, 1)
	}

	// Update max duration
	durationNs := int64(duration)
	for {
		current := atomic.LoadInt64(&tm.maxDuration)
		if durationNs <= current || atomic.CompareAndSwapInt64(&tm.maxDuration, current, durationNs) {
			break
		}
	}

	// Update min duration
	for {
		current := atomic.LoadInt64(&tm.minDuration)
		if durationNs >= current || atomic.CompareAndSwapInt64(&tm.minDuration, current, durationNs) {
			break
		}
	}
}

// getMetrics returns the current transaction metrics
func (tm *transactionMonitor) getMetrics() TransactionMetrics {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	total := atomic.LoadInt64(&tm.totalCount)
	success := atomic.LoadInt64(&tm.successCount)
	failure := atomic.LoadInt64(&tm.failureCount)
	totalDur := atomic.LoadInt64(&tm.totalDuration)
	maxDur := atomic.LoadInt64(&tm.maxDuration)
	minDur := atomic.LoadInt64(&tm.minDuration)

	var avgDuration time.Duration
	if total > 0 {
		avgDuration = time.Duration(totalDur / total)
	}

	return TransactionMetrics{
		TotalTransactions:      total,
		SuccessfulTransactions: success,
		FailedTransactions:     failure,
		AverageDuration:        avgDuration,
		MaxDuration:            time.Duration(maxDur),
		MinDuration:            time.Duration(minDur),
		LastReset:              tm.lastReset,
	}
}

// reset resets all metrics
func (tm *transactionMonitor) reset() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	atomic.StoreInt64(&tm.totalCount, 0)
	atomic.StoreInt64(&tm.successCount, 0)
	atomic.StoreInt64(&tm.failureCount, 0)
	atomic.StoreInt64(&tm.totalDuration, 0)
	atomic.StoreInt64(&tm.maxDuration, 0)
	atomic.StoreInt64(&tm.minDuration, int64(time.Hour))
	tm.lastReset = time.Now()
}
