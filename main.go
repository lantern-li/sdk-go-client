package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"chainmaker.org/chainmaker/common/v2/random/uuid"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
)

const (
	createContractTimeout    = 5
	claimContractName        = "claim001"
	claimVersion             = "2.0.0"
	claimByteCodePath        = "./config/rust-fact-2.0.0.wasm"
	sdkConfigOrg1Client1Path = "./config/sdk_config.yml"
)

func main() {
	userContractClaim()
}

func userContractClaim() {
	fmt.Println("====================== create client ======================")

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	fmt.Println("====================== 创建合约 ======================")
	createPayload, err := client.CreateContractCreatePayload(claimContractName, claimVersion, claimByteCodePath, common.RuntimeType_WASMER, []*common.KeyValuePair{})
	panicErr(err)
	resp, err := client.SendContractManageRequest(createPayload, nil, createContractTimeout, true)
	panicErr(err)
	fmt.Printf("blockHeight:%d, txId:%s, result:%s, msg:%s\n\n", resp.TxBlockHeight, resp.TxId, resp.ContractResult.Result, resp.ContractResult.Message)

	fmt.Println("====================== 调用合约 ======================")
	curTime := strconv.FormatInt(time.Now().Unix(), 10)
	fileHash := uuid.GetUUID()
	kvs := []*common.KeyValuePair{
		{
			Key:   "time",
			Value: []byte(curTime),
		},
		{
			Key:   "file_hash",
			Value: []byte(fileHash),
		},
		{
			Key:   "file_name",
			Value: []byte(fmt.Sprintf("file_%s", curTime)),
		},
	}
	resp, err = client.InvokeContract(claimContractName, "save", "", kvs, -1, true)
	panicErr(err)
	if resp.Code != common.TxStatusCode_SUCCESS {
		err = fmt.Errorf("invoke contract failed, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
		panicErr(err)
	}
	fmt.Printf("blockHeight:%d, txId:%s, result:%s, msg:%s, fileHash:%s\n\n",
		resp.TxBlockHeight, resp.TxId, resp.ContractResult.Result, resp.ContractResult.Message, fileHash)

	fmt.Println("====================== 执行合约查询接口 ======================")
	kvs = []*common.KeyValuePair{
		{
			Key:   "file_hash",
			Value: []byte(fileHash),
		},
	}
	resp, err = client.QueryContract(claimContractName, "find_by_file_hash", kvs, -1)
	panicErr(err)

	fmt.Printf("QUERY claim contract resp: %+v\n\n", resp)
}

func panicErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
