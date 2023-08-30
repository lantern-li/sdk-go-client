package main

import (
	"chainmaker.org/chainmaker/spv/v2/pb/api"
	spvPb "chainmaker.org/chainmaker/spv/v2/pb/protogo/cm_pbgo"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
)

func main() {
	spv()
}

// 转发示例
func spv() {
	chainmakerReq := &spvPb.TxRequest{
		Payload:   nil,
		Sender:    nil,
		Endorsers: nil,
	}

	// 1.构造Client
	conn, err := grpc.Dial("127.0.0.1:12308", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	client := api.NewRpcForwarderClient(conn)
	resp, err := client.ForwardRequest(context.Background(), chainmakerReq)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%v", resp)
}
