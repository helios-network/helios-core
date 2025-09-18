package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	svrconfig "helios-core/helios-chain/server/config"

	"github.com/cosmos/cosmos-sdk/client"
)

// StartupNotifier manages startup notifications to the Helios network registry
type StartupNotifier struct {
	endpoint   string
	httpClient *http.Client
	clientCtx  client.Context
	logger     interface{ Info(string, ...interface{}) }
	rpcAddress string
}

// NotificationPayload represents the JSON payload sent to the network registry
type NotificationPayload struct {
	Name string `json:"name"`
	Port uint64 `json:"port,omitempty"`
}

// NewStartupNotifier creates a new startup notifier instance
func NewStartupNotifier(endpoint string, clientCtx client.Context, logger interface{ Info(string, ...interface{}) }, config *svrconfig.Config) *StartupNotifier {
	rpcAddress := ""
	if config != nil && config.JSONRPC.Enable {
		rpcAddress = config.JSONRPC.Address
	}

	return &StartupNotifier{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		clientCtx:  clientCtx,
		logger:     logger,
		rpcAddress: rpcAddress,
	}
}

// NotifyStartup checks if the blockchain is ready and notifies the network registry
func (sn *StartupNotifier) NotifyStartup(moniker string) error {
	// First, check if the blockchain is ready by calling eth_blockNumber
	if err := sn.checkBlockchainHealth(); err != nil {
		return fmt.Errorf("blockchain health check failed: %w", err)
	}

	// If health check passes, send notification to network registry
	if err := sn.sendNotification(moniker); err != nil {
		return fmt.Errorf("failed to send startup notification: %w", err)
	}

	sn.logger.Info("Successfully notified network registry of blockchain startup", "moniker", moniker, "endpoint", sn.endpoint, "rpc_address", sn.rpcAddress)
	return nil
}

// checkBlockchainHealth verifies the blockchain is running by calling eth_blockNumber RPC
func (sn *StartupNotifier) checkBlockchainHealth() error {
	// Create a simple HTTP client to test the RPC endpoint
	// We'll try to call the eth_blockNumber method via JSON-RPC

	jsonRPCPayload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
		"id":      1,
	}

	payloadBytes, err := json.Marshal(jsonRPCPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON-RPC payload: %w", err)
	}

	// Use the configured RPC address or fallback to default
	rpcURL := sn.getRPCURL()

	req, err := http.NewRequest("POST", rpcURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := sn.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("RPC call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("RPC call returned status %d", resp.StatusCode)
	}

	// Parse response to ensure it's valid
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode RPC response: %w", err)
	}

	// Check if there's a result field (which should contain the block number)
	if _, exists := result["result"]; !exists {
		return fmt.Errorf("RPC response missing result field")
	}

	sn.logger.Info("Blockchain health check passed", "result", result)

	return nil
}

// getRPCURL returns the RPC URL to use for health checks
func (sn *StartupNotifier) getRPCURL() string {
	return "http://localhost:" + strconv.FormatUint(sn.getRPCPort(), 10)
}

func (sn *StartupNotifier) getRPCPort() uint64 {
	if sn.rpcAddress == "" {
		return 8545
	}

	parts := strings.Split(sn.rpcAddress, ":")
	if len(parts) < 2 {
		return 8545
	}

	port, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 8545
	}

	return port
}

// sendNotification sends the startup notification to the network registry
func (sn *StartupNotifier) sendNotification(moniker string) error {
	payload := NotificationPayload{
		Name: moniker,
		Port: sn.getRPCPort(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal notification payload: %w", err)
	}

	req, err := http.NewRequest("POST", sn.endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := sn.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notification endpoint returned status %d", resp.StatusCode)
	}

	return nil
}

// NotifyStartupAsync performs the startup notification asynchronously with retry logic
func (sn *StartupNotifier) NotifyStartupAsync(moniker string) {
	go func() {
		maxRetries := 5
		retryDelay := 10 * time.Second

		for attempt := 1; attempt <= maxRetries; attempt++ {
			if err := sn.NotifyStartup(moniker); err != nil {
				sn.logger.Info("Startup notification attempt failed",
					"attempt", attempt,
					"maxRetries", maxRetries,
					"error", err.Error(),
					"retryIn", retryDelay)

				if attempt < maxRetries {
					time.Sleep(retryDelay)
					retryDelay = retryDelay * 2 // Exponential backoff
					continue
				}

				sn.logger.Info("Failed to notify network registry after all attempts", "error", err.Error())
				return
			}

			// Success
			return
		}
	}()
}
