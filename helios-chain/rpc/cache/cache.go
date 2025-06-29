package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// RPCCache provides caching functionality for RPC requests
type RPCCache struct {
	blockCache    *lru.Cache[string, *CacheEntry]
	mu            sync.RWMutex
	defaultTTL    time.Duration
	cachedMethods map[string]time.Duration // method name -> ttl
}

// CacheEntry represents a cached item with expiration
type CacheEntry struct {
	Data      interface{} `json:"data"`
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
}

// NewRPCCache creates a new RPC cache instance
func NewRPCCache(blockCacheSize int, defaultTTL time.Duration) (*RPCCache, error) {
	blockCache, err := lru.New[string, *CacheEntry](blockCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create block cache: %w", err)
	}

	cache := &RPCCache{
		blockCache:    blockCache,
		defaultTTL:    defaultTTL,
		cachedMethods: make(map[string]time.Duration),
	}

	// Set default cached methods with 15 seconds TTL
	cache.setDefaultCachedMethods()

	return cache, nil
}

// setDefaultCachedMethods sets up the 4 methods that should be cached
func (c *RPCCache) setDefaultCachedMethods() {
	cachedMethods := map[string]time.Duration{
		"GetAllHyperionTransferTxs":                  15 * time.Second,
		"GetHyperionAccountTransferTxsByPageAndSize": 15 * time.Second,
		"GetValidatorWithHisAssetsAndCommission":     15 * time.Second,
		"GetAllWhitelistedAssets":                    15 * time.Second,
		"GetLastTransactionsInfo":                    15 * time.Second,
		"GetAccountLastTransactionsInfo":             60 * time.Second,
		"GetTokensByChainIdAndPageAndSize":           60 * time.Second,
		"GetActiveValidatorCount":                    60 * time.Second,
		"GetTokenDetails":                            5 * time.Minute,
		"ChainId":                                    5 * time.Second,
		"BlockNumber":                                5 * time.Second,
	}

	for method, ttl := range cachedMethods {
		c.cachedMethods[method] = ttl
	}
}

// IsMethodCached checks if a method should be cached
func (c *RPCCache) IsMethodCached(methodName string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.cachedMethods[methodName]
	return exists
}

// GetMethodTTL returns the TTL for a method
func (c *RPCCache) GetMethodTTL(methodName string) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if ttl, exists := c.cachedMethods[methodName]; exists {
		return ttl
	}
	return c.defaultTTL
}

// InterceptCall intercepts an RPC method call and applies caching if configured
func (c *RPCCache) InterceptCall(methodName string, params []interface{}, fetchFunc func() (interface{}, error)) (interface{}, error) {
	// Check if method should be cached
	if !c.IsMethodCached(methodName) {
		return fetchFunc()
	}

	// Generate cache key
	cacheKey := generateKey(methodName, params...)
	ttl := c.GetMethodTTL(methodName)

	// Use existing GetWithCache logic
	return c.GetWithCache(cacheKey, fetchFunc, ttl)
}

// generateKey creates a unique cache key for the given method and parameters
func generateKey(method string, params ...interface{}) string {
	// Create a hash of the method and parameters
	data := map[string]interface{}{
		"method": method,
		"params": params,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		// Fallback to simple string concatenation if JSON marshaling fails
		return fmt.Sprintf("%s-%v", method, params)
	}

	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// GenerateKey is the public version of generateKey
func GenerateKey(method string, params ...interface{}) string {
	return generateKey(method, params...)
}

// Get retrieves a value from the block cache
func (c *RPCCache) GetBlock(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.blockCache.Get(key)
	if !exists {
		return nil, false
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		c.blockCache.Remove(key)
		return nil, false
	}

	return entry.Data, true
}

// GetWithCache is a generic function to handle cache operations for RPC functions
// It takes a cache key, a function to execute if cache miss, and optional TTL
func (c *RPCCache) GetWithCache(key string, fetchFunc func() (interface{}, error), ttl ...time.Duration) (interface{}, error) {
	// Check cache first
	if cachedData, found := c.GetBlock(key); found {
		return cachedData, nil
	}

	// Cache miss - execute the fetch function
	result, err := fetchFunc()
	if err != nil {
		return nil, err
	}

	// Cache the result if it's not nil
	if result != nil {
		if len(ttl) > 0 {
			c.SetBlockWithTTL(key, result, ttl[0])
		} else {
			c.SetBlock(key, result)
		}
	}

	return result, nil
}

// Set stores a value in the block cache with the default TTL
func (c *RPCCache) SetBlock(key string, value interface{}) {
	c.SetBlockWithTTL(key, value, c.defaultTTL)
}

// SetBlockWithTTL stores a value in the block cache with a custom TTL
func (c *RPCCache) SetBlockWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := &CacheEntry{
		Data:      value,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}

	c.blockCache.Add(key, entry)
}

// Stats returns cache statistics
func (c *RPCCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	methods := make(map[string]interface{})
	for method, ttl := range c.cachedMethods {
		methods[method] = map[string]interface{}{
			"enabled": true,
			"ttl":     ttl.String(),
		}
	}

	return map[string]interface{}{
		"block_cache_size": c.blockCache.Len(),
		"default_ttl":      c.defaultTTL.String(),
		"cached_methods":   methods,
	}
}

// Clear removes all entries from the cache
func (c *RPCCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.blockCache.Purge()
}

// Cleanup removes expired entries from the cache
func (c *RPCCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// LRU cache doesn't have a direct cleanup method, but we can iterate and remove expired entries
	// This is a simple implementation - in production you might want a more efficient approach
	keys := c.blockCache.Keys()
	for _, key := range keys {
		if entry, exists := c.blockCache.Get(key); exists {
			if time.Now().After(entry.ExpiresAt) {
				c.blockCache.Remove(key)
			}
		}
	}
}
