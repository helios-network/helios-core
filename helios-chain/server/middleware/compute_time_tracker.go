package middleware

import (
	"sync"
	"time"
)

// ComputeTimeEntry represents a single compute time entry
type ComputeTimeEntry struct {
	Timestamp time.Time
	Duration  time.Duration
}

// ComputeTimeTracker tracks compute time per IP within a sliding window
// and predicts execution time based on method history
type ComputeTimeTracker struct {
	ipEntries         map[string][]ComputeTimeEntry // Store time and duration together
	methodAverages    map[string]time.Duration      // Track average execution time per method
	mutex             sync.RWMutex
	limit             time.Duration
	window            time.Duration
	defaultMethodTime time.Duration // Default time if no history available
}

// NewComputeTimeTracker creates a new compute time tracker
func NewComputeTimeTracker(limit, window time.Duration) *ComputeTimeTracker {
	return &ComputeTimeTracker{
		ipEntries:         make(map[string][]ComputeTimeEntry),
		methodAverages:    make(map[string]time.Duration),
		limit:             limit,
		window:            window,
		defaultMethodTime: 1 * time.Millisecond, // Default 1 second if no history
	}
}

// PredictComputeTime predicts if a request will exceed the compute time limit
// Returns true if the request should be allowed, false if it should be blocked
func (ctt *ComputeTimeTracker) PredictComputeTime(ip, method string) bool {
	// Minimize lock time by copying data quickly
	var currentTotalTime time.Duration
	var predictedTime time.Duration

	func() {
		ctt.mutex.RLock()
		defer ctt.mutex.RUnlock()

		// Quick copy of data under lock
		entries, exists := ctt.ipEntries[ip]
		if !exists {
			return
		}

		// Calculate current total compute time within window
		now := time.Now()
		cutoff := now.Add(-ctt.window)

		for _, entry := range entries {
			if entry.Timestamp.After(cutoff) {
				currentTotalTime += entry.Duration
			}
		}

		// Get predicted time for this method
		if avg, exists := ctt.methodAverages[method]; exists && avg > 0 {
			predictedTime = avg
		} else {
			predictedTime = ctt.defaultMethodTime
		}
	}()

	// Check if adding this request would exceed the limit
	return (currentTotalTime + predictedTime) <= ctt.limit
}

// AddComputeTime adds compute time for an IP and returns true if within limit
func (ctt *ComputeTimeTracker) AddComputeTime(ip string, duration time.Duration) bool {
	// Minimize lock time by doing calculations outside the lock
	var totalTime time.Duration
	var shouldAdd bool

	func() {
		ctt.mutex.Lock()
		defer ctt.mutex.Unlock()

		// Quick access and modification under lock
		entries, exists := ctt.ipEntries[ip]
		if !exists {
			entries = []ComputeTimeEntry{}
		}

		// Add current entry
		entries = append(entries, ComputeTimeEntry{
			Timestamp: time.Now(),
			Duration:  duration,
		})
		ctt.ipEntries[ip] = entries

		// Quick calculation of total time
		now := time.Now()
		cutoff := now.Add(-ctt.window)

		for _, entry := range entries {
			if entry.Timestamp.After(cutoff) {
				totalTime += entry.Duration
			}
		}

		shouldAdd = totalTime <= ctt.limit
	}()

	return shouldAdd
}

// UpdateMethodAverage updates the average execution time for a method using EMA
func (ctt *ComputeTimeTracker) UpdateMethodAverage(method string, duration time.Duration) {
	ctt.mutex.Lock()
	defer ctt.mutex.Unlock()

	// Use Exponential Moving Average (EMA) with alpha = 0.3
	alpha := 0.3
	currentAvg := ctt.methodAverages[method]

	if currentAvg == 0 {
		// First time seeing this method, use the duration directly
		ctt.methodAverages[method] = duration
	} else {
		// Update using EMA formula: new_avg = alpha * new_value + (1-alpha) * old_avg
		newAvg := time.Duration(float64(duration)*alpha + float64(currentAvg)*(1-alpha))
		ctt.methodAverages[method] = newAvg
	}
}

// GetMetrics returns metrics for the compute time tracker
func (ctt *ComputeTimeTracker) GetMetrics() map[string]interface{} {
	ctt.mutex.RLock()
	defer ctt.mutex.RUnlock()

	now := time.Now()
	cutoff := now.Add(-ctt.window)

	// Count active IPs and their compute times
	activeIPs := 0
	totalComputeTime := time.Duration(0)

	for ip, entries := range ctt.ipEntries {
		validEntries := 0
		for _, entry := range entries {
			if entry.Timestamp.After(cutoff) {
				validEntries++
			}
		}
		if validEntries > 0 {
			activeIPs++
			// Use the actual IP for logging or other purposes
			_ = ip // Avoid unused variable warning
			totalComputeTime += time.Duration(validEntries) * ctt.defaultMethodTime
		}
	}

	return map[string]interface{}{
		"limit":               ctt.limit.String(),
		"window_duration":     ctt.window.String(),
		"active_ips":          activeIPs,
		"total_compute_time":  totalComputeTime.String(),
		"default_method_time": ctt.defaultMethodTime.String(),
		"method_averages":     ctt.methodAverages,
	}
}

// Reset clears compute time data for a specific IP
func (ctt *ComputeTimeTracker) Reset(ip string) {
	ctt.mutex.Lock()
	defer ctt.mutex.Unlock()

	if ip == "" {
		// Reset all IPs
		ctt.ipEntries = make(map[string][]ComputeTimeEntry)
		ctt.methodAverages = make(map[string]time.Duration)
	} else {
		// Reset specific IP
		delete(ctt.ipEntries, ip)
	}
}
