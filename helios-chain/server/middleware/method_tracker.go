package middleware

import (
	"sync"
	"time"
)

// MethodTracker tracks method calls and their response times
type MethodTracker struct {
	methods            map[string]*MethodStats
	mutex              sync.RWMutex
	computeTimeTracker *ComputeTimeTracker // Reference to compute time tracker
}

// MethodStats contains statistics for a specific method
type MethodStats struct {
	TotalCalls   int64         `json:"total_calls"`
	TotalTime    time.Duration `json:"total_time"`
	AverageTime  time.Duration `json:"average_time"`
	MinTime      time.Duration `json:"min_time"`
	MaxTime      time.Duration `json:"max_time"`
	LastCallTime time.Time     `json:"last_call_time"`
	ErrorCount   int64         `json:"error_count"`
}

// NewMethodTracker creates a new method tracker
func NewMethodTracker() *MethodTracker {
	return &MethodTracker{
		methods: make(map[string]*MethodStats),
	}
}

// SetComputeTimeTracker sets the reference to the compute time tracker
func (mt *MethodTracker) SetComputeTimeTracker(ctt *ComputeTimeTracker) {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()
	mt.computeTimeTracker = ctt
}

// TrackMethod tracks a method call and its response time
func (mt *MethodTracker) TrackMethod(method string, duration time.Duration, isError bool) {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()

	stats, exists := mt.methods[method]
	if !exists {
		stats = &MethodStats{
			MinTime: duration,
			MaxTime: duration,
		}
		mt.methods[method] = stats
	}

	// Update statistics
	stats.TotalCalls++
	stats.TotalTime += duration
	stats.AverageTime = time.Duration(stats.TotalTime.Nanoseconds() / stats.TotalCalls)
	stats.LastCallTime = time.Now()

	if duration < stats.MinTime {
		stats.MinTime = duration
	}
	if duration > stats.MaxTime {
		stats.MaxTime = duration
	}

	if isError {
		stats.ErrorCount++
	}

	// Update compute time tracker with method average
	if mt.computeTimeTracker != nil {
		mt.computeTimeTracker.UpdateMethodAverage(method, stats.AverageTime)
	}
}

// GetMethodStats returns statistics for a specific method
func (mt *MethodTracker) GetMethodStats(method string) *MethodStats {
	mt.mutex.RLock()
	defer mt.mutex.RUnlock()

	if stats, exists := mt.methods[method]; exists {
		// Create a copy to avoid race conditions
		return &MethodStats{
			TotalCalls:   stats.TotalCalls,
			TotalTime:    stats.TotalTime,
			AverageTime:  stats.AverageTime,
			MinTime:      stats.MinTime,
			MaxTime:      stats.MaxTime,
			LastCallTime: stats.LastCallTime,
			ErrorCount:   stats.ErrorCount,
		}
	}
	return nil
}

// GetAllMethodStats returns statistics for all methods
func (mt *MethodTracker) GetAllMethodStats() map[string]*MethodStats {
	mt.mutex.RLock()
	defer mt.mutex.RUnlock()

	result := make(map[string]*MethodStats)
	for method, stats := range mt.methods {
		// Create a copy to avoid race conditions
		result[method] = &MethodStats{
			TotalCalls:   stats.TotalCalls,
			TotalTime:    stats.TotalTime,
			AverageTime:  stats.AverageTime,
			MinTime:      stats.MinTime,
			MaxTime:      stats.MaxTime,
			LastCallTime: stats.LastCallTime,
			ErrorCount:   stats.ErrorCount,
		}
	}
	return result
}

// Reset clears all method tracking data
func (mt *MethodTracker) Reset() {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()
	mt.methods = make(map[string]*MethodStats)
}
