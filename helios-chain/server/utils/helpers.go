package utils

import (
	"strconv"
	"strings"
)

// ParseMethodRateLimits parses the method-rate-limits string from config
// Format: "eth_call:1,eth_estimateGas:1"
func ParseMethodRateLimits(configString string) map[string]int {
	result := make(map[string]int)
	if configString == "" {
		return result
	}

	methods := strings.Split(configString, ",")
	for _, method := range methods {
		parts := strings.Split(strings.TrimSpace(method), ":")
		if len(parts) == 2 {
			methodName := strings.TrimSpace(parts[0])
			limitStr := strings.TrimSpace(parts[1])
			if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
				result[methodName] = limit
			}
		}
	}

	return result
}
