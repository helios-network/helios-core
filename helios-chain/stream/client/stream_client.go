package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc/keepalive"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"helios-core/helios-chain/stream/types"
)

var kacp = keepalive.ClientParameters{
	Time:                30 * time.Second, // send pings every 30 seconds if there is no activity
	Timeout:             5 * time.Second,  // wait 5 second for ping ack before considering the connection dead
	PermitWithoutStream: false,            // do not send pings without active streams
}

func main() {
	cc, err := grpc.NewClient(
		"localhost:9999",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kacp),
	)
	// nolint:staticcheck //ignored on purpose
	defer cc.Close()
	if err != nil {
		panic(err)
	}
	client := types.NewStreamClient(cc)

	ctx := context.Background()
	stream, err := client.Stream(ctx, &types.StreamRequest{
		BankBalancesFilter: &types.BankBalancesFilter{
			Accounts: []string{"*"},
		},
	})
	if err != nil {
		panic(err)
	}

	for {
		res, err := stream.Recv()
		if err != nil {
			panic(err)
		}
		bz, _ := json.Marshal(res)
		fmt.Println(string(bz))
	}
}
