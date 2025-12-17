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
func (c *CachedPublicAPI) GetAllHyperionTransferTxs(size hexutil.Uint64) ([]*hyperiontypes.QueryTransferTx, error) {
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
		if txs, ok := results[0].([]*hyperiontypes.QueryTransferTx); ok {
			return txs, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetAllHyperionTransferTxs")
}

// GetHyperionAccountTransferTxsByPageAndSize returns hyperion account transfer transactions with caching
func (c *CachedPublicAPI) GetHyperionAccountTransferTxsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*hyperiontypes.QueryTransferTx, error) {
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
		if txs, ok := results[0].([]*hyperiontypes.QueryTransferTx); ok {
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

// GetValidatorsByPageAndSizeWithHisAssetsAndCommissionAndDelegation returns validators by page and size with assets, commission and delegation with caching
func (c *CachedPublicAPI) GetValidatorsByPageAndSizeWithHisAssetsAndCommissionAndDelegation(page hexutil.Uint64, size hexutil.Uint64) ([]types.ValidatorWithAssetsAndCommissionAndDelegationRPC, error) {
	methodName := "GetValidatorsByPageAndSizeWithHisAssetsAndCommissionAndDelegation"
	args := []interface{}{page, size}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if validators, ok := results[0].([]types.ValidatorWithAssetsAndCommissionAndDelegationRPC); ok {
			return validators, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetValidatorsByPageAndSizeWithHisAssetsAndCommissionAndDelegation")
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
func (c *CachedPublicAPI) GetTokensByChainIdAndPageAndSize(chainId uint64, page hexutil.Uint64, size hexutil.Uint64) ([]*banktypes.FullMetadata, error) {
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
		if tokens, ok := results[0].([]*banktypes.FullMetadata); ok {
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

// GetCoinbase returns the coinbase with caching
func (c *CachedPublicAPI) GetCoinbase() (common.Address, error) {
	methodName := "GetCoinbase"
	args := []interface{}{}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return common.Address{}, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return common.Address{}, err
	}

	if len(results) > 0 {
		if coinbase, ok := results[0].(common.Address); ok {
			return coinbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("invalid return type for GetCoinbase")
}

// GetHyperionHistoricalFees returns hyperion historical fees with caching
func (c *CachedPublicAPI) GetHyperionHistoricalFees(hyperionId uint64) (*hyperiontypes.QueryHistoricalFeesResponse, error) {
	methodName := "GetHyperionHistoricalFees"
	args := []interface{}{hyperionId}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if fees, ok := results[0].(*hyperiontypes.QueryHistoricalFeesResponse); ok {
			return fees, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetHyperionHistoricalFees")
}

// GetValidatorHyperionData returns validator hyperion data with caching
func (c *CachedPublicAPI) GetValidatorHyperionData(address common.Address) (*hyperiontypes.OrchestratorData, error) {
	methodName := "GetValidatorHyperionData"
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
		if data, ok := results[0].(*hyperiontypes.OrchestratorData); ok {
			return data, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetValidatorHyperionData")
}

// GetProposalsByPageAndSize returns proposals by page and size with caching
func (c *CachedPublicAPI) GetProposalsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.ProposalRPC, error) {
	methodName := "GetProposalsByPageAndSize"
	args := []interface{}{page, size}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if proposals, ok := results[0].([]*rpctypes.ProposalRPC); ok {
			return proposals, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetProposalsByPageAndSize")
}

// GetProposalsByPageAndSizeWithFilter returns proposals by page and size with filter with caching
func (c *CachedPublicAPI) GetProposalsByPageAndSizeWithFilter(page hexutil.Uint64, size hexutil.Uint64, filter string) ([]*rpctypes.ProposalRPC, error) {
	methodName := "GetProposalsByPageAndSizeWithFilter"
	args := []interface{}{page, size, filter}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if proposals, ok := results[0].([]*rpctypes.ProposalRPC); ok {
			return proposals, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetProposalsByPageAndSizeWithFilter")
}

// GetValidatorCount returns validator count with caching
func (c *CachedPublicAPI) GetValidatorCount() (int, error) {
	methodName := "GetValidatorCount"
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
	return 0, fmt.Errorf("invalid return type for GetValidatorCount")
}

// GetValidatorAPYDetails returns validator APY details with caching
func (c *CachedPublicAPI) GetValidatorAPYDetails(address common.Address) (*rpctypes.ValidatorAPYDetailsRPC, error) {
	methodName := "GetValidatorAPYDetails"
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
		if details, ok := results[0].(*rpctypes.ValidatorAPYDetailsRPC); ok {
			return details, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetValidatorAPYDetails")
}

// GetValidatorsAPYByPageAndSize returns validators APY by page and size with caching
func (c *CachedPublicAPI) GetValidatorsAPYByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]rpctypes.ValidatorAPYDetailsRPC, error) {
	methodName := "GetValidatorsAPYByPageAndSize"
	args := []interface{}{page, size}

	method := reflect.ValueOf(c.PublicAPI).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	results, err := c.interceptMethodCall(methodName, args, method)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if validators, ok := results[0].([]rpctypes.ValidatorAPYDetailsRPC); ok {
			return validators, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetValidatorsAPYByPageAndSize")
}

// GetCoinInfo returns coin information with caching
func (c *CachedPublicAPI) GetCoinInfo() (*rpctypes.CoinInfoRPC, error) {
	methodName := "GetCoinInfo"
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
		if coinInfo, ok := results[0].(*rpctypes.CoinInfoRPC); ok {
			return coinInfo, nil
		}
	}
	return nil, fmt.Errorf("invalid return type for GetCoinInfo")
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

func (c *CachedPublicAPI) CleanupCache() {
	c.cache.Cleanup()
}

func (c *CachedPublicAPI) StartCleanupCacheRoutine() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			c.cache.Cleanup()
		}
	}()
}
