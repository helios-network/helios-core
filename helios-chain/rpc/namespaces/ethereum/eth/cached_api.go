package eth

import (
	"fmt"
	"reflect"
	"time"

	"helios-core/helios-chain/rpc/backend"
	"helios-core/helios-chain/rpc/cache"
	"helios-core/helios-chain/rpc/types"
	rpctypes "helios-core/helios-chain/rpc/types"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	hyperiontypes "helios-core/helios-chain/x/hyperion/types"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// CachedPublicAPI is a proxy wrapper around PublicAPI that automatically applies caching
// to method calls based on configuration
type CachedPublicAPI struct {
	*PublicAPI
	logger log.Logger
	cache  *cache.RPCCache
}

// NewCachedPublicAPI creates a new cached API wrapper
func NewCachedPublicAPI(logger log.Logger, backend backend.EVMBackend) *CachedPublicAPI {
	// Create the original API
	originalAPI := NewPublicAPI(logger, backend)

	// Create cache for the API
	rpcCache, err := cache.NewRPCCache(1000, 30*time.Second)
	if err != nil {
		logger.Error("Failed to create cache for API", "error", err)
		// Return original API without cache
		return &CachedPublicAPI{
			PublicAPI: originalAPI,
			logger:    logger.With("module", "cached-eth-api"),
			cache:     nil,
		}
	}

	return &CachedPublicAPI{
		PublicAPI: originalAPI,
		logger:    logger.With("module", "cached-eth-api"),
		cache:     rpcCache,
	}
}

// interceptMethodCall intercepts a method call and applies caching if configured
func (c *CachedPublicAPI) interceptMethodCall(methodName string, args []interface{}, method reflect.Value) ([]interface{}, error) {
	// If no cache available, call directly
	if c.cache == nil {
		return c.callMethod(method, args)
	}

	// Check if method should be cached
	if !c.cache.IsMethodCached(methodName) {
		// Method not configured for caching, call directly
		return c.callMethod(method, args)
	}

	// Generate cache key from method name and arguments
	cacheKey := cache.GenerateKey(methodName, args...)
	ttl := c.cache.GetMethodTTL(methodName)

	// Check cache first
	if cachedData, found := c.cache.GetBlock(cacheKey); found {
		c.logger.Debug("Cache hit", "method", methodName, "key", cacheKey)
		return []interface{}{cachedData}, nil
	}

	// Cache miss - execute the method
	c.logger.Debug("Cache miss", "method", methodName, "key", cacheKey)
	result, err := c.callMethod(method, args)
	if err != nil {
		return nil, err
	}

	// Cache the result if it's not nil and not an error
	if len(result) > 0 && result[0] != nil {
		c.cache.SetBlockWithTTL(cacheKey, result[0], ttl)
		c.logger.Debug("Cached result", "method", methodName, "key", cacheKey, "ttl", ttl)
	}

	return result, nil
}

// callMethod calls the actual method with the given arguments
func (c *CachedPublicAPI) callMethod(method reflect.Value, args []interface{}) ([]interface{}, error) {
	// Convert args to reflect.Value slice
	argValues := make([]reflect.Value, len(args))
	for i, arg := range args {
		argValues[i] = reflect.ValueOf(arg)
	}

	// Call the method
	results := method.Call(argValues)

	// Convert results to interface{} slice
	resultInterfaces := make([]interface{}, len(results))
	for i, result := range results {
		resultInterfaces[i] = result.Interface()
	}

	// Check for error in results
	if len(results) > 0 {
		if err, ok := results[len(results)-1].Interface().(error); ok && err != nil {
			return nil, err
		}
	}

	return resultInterfaces, nil
}

// GetCacheStats returns cache statistics
func (c *CachedPublicAPI) GetCacheStats() map[string]interface{} {
	if c.cache == nil {
		return map[string]interface{}{
			"cache_available": false,
		}
	}
	return c.cache.Stats()
}

// Method call wrappers with exact signatures matching the original PublicAPI

// GetAllHyperionTransferTxs returns all hyperion transfer transactions with caching
func (c *CachedPublicAPI) GetAllHyperionTransferTxs(size hexutil.Uint64) ([]*hyperiontypes.TransferTx, error) {
	methodName := "GetAllHyperionTransferTxs"
	args := []interface{}{size}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if txs, ok := results[0].([]*hyperiontypes.TransferTx); ok {
			return txs, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetAllHyperionTransferTxs")
}

// GetHyperionAccountTransferTxsByPageAndSize returns hyperion account transfer transactions with caching
func (c *CachedPublicAPI) GetHyperionAccountTransferTxsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*hyperiontypes.TransferTx, error) {
	methodName := "GetHyperionAccountTransferTxsByPageAndSize"
	args := []interface{}{address, page, size}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if txs, ok := results[0].([]*hyperiontypes.TransferTx); ok {
			return txs, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetHyperionAccountTransferTxsByPageAndSize")
}

// GetValidatorWithHisAssetsAndCommission returns validator with assets and commission with caching
func (c *CachedPublicAPI) GetValidatorWithHisAssetsAndCommission(address common.Address) (*rpctypes.ValidatorWithCommissionAndAssetsRPC, error) {
	methodName := "GetValidatorWithHisAssetsAndCommission"
	args := []interface{}{address}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if validator, ok := results[0].(*types.ValidatorWithCommissionAndAssetsRPC); ok {
			return validator, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetValidatorWithHisAssetsAndCommission")
}

// GetAllWhitelistedAssets returns all whitelisted assets with caching
func (c *CachedPublicAPI) GetAllWhitelistedAssets() ([]types.WhitelistedAssetRPC, error) {
	methodName := "GetAllWhitelistedAssets"
	args := []interface{}{}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if assets, ok := results[0].([]types.WhitelistedAssetRPC); ok {
			return assets, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetAllWhitelistedAssets")
}

// GetLastTransactionsInfo returns last transactions info with caching
func (c *CachedPublicAPI) GetLastTransactionsInfo(size hexutil.Uint64) ([]*rpctypes.ParsedRPCTransaction, error) {
	methodName := "GetLastTransactionsInfo"
	args := []interface{}{size}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if txs, ok := results[0].([]*rpctypes.ParsedRPCTransaction); ok {
			return txs, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetLastTransactionsInfo")
}

// GetAccountLastTransactionsInfo returns account last transactions info with caching
func (c *CachedPublicAPI) GetAccountLastTransactionsInfo(address common.Address) ([]*rpctypes.ParsedRPCTransaction, error) {
	methodName := "GetAccountLastTransactionsInfo"
	args := []interface{}{address}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if txs, ok := results[0].([]*rpctypes.ParsedRPCTransaction); ok {
			return txs, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetAccountLastTransactionsInfo")
}

// GetTokensByChainIdAndPageAndSize returns tokens by chain id and page and size with caching
func (c *CachedPublicAPI) GetTokensByChainIdAndPageAndSize(chainId uint64, page hexutil.Uint64, size hexutil.Uint64) ([]banktypes.FullMetadata, error) {
	methodName := "GetTokensByChainIdAndPageAndSize"
	args := []interface{}{chainId, page, size}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if tokens, ok := results[0].([]banktypes.FullMetadata); ok {
			return tokens, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetTokensByChainIdAndPageAndSize")
}

// GetActiveValidatorCount returns active validator count with caching
func (c *CachedPublicAPI) GetActiveValidatorCount() (int, error) {
	methodName := "GetActiveValidatorCount"
	args := []interface{}{}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return 0, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return 0, err
	}

	if len(results) > 0 {
		if count, ok := results[0].(int); ok {
			return count, nil
		}
	}
	return 0, fmt.Errorf("invalid return type for GetActiveValidatorCount")
}

// GetTokenDetails returns token details with caching
func (c *CachedPublicAPI) GetTokenDetails(tokenAddress common.Address) (*banktypes.FullMetadata, error) {
	methodName := "GetTokenDetails"
	args := []interface{}{tokenAddress}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if token, ok := results[0].(*banktypes.FullMetadata); ok {
			return token, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetTokenDetails")
}

// ChainId returns the chain id with caching
func (c *CachedPublicAPI) ChainId() (*hexutil.Big, error) {
	methodName := "ChainId"
	args := []interface{}{}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if chainId, ok := results[0].(*hexutil.Big); ok {
			return chainId, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for ChainId")
}

// BlockNumber returns the block number with caching
func (c *CachedPublicAPI) BlockNumber() (hexutil.Uint64, error) {
	methodName := "BlockNumber"
	args := []interface{}{}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return 0, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return 0, err
	}

	if len(results) > 0 {
		if blockNumber, ok := results[0].(hexutil.Uint64); ok {
			return blockNumber, nil
		}
	}
	return 0, fmt.Errorf("invalid return type for BlockNumber")
}

// Generic method interceptor using reflection
func (c *CachedPublicAPI) InterceptMethod(methodName string, args ...interface{}) (interface{}, error) {
	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		return results[0], nil
	}
	return nil, nil
}
