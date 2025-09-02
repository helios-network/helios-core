package middleware

import (
	"sync"
)

// ConnectionLimiter implements a connection limiting mechanism
type ConnectionLimiter struct {
	connections map[string]bool
	mutex       sync.RWMutex
	maxConn     int
}

// NewConnectionLimiter creates a new connection limiter
func NewConnectionLimiter(maxConnections int) *ConnectionLimiter {
	return &ConnectionLimiter{
		connections: make(map[string]bool),
		maxConn:     maxConnections,
	}
}

// AllowConnection checks if a new connection from the given IP is allowed
func (cl *ConnectionLimiter) AllowConnection(ip string) bool {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	if len(cl.connections) >= cl.maxConn {
		return false
	}

	cl.connections[ip] = true
	return true
}

// RemoveConnection removes a connection for the given IP
func (cl *ConnectionLimiter) RemoveConnection(ip string) {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	delete(cl.connections, ip)
}

// GetConnectionCount returns the current number of connections
func (cl *ConnectionLimiter) GetConnectionCount() int {
	cl.mutex.RLock()
	defer cl.mutex.RUnlock()

	return len(cl.connections)
}

// GetMetrics returns metrics about the connection limiter
func (cl *ConnectionLimiter) GetMetrics() map[string]interface{} {
	cl.mutex.RLock()
	defer cl.mutex.RUnlock()

	return map[string]interface{}{
		"current_connections": len(cl.connections),
		"max_connections":     cl.maxConn,
		"available_slots":     cl.maxConn - len(cl.connections),
	}
}
