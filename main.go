package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

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

	// smallbank 合约配置
	smallbankContractName = "smallbank_contract"
	smallbankVersion      = "1.0.0"
	smallbankByteCodePath = "./config/smallbank.wasm"

	sdkConfigOrg1Client1Path = "./config/sdk_config.yml"
)

func main() {
	// fact 合约相关
	//userContractClaim()

	// test_rwset 合约相关
	//deployTestRwsetContract()
	//invokeTestRwsetContract()

	// smallbank合约相关
	//deploySmallbankContract()
	//invokeSmallbankCreateAccount()
	//invokeSmallbankDepositChecking()
	//invokeSmallbankTransactSaving()
	//invokeSmallbankAmalgamate()
	//invokeSmallbankWriteCheck()
	invokeSmallbankSendPayment()
}

func userContractClaim() {
	fmt.Println("====================== create client ======================")

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	//fmt.Println("====================== 创建合约 ======================")
	//createPayload, err := client.CreateContractCreatePayload(claimContractName, claimVersion, claimByteCodePath, common.RuntimeType_WASMER, []*common.KeyValuePair{})
	//panicErr(err)
	//resp, err := client.SendContractManageRequest(createPayload, nil, createContractTimeout, true)
	//panicErr(err)
	//fmt.Printf("blockHeight:%d, txId:%s, result:%s, msg:%s\n\n", resp.TxBlockHeight, resp.TxId, resp.ContractResult.Result, resp.ContractResult.Message)

	//fmt.Println("====================== 调用合约 ======================")
	//curTime := strconv.FormatInt(time.Now().Unix(), 10)
	//fileHash := uuid.GetUUID()
	//kvs := []*common.KeyValuePair{
	//	{
	//		Key:   "time",
	//		Value: []byte(curTime),
	//	},
	//	{
	//		Key:   "file_hash",
	//		Value: []byte(fileHash),
	//	},
	//	{
	//		Key:   "file_name",
	//		Value: []byte(fmt.Sprintf("file_%s", curTime)),
	//	},
	//}
	//resp, err := client.InvokeContract(claimContractName, "save", "", kvs, -1, true)
	//panicErr(err)
	//if resp.Code != common.TxStatusCode_SUCCESS {
	//	err = fmt.Errorf("invoke contract failed, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
	//	panicErr(err)
	//}
	//fmt.Printf("blockHeight:%d, txId:%s, result:%s, msg:%s, fileHash:%s\n\n",
	//	resp.TxBlockHeight, resp.TxId, resp.ContractResult.Result, resp.ContractResult.Message, fileHash)

	fmt.Println("====================== 执行合约查询接口 ======================")
	kvs := []*common.KeyValuePair{
		{
			Key:   "file_hash",
			Value: []byte("3d0372e176c240efa0693873de1844bd"),
		},
	}
	resp, err := client.QueryContract(claimContractName, "find_by_file_hash", kvs, -1)
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

func deploySmallbankContract() {
	fmt.Println("====================== 部署 smallbank 合约 ======================")

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	// 检查合约是否已存在
	contract, err := client.GetContractInfo(smallbankContractName)
	if err != nil {
		if strings.Contains(err.Error(), "contract not exist") {
			fmt.Printf("合约[%s]不存在，开始创建合约\n", smallbankContractName)
		} else {
			panicErr(err)
		}
	} else {
		fmt.Printf("合约已存在: %s (version: %s)，跳过创建\n\n", contract.Name, contract.Version)
		return
	}

	fmt.Println("====================== 创建 smallbank 合约 ======================")
	createPayload, err := client.CreateContractCreatePayload(
		smallbankContractName,
		smallbankVersion,
		smallbankByteCodePath,
		common.RuntimeType_WASMER,
		[]*common.KeyValuePair{},
	)
	panicErr(err)

	resp, err := client.SendContractManageRequest(createPayload, nil, createContractTimeout, true)
	panicErr(err)

	fmt.Printf("✓ smallbank 合约部署成功!\n")
	fmt.Printf("  - 合约名称: %s\n", smallbankContractName)
	fmt.Printf("  - 合约版本: %s\n", smallbankVersion)
	fmt.Printf("  - 区块高度: %d\n", resp.TxBlockHeight)
	fmt.Printf("  - 交易ID: %s\n", resp.TxId)
	fmt.Printf("  - 执行结果: %s\n", resp.ContractResult.Result)
	fmt.Printf("  - 消息: %s\n\n", resp.ContractResult.Message)
}

func invokeSmallbankCreateAccount() {
	fmt.Println("====================== 调用 smallbank 合约 - create_account ======================")

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	// 准备调用参数 - 创建账户
	accountName := fmt.Sprintf("Bob_1234567890") //Bob_1234567890
	customerId := fmt.Sprintf("B1234567890")

	kvs := []*common.KeyValuePair{
		{
			Key:   "name",
			Value: []byte(accountName),
		},
		{
			Key:   "customer_id",
			Value: []byte(customerId),
		},
	}

	fmt.Println("调用参数:")
	fmt.Printf("  - name: %s\n", accountName)
	fmt.Printf("  - customer_id: %s\n\n", customerId)

	// 调用合约的 create_account 方法
	resp, err := client.InvokeContract(smallbankContractName, "create_account", "", kvs, -1, true)
	panicErr(err)

	if resp.Code != common.TxStatusCode_SUCCESS {
		err = fmt.Errorf("调用合约失败, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
		panicErr(err)
	}

	fmt.Printf("✓ create_account 方法调用成功!\n")
	fmt.Printf("  - 区块高度: %d\n", resp.TxBlockHeight)
	fmt.Printf("  - 交易ID: %s\n", resp.TxId)
	fmt.Printf("  - 执行结果: %s\n", string(resp.ContractResult.Result))
	fmt.Printf("  - 消息: %s\n", resp.ContractResult.Message)
	fmt.Printf("  - 账户名: %s\n", accountName)
	fmt.Printf("  - 客户ID: %s\n", customerId)
	fmt.Printf("  - 初始余额: saving=100000, checking=100000\n\n")
}

func invokeSmallbankDepositChecking() {
	fmt.Println("====================== 调用 smallbank 合约 - deposit_checking ======================")

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	// 准备调用参数 - 向 checking 账户存款
	accountName := "Alice_1234567890" // 使用已创建的账户名
	amount := "5000"

	kvs := []*common.KeyValuePair{
		{
			Key:   "name",
			Value: []byte(accountName),
		},
		{
			Key:   "amount",
			Value: []byte(amount),
		},
	}

	fmt.Println("调用参数:")
	fmt.Printf("  - name: %s\n", accountName)
	fmt.Printf("  - amount: %s\n\n", amount)

	// 调用合约的 deposit_checking 方法
	resp, err := client.InvokeContract(smallbankContractName, "deposit_checking", "", kvs, -1, true)
	panicErr(err)

	if resp.Code != common.TxStatusCode_SUCCESS {
		err = fmt.Errorf("调用合约失败, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
		panicErr(err)
	}

	fmt.Printf("✓ deposit_checking 方法调用成功!\n")
	fmt.Printf("  - 区块高度: %d\n", resp.TxBlockHeight)
	fmt.Printf("  - 交易ID: %s\n", resp.TxId)
	fmt.Printf("  - 新的 checking 余额: %s\n", string(resp.ContractResult.Result)) // 这里放的是合约执行的结果
	fmt.Printf("  - 消息: %s\n\n", resp.ContractResult.Message)
}

func invokeSmallbankTransactSaving() {
	fmt.Println("====================== 调用 smallbank 合约 - transact_saving ======================")

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	// 准备调用参数 - saving 账户交易 (正数存款，负数取款)
	accountName := "Alice_1234567890" // 使用已创建的账户名
	amount := "3000"                  // 正数表示存款，可以改为 "-2000" 表示取款

	kvs := []*common.KeyValuePair{
		{
			Key:   "name",
			Value: []byte(accountName),
		},
		{
			Key:   "amount",
			Value: []byte(amount),
		},
	}

	fmt.Println("调用参数:")
	fmt.Printf("  - name: %s\n", accountName)
	fmt.Printf("  - amount: %s (正数=存款, 负数=取款)\n\n", amount)

	// 调用合约的 transact_saving 方法
	resp, err := client.InvokeContract(smallbankContractName, "transact_saving", "", kvs, -1, true)
	panicErr(err)

	if resp.Code != common.TxStatusCode_SUCCESS {
		err = fmt.Errorf("调用合约失败, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
		panicErr(err)
	}

	fmt.Printf("✓ transact_saving 方法调用成功!\n")
	fmt.Printf("  - 区块高度: %d\n", resp.TxBlockHeight)
	fmt.Printf("  - 交易ID: %s\n", resp.TxId)
	fmt.Printf("  - 新的 saving 余额: %s\n", string(resp.ContractResult.Result))
	fmt.Printf("  - 消息: %s\n\n", resp.ContractResult.Message)
}

func invokeSmallbankAmalgamate() {
	fmt.Println("====================== 调用 smallbank 合约 - amalgamate ======================")

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	// 准备调用参数 - 将 name1 的所有资金转移到 name2
	accountName1 := "Alice_1234567890" // 源账户
	accountName2 := "Bob_1234567890"   // 目标账户

	kvs := []*common.KeyValuePair{
		{
			Key:   "name1",
			Value: []byte(accountName1),
		},
		{
			Key:   "name2",
			Value: []byte(accountName2),
		},
	}

	fmt.Println("调用参数:")
	fmt.Printf("  - name1 (源账户): %s\n", accountName1)
	fmt.Printf("  - name2 (目标账户): %s\n\n", accountName2)

	// 调用合约的 amalgamate 方法
	resp, err := client.InvokeContract(smallbankContractName, "amalgamate", "", kvs, -1, true)
	panicErr(err)

	if resp.Code != common.TxStatusCode_SUCCESS {
		err = fmt.Errorf("调用合约失败, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
		panicErr(err)
	}

	fmt.Printf("✓ amalgamate 方法调用成功!\n")
	fmt.Printf("  - 区块高度: %d\n", resp.TxBlockHeight)
	fmt.Printf("  - 交易ID: %s\n", resp.TxId)
	fmt.Printf("  - 执行结果: %s\n", string(resp.ContractResult.Result))
	fmt.Printf("  - 消息: %s\n\n", resp.ContractResult.Message)
}

func invokeSmallbankWriteCheck() {
	fmt.Println("====================== 调用 smallbank 合约 - write_check ======================")

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	// 准备调用参数 - 开支票
	accountName := "Alice_1234567890" // 使用已创建的账户名
	amount := "8000"                  // 支票金额

	kvs := []*common.KeyValuePair{
		{
			Key:   "name",
			Value: []byte(accountName),
		},
		{
			Key:   "amount",
			Value: []byte(amount),
		},
	}

	fmt.Println("调用参数:")
	fmt.Printf("  - name: %s\n", accountName)
	fmt.Printf("  - amount: %s\n", amount)
	fmt.Printf("  - 说明: 如果总余额 < 金额，将扣除(金额+1)作为罚金\n\n")

	// 调用合约的 write_check 方法
	resp, err := client.InvokeContract(smallbankContractName, "write_check", "", kvs, -1, true)
	panicErr(err)

	if resp.Code != common.TxStatusCode_SUCCESS {
		err = fmt.Errorf("调用合约失败, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
		panicErr(err)
	}

	fmt.Printf("✓ write_check 方法调用成功!\n")
	fmt.Printf("  - 区块高度: %d\n", resp.TxBlockHeight)
	fmt.Printf("  - 交易ID: %s\n", resp.TxId)
	fmt.Printf("  - 执行结果: %s\n", string(resp.ContractResult.Result))
	fmt.Printf("  - 消息: %s\n\n", resp.ContractResult.Message)
}

func invokeSmallbankSendPayment() {
	fmt.Println("====================== 调用 smallbank 合约 - send_payment ======================")

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	// 准备调用参数 - 转账
	sendName := "Bob_1234567890"   // 发送方账户名Bob_1234567890
	destName := "Alice_1234567890" // 接收方账户名
	amount := "2000"               // 转账金额

	kvs := []*common.KeyValuePair{
		{
			Key:   "send_name",
			Value: []byte(sendName),
		},
		{
			Key:   "dest_name",
			Value: []byte(destName),
		},
		{
			Key:   "amount",
			Value: []byte(amount),
		},
	}

	fmt.Println("调用参数:")
	fmt.Printf("  - send_name (发送方): %s\n", sendName)
	fmt.Printf("  - dest_name (接收方): %s\n", destName)
	fmt.Printf("  - amount: %s\n\n", amount)

	// 调用合约的 send_payment 方法
	resp, err := client.InvokeContract(smallbankContractName, "send_payment", "", kvs, -1, true)
	panicErr(err)

	if resp.Code != common.TxStatusCode_SUCCESS {
		err = fmt.Errorf("调用合约失败, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
		panicErr(err)
	}

	fmt.Printf("✓ send_payment 方法调用成功!\n")
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
