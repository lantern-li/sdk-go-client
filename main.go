package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/random/uuid"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
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
	//userContractClaim()

	// 部署 test_rwset 合约
	//deployTestRwsetContract()

	// 调用 test_rwset 合约
	//invokeTestRwsetContract()

	// 压测模式：取消下面的注释来运行压测
	StressTest()
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

// ==================== 压测相关代码 ====================

var (
	totalTxCount int64 = 0 // 总交易计数器
	orgIdCounter int64 = 0 // 组织ID计数器
)

// StressTest 压测函数：并发发送大量交易到链上
func StressTest() {
	fmt.Println("====================== 压测模式 ======================")
	fmt.Println("====================== 创建客户端 ======================")

	// 创建 SDK 客户端
	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	// 检查合约是否存在
	contract, err := client.GetContractInfo(claimContractName)
	if err != nil {
		if strings.Contains(err.Error(), "contract not exist") {
			fmt.Printf("合约[%s]不存在，开始创建合约\n", claimContractName)
			fmt.Println("====================== 创建合约 ======================")
			createPayload, err := client.CreateContractCreatePayload(
				claimContractName,
				claimVersion,
				claimByteCodePath,
				common.RuntimeType_WASMER,
				[]*common.KeyValuePair{},
			)
			panicErr(err)
			resp, err := client.SendContractManageRequest(createPayload, nil, createContractTimeout, true)
			panicErr(err)
			fmt.Printf("合约创建成功 - blockHeight:%d, txId:%s\n\n", resp.TxBlockHeight, resp.TxId)
		} else {
			panicErr(err)
		}
	} else {
		fmt.Printf("合约已存在: %s (version: %s)\n\n", contract.Name, contract.Version)
	}

	// 启动 TPS 统计 goroutine
	go monitorTPS()

	// 并发配置（单节点场景）
	goroutineCount := 100 // 并发 goroutine 数量
	txPerGoroutine := 10  // 每个 goroutine 发送的交易数（总共 5*20=100 笔）

	fmt.Printf("启动压测: %d 个并发 goroutine，每个发送 %d 笔交易（总共 %d 笔）\n",
		goroutineCount, txPerGoroutine, goroutineCount*txPerGoroutine)
	fmt.Println("开始发送交易...")
	fmt.Println("------------------------------------------------------------")

	// 启动多个 goroutine 并发发送交易
	var wg sync.WaitGroup
	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func(goroutineId int) {
			defer wg.Done()
			sendTransactions(client, txPerGoroutine, goroutineId)
		}(i)
	}

	// 等待所有 goroutine 完成
	wg.Wait()
	fmt.Println("\n压测完成！")
}

// sendTransactions 发送交易的核心函数
func sendTransactions(client *sdk.ChainClient, count int, goroutineId int) {
	for i := 0; i < count; i++ {
		// 生成交易参数
		curTime := strconv.FormatInt(time.Now().Unix(), 10)
		fileHash := uuid.GetUUID() // 返回一个唯一的标识符。

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
				Value: []byte(fmt.Sprintf("stress_test_file_%s_%d_%d", curTime, goroutineId, i)),
			},
		}

		// 使用不同的签名者发送交易（模拟多用户场景）
		signer := getRotatingSigner()

		// 同步发送交易（withSyncResult: true）以获取交易结果
		resp, err := client.InvokeContractBySigner(
			claimContractName,
			"save", // 合约方法
			"",     // txId（空表示自动生成）
			kvs,    // 参数
			-1,     // timeout（-1 表示使用默认值）
			false,  // withSyncResult（true = 同步，等待上链结果）
			nil,    // limit
			signer, // 使用签名者
		)

		if err != nil {
			// 发送失败时打印错误，但不中断压测
			fmt.Printf("[Goroutine %d][Tx %d] ❌ 交易发送失败: %v\n", goroutineId, i+1, err)
			continue
		}

		// 检查交易执行结果
		if resp.Code != common.TxStatusCode_SUCCESS {
			fmt.Printf("[Goroutine %d][Tx %d] ❌ 交易执行失败 - Code:%d, Msg:%s, TxId:%s\n",
				goroutineId, i+1, resp.Code, resp.Message, resp.TxId)
			continue
		}

		// 打印交易成功信息
		fmt.Printf("[Goroutine %d][Tx %d] ✓ 交易成功 - BlockHeight:%d, TxId:%s, FileHash:%s\n",
			goroutineId, i+1, resp.TxBlockHeight, resp.TxId, fileHash)

		// 增加成功交易计数
		atomic.AddInt64(&totalTxCount, 1)
	}
}

// getRotatingSigner 获取签名者（单节点场景，使用 wx-org.chainmaker.org）
func getRotatingSigner() *sdk.CertModeSigner {
	// 单节点场景：只使用 wx-org.chainmaker.org 的证书
	orgId := "wx-org.chainmaker.org"
	certPath := "./config/crypto-config/wx-org.chainmaker.org/user/client1/client1.sign.crt"
	keyPath := "./config/crypto-config/wx-org.chainmaker.org/user/client1/client1.sign.key"

	// 读取证书文件
	certPem, err := ioutil.ReadFile(certPath)
	panicErr(err)

	// 解析证书
	cert, err := sdkutils.ParseCert(certPem)
	panicErr(err)

	// 读取私钥文件
	privKeyPem, err := ioutil.ReadFile(keyPath)
	panicErr(err)

	// 解析私钥
	privateKey, err := asym.PrivateKeyFromPEM(privKeyPem, nil)
	panicErr(err)

	// 返回签名者对象
	return &sdk.CertModeSigner{
		PrivateKey: privateKey,
		Cert:       cert,
		OrgId:      orgId,
	}
}

// monitorTPS TPS 监控函数，每 2 秒打印一次统计信息
func monitorTPS() {
	ticker := time.NewTicker(2 * time.Second)
	startTime := time.Now()
	lastCount := int64(0)

	for {
		select {
		case <-ticker.C:
			currentCount := atomic.LoadInt64(&totalTxCount)
			elapsed := time.Since(startTime).Seconds()

			// 计算平均 TPS 和瞬时 TPS
			avgTPS := float64(currentCount) / elapsed
			instantTPS := float64(currentCount-lastCount) / 2.0

			fmt.Printf("[TPS Monitor] 总交易数: %d | 平均TPS: %.2f | 瞬时TPS: %.2f | 运行时间: %.0f秒\n",
				currentCount, avgTPS, instantTPS, elapsed)

			lastCount = currentCount

			// 可选：运行一段时间后重置计数器
			if elapsed > 60 {
				// 每 60 秒重置一次，避免数值过大
				startTime = time.Now()
				atomic.StoreInt64(&totalTxCount, 0)
				lastCount = 0
				fmt.Println("------------------------------------------------------------")
			}
		}
	}
}
