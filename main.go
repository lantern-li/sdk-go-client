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
	createContractTimeout = 5
	claimContractName     = "claim001"
	claimVersion          = "2.0.0"
	claimByteCodePath     = "./config/fact.wasm"

	// test_rwset 合约配置
	testRwsetContractName = "test_rwset_contract"
	testRwsetVersion      = "1.0.0"
	testRwsetByteCodePath = "./config/test_rwset.wasm"

	sdkConfigOrg1Client1Path = "./config/sdk_config.yml"
)

func main() {
	// 普通测试：创建合约并发送单笔交易
	userContractClaim()

	// 部署 test_rwset 合约
	deployTestRwsetContract()

	// 调用 test_rwset 合约
	invokeTestRwsetContract()
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

func deployTestRwsetContract() {
	fmt.Println("====================== 部署 test_rwset 合约 ======================")

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	// 检查合约是否已存在
	contract, err := client.GetContractInfo(testRwsetContractName)
	if err != nil {
		if strings.Contains(err.Error(), "contract not exist") {
			fmt.Printf("合约[%s]不存在，开始创建合约\n", testRwsetContractName)
		} else {
			panicErr(err)
		}
	} else {
		fmt.Printf("合约已存在: %s (version: %s)，跳过创建\n\n", contract.Name, contract.Version)
		return
	}

	fmt.Println("====================== 创建 test_rwset 合约 ======================")
	createPayload, err := client.CreateContractCreatePayload(
		testRwsetContractName,
		testRwsetVersion,
		testRwsetByteCodePath,
		common.RuntimeType_WASMER,
		[]*common.KeyValuePair{},
	)
	panicErr(err)

	resp, err := client.SendContractManageRequest(createPayload, nil, createContractTimeout, true)
	panicErr(err)

	fmt.Printf("✓ test_rwset 合约部署成功!\n")
	fmt.Printf("  - 合约名称: %s\n", testRwsetContractName)
	fmt.Printf("  - 合约版本: %s\n", testRwsetVersion)
	fmt.Printf("  - 区块高度: %d\n", resp.TxBlockHeight)
	fmt.Printf("  - 交易ID: %s\n", resp.TxId)
	fmt.Printf("  - 执行结果: %s\n", resp.ContractResult.Result)
	fmt.Printf("  - 消息: %s\n\n", resp.ContractResult.Message)
}

func invokeTestRwsetContract() {
	fmt.Println("====================== 调用 test_rwset 合约 ======================")

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	// 准备调用参数
	curTime := strconv.FormatInt(time.Now().Unix(), 10)
	kvs := []*common.KeyValuePair{
		{
			Key:   "read_key",
			Value: []byte(fmt.Sprintf("test_read_key_%s", curTime)),
		},
		{
			Key:   "read_field",
			Value: []byte(""), // 空，使用默认值 "data"
		},
		{
			Key:   "write_key",
			Value: []byte(fmt.Sprintf("test_write_key_%s", curTime)),
		},
		{
			Key:   "write_field",
			Value: []byte(""), // 空，使用默认值 "data"
		},
		{
			Key:   "write_value",
			Value: []byte(fmt.Sprintf("test_value_%s", curTime)),
		},
	}

	fmt.Println("调用参数:")
	fmt.Printf("  - read_key: test_read_key_%s\n", curTime)
	fmt.Printf("  - read_field: (空，使用默认 'data')\n")
	fmt.Printf("  - write_key: test_write_key_%s\n", curTime)
	fmt.Printf("  - write_field: (空，使用默认 'data')\n")
	fmt.Printf("  - write_value: test_value_%s\n\n", curTime)

	// 调用合约
	resp, err := client.InvokeContract(testRwsetContractName, "test_rwset", "", kvs, -1, true)
	panicErr(err)

	if resp.Code != common.TxStatusCode_SUCCESS {
		err = fmt.Errorf("调用合约失败, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
		panicErr(err)
	}

	fmt.Printf("✓ test_rwset 合约调用成功!\n")
	fmt.Printf("  - 区块高度: %d\n", resp.TxBlockHeight)
	fmt.Printf("  - 交易ID: %s\n", resp.TxId)
	fmt.Printf("  - 执行结果: %s\n", string(resp.ContractResult.Result))
	fmt.Printf("  - 消息: %s\n\n", resp.ContractResult.Message)
}

func panicErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
