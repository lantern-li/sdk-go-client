package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"chainmaker.org/chainmaker/common/v2/random/uuid"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
)

const (
	admin1CertPath = "./simple-gm/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt"
	admin1KeyPath  = "./simple-gm/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key"
	admin2CertPath = "./simple-gm/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt"
	admin2KeyPath  = "./simple-gm/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key"
	admin3CertPath = "./simple-gm/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt"
	admin3KeyPath  = "./simple-gm/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key"
	admin5CertPath = "./simple-gm/crypto-config/wx-org5.chainmaker.org/user/admin1/admin1.sign.crt"
	admin5KeyPath  = "./simple-gm/crypto-config/wx-org5.chainmaker.org/user/admin1/admin1.sign.key"

	createContractTimeout    = 5
	claimContractName        = "go_gm_claim002"
	claimVersion             = "2.0.0"
	claimByteCodePath        = "./simple-gm/rust-fact-2.0.0.wasm"
	sdkConfigOrg1Client1Path = "./simple-gm/sdk_config.yml"

	claimQueryMethod  = "find_by_file_hash"
	claimInvokeMethod = "save"

	org5Id         = "wx-org5.chainmaker.org"
	org5CaCertPath = "./simple-gm/crypto-config/wx-org5.chainmaker.org/ca/ca.crt"
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

	contract, err := client.GetContractInfo(claimContractName)
	if err != nil {
		if strings.Contains(err.Error(), "contract not exist") || contract == nil || contract.Name == "" {
			fmt.Printf("合约[%s]不存在\n", claimContractName)
			fmt.Println("====================== 创建合约 ======================")
			createPayload, err := client.CreateContractCreatePayload(claimContractName, claimVersion, claimByteCodePath, common.RuntimeType_WASMER, []*common.KeyValuePair{})
			panicErr(err)
			resp, err := client.SendContractManageRequest(createPayload, nil, createContractTimeout, true)
			panicErr(err)
			fmt.Printf("blockHeight:%d, txId:%s, result:%s, msg:%s\n\n", resp.TxBlockHeight, resp.TxId, resp.ContractResult.Result, resp.ContractResult.Message)
		} else {
			panicErr(err)
		}
	} else {
		fmt.Printf("合约已存在 %+v \n\n", contract)
	}

	fmt.Println("====================== 调用合约 ======================")
	time.Sleep(time.Second * 2)
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
	resp, err := client.InvokeContract(claimContractName, claimInvokeMethod, "", kvs, -1, true)
	panicErr(err)
	if resp.Code != common.TxStatusCode_SUCCESS {
		err = fmt.Errorf("invoke contract failed, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
		panicErr(err)
	}
	txId := resp.TxId
	blockHeight := resp.TxBlockHeight
	fmt.Printf("blockHeight:%d, txId:%s, msg:%s, fileHash:%s\n\n",
		resp.TxBlockHeight, resp.TxId, resp.ContractResult.Message, fileHash)

	fmt.Println("====================== 执行合约查询接口 ======================")
	time.Sleep(time.Second * 2)
	kvs = []*common.KeyValuePair{
		{
			Key:   "file_hash",
			Value: []byte(fileHash),
		},
	}
	resp, err = client.QueryContract(claimContractName, claimQueryMethod, kvs, -1)
	panicErr(err)
	fmt.Printf("QUERY claim contract resp: %+v\n\n", resp)

	fmt.Println("====================== 执行交易查询接口 ======================")
	time.Sleep(time.Second * 2)
	tx, err := client.GetTxByTxId(txId)
	panicErr(err)
	fmt.Printf("%+v \n\n", tx)

	fmt.Println("====================== 执行区块查询接口 ======================")
	time.Sleep(time.Second * 2)
	block, err := client.GetBlockByHeight(blockHeight, false)
	panicErr(err)
	fmt.Printf("%+v \n\n", block)
}

func panicErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
