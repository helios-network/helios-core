package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMethodRateLimits(t *testing.T) {
	// Test empty string
	result := parseMethodRateLimits("")
	require.Equal(t, 0, len(result))

	// Test single method
	result = parseMethodRateLimits("eth_call:1")
	require.Equal(t, 1, len(result))
	require.Equal(t, 1, result["eth_call"])

	// Test multiple methods
	result = parseMethodRateLimits("eth_call:1,eth_estimateGas:1")
	require.Equal(t, 2, len(result))
	require.Equal(t, 1, result["eth_call"])
	require.Equal(t, 1, result["eth_estimateGas"])

	// Test with spaces
	result = parseMethodRateLimits(" eth_call : 1 , eth_estimateGas : 1 ")
	require.Equal(t, 2, len(result))
	require.Equal(t, 1, result["eth_call"])
	require.Equal(t, 1, result["eth_estimateGas"])

	// Test with different limits
	result = parseMethodRateLimits("eth_call:1,eth_getLogs:3,eth_getStorageAt:5")
	require.Equal(t, 3, len(result))
	require.Equal(t, 1, result["eth_call"])
	require.Equal(t, 3, result["eth_getLogs"])
	require.Equal(t, 5, result["eth_getStorageAt"])

	// Test invalid formats (should be ignored)
	result = parseMethodRateLimits("eth_call:1,invalid_format,eth_estimateGas:2")
	require.Equal(t, 2, len(result))
	require.Equal(t, 1, result["eth_call"])
	require.Equal(t, 2, result["eth_estimateGas"])

	// Test invalid limits (should be ignored)
	result = parseMethodRateLimits("eth_call:1,eth_estimateGas:abc,eth_getLogs:3")
	require.Equal(t, 2, len(result))
	require.Equal(t, 1, result["eth_call"])
	require.Equal(t, 3, result["eth_getLogs"])

	// Test zero and negative limits (should be ignored)
	result = parseMethodRateLimits("eth_call:0,eth_estimateGas:-1,eth_getLogs:3")
	require.Equal(t, 1, len(result))
	require.Equal(t, 3, result["eth_getLogs"])
}
