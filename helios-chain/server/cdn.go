package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	hyperiontypes "helios-core/helios-chain/x/hyperion/types"
	logostypes "helios-core/helios-chain/x/logos/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"golang.org/x/sync/errgroup"

	rpctypes "helios-core/helios-chain/rpc/types"
	svrconfig "helios-core/helios-chain/server/config"
	erc20types "helios-core/helios-chain/x/erc20/types"
)

// StartCDNServer starts an HTTP server to serve token and blockchain logos
func StartCDNServer(
	svrCtx *server.Context,
	clientCtx client.Context,
	g *errgroup.Group,
	config svrconfig.Config,
) (*http.Server, chan struct{}, error) {

	logger := svrCtx.Logger.With("module", "cdn")

	r := mux.NewRouter()

	r.HandleFunc("/token/{address}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		address := vars["address"]

		if !common.IsHexAddress(address) {
			http.Error(w, "Invalid Ethereum address", http.StatusBadRequest)
			return
		}

		tokenAddress := common.HexToAddress(address)

		logoData, err := getTokenLogo(r.Context(), clientCtx, tokenAddress)
		if err != nil {
			http.Error(w, "Failed to retrieve token logo: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24h
		logoData = strings.TrimPrefix(logoData, "data:image/png;base64,")
		imgBytes, err := base64.StdEncoding.DecodeString(logoData)
		if err != nil {
			http.Error(w, "Failed to decode logo image", http.StatusInternalServerError)
			return
		}

		w.Write(imgBytes)
	})

	r.HandleFunc("/blockchain/{chain_id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		chainID := vars["chain_id"]
		logoData := ""

		chainIDUint64, err := strconv.ParseUint(chainID, 10, 64)
		if err != nil {
			http.Error(w, "Invalid chain ID", http.StatusBadRequest)
			return
		}
		hyperionClient := hyperiontypes.NewQueryClient(clientCtx.GRPCClient)
		chainsRes, err := hyperionClient.QueryGetCounterpartyChainParamsByChainId(r.Context(), &hyperiontypes.QueryGetCounterpartyChainParamsByChainIdRequest{
			ChainId: chainIDUint64,
		})
		if err == nil {
			logoHash := chainsRes.CounterpartyChainParams.BridgeChainLogo

			logger.Info("logoHash", "logoHash", logoHash)
			if logoHash != "" {
				logosClient := logostypes.NewQueryClient(clientCtx.GRPCClient)
				logoResp, err := logosClient.Logo(r.Context(), &logostypes.QueryLogoRequest{
					Hash: logoHash,
				})
				if err == nil {
					logoData = logoResp.Logo.Data
				}
			}
		} else {
			http.Error(w, fmt.Sprintf("%d %s", chainIDUint64, err.Error()), http.StatusBadRequest)
			return
		}

		if logoData == "" {
			logoData, err = getBlockchainLogo(chainID)
			if err != nil {
				http.Error(w, "Failed to retrieve blockchain logo: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24h

		logoData = strings.TrimPrefix(logoData, "data:image/png;base64,")

		imgBytes, err := base64.StdEncoding.DecodeString(logoData)
		if err != nil {
			http.Error(w, "Failed to decode logo image", http.StatusInternalServerError)
			return
		}

		w.Write(imgBytes)
	})

	// Définir les handlers pour les routes
	r.HandleFunc("/hash/{hash}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		hash := vars["hash"]

		logosClient := logostypes.NewQueryClient(clientCtx.GRPCClient)
		logoResp, err := logosClient.Logo(r.Context(), &logostypes.QueryLogoRequest{
			Hash: hash,
		})
		if err != nil {
			http.Error(w, "Failed to retrieve logo: "+err.Error(), http.StatusInternalServerError)
			return
		}

		logoData := logoResp.Logo.Data

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24h

		logoData = strings.TrimPrefix(logoData, "data:image/png;base64,")

		imgBytes, err := base64.StdEncoding.DecodeString(logoData)
		if err != nil {
			http.Error(w, "Failed to decode logo image", http.StatusInternalServerError)
			return
		}

		w.Write(imgBytes)
	})

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		html := "<h1>Helios CDN API</h1>"
		html += "<p>Available endpoints:</p>"
		html += "<ul>"
		html += "<li><b>/token/{address}</b> - Get token logo by address<br>Example: /token/0x1234567890123456789012345678901234567890</li><br>"
		html += "<li><b>/blockchain/{chain_id}</b> - Get blockchain logo by chain ID<br>Example: /blockchain/1</li><br>"
		html += "<li><b>/hash/{hash}</b> - Get logo directly by hash<br>Example: /hash/abcdef1234567890</li><br>"
		html += "</ul>"

		w.Write([]byte(html))
	})

	httpSrv := &http.Server{
		Addr:    config.Cdn.Address,
		Handler: r,
	}

	httpSrvDone := make(chan struct{}, 1)

	ln, err := Listen(httpSrv.Addr, &config)
	if err != nil {
		return nil, nil, err
	}

	errCh := make(chan error)
	go func() {
		svrCtx.Logger.Info("Starting CDN server", "address", config.Cdn.Address)
		if err := httpSrv.Serve(ln); err != nil {
			if err == http.ErrServerClosed {
				close(httpSrvDone)
				return
			}

			svrCtx.Logger.Error("failed to start CDN server", "error", err.Error())
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		svrCtx.Logger.Error("failed to boot CDN server", "error", err.Error())
		return nil, nil, err
	case <-time.After(svrconfig.ServerStartTime): // assume JSON RPC server started successfully
	}

	svrCtx.Logger.Info("CDN server started successfully", "address", config.Cdn.Address)

	return httpSrv, httpSrvDone, nil
}

// getTokenLogo retrieves the logo of a token
func getTokenLogo(ctx context.Context, clientCtx client.Context, tokenAddress common.Address) (string, error) {

	logoData := ""

	erc20Client := erc20types.NewQueryClient(clientCtx.GRPCClient)
	erc20Req := &erc20types.QueryTokenPairRequest{
		Token: tokenAddress.String(),
	}
	erc20Res, err := erc20Client.TokenPair(ctx, erc20Req)
	if err == nil {
		bankClient := banktypes.NewQueryClient(clientCtx.GRPCClient)

		bankRes, err := bankClient.DenomMetadata(ctx, &banktypes.QueryDenomMetadataRequest{
			Denom: erc20Res.TokenPair.Denom,
		})
		if err == nil {
			logoHash := bankRes.Metadata.Logo
			if logoHash != "" {
				logosClient := logostypes.NewQueryClient(clientCtx.GRPCClient)
				logoResp, err := logosClient.Logo(ctx, &logostypes.QueryLogoRequest{
					Hash: logoHash,
				})
				if err == nil {
					logoData = logoResp.Logo.Data
				}
			} else {
				return rpctypes.GenerateTokenLogoBase64(bankRes.Metadata.Symbol)
			}
		} else {
			return rpctypes.GenerateTokenLogoBase64(erc20Res.TokenPair.Denom)
		}
	}

	if logoData == "" {
		return rpctypes.GenerateTokenLogoBase64("...")
	}

	return logoData, nil
}

// getBlockchainLogo retrieves the logo of a blockchain
func getBlockchainLogo(chainID string) (string, error) {
	// Ici vous pouvez appeler votre backend pour récupérer le logo de la blockchain
	// Pour l'instant, générer un logo de placeholder

	// Générer un logo par défaut basé sur l'ID de la chaîne
	return rpctypes.GenerateTokenLogoBase64(chainID)
}
