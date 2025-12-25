package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
)

const (
	createContractTimeout = 5

	// test_rwset 合约配置
	testRwsetContractName = "test_rwset_contract"
	testRwsetVersion      = "1.0.0"
	testRwsetByteCodePath = "./config/test_rwset.wasm"

	sdkConfigOrg1Client1Path = "./config/sdk_config.yml"

	blockTxCapacity = 1000
)

// 交易数据结构
type Transaction struct {
	ReadKey    string
	ReadField  string
	WriteKey   string
	WriteField string
	WriteValue string
}

// 性能测试配置
type PerfTestConfig struct {
	ConflictRate   float64 // 冲突率 (0.0 - 1.0)
	TotalTxCount   int     // 总交易数
	GoroutineCount int     // 并发 goroutine 数量
}

func main() {
	fmt.Println("====================== ChainMaker 性能压测工具 ======================")
	fmt.Println("合约: test_rwset_contract")
	fmt.Println("方法: test_rwset")
	fmt.Println("场景: 不同冲突率下的性能测试")
	fmt.Println("===================================================================\n")

	// 创建客户端
	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	// 检查并部署合约
	ensureContractDeployed(client)

	// 测试场景配置
	testConfigs := []PerfTestConfig{
		{ConflictRate: 0.9, TotalTxCount: 1000000, GoroutineCount: 100}, // 10% 冲突率
		// 可以添加更多场景
		// {ConflictRate: 0.0, TotalTxCount: 1000, GoroutineCount: 10},  // 0% 冲突率
		// {ConflictRate: 0.3, TotalTxCount: 1000, GoroutineCount: 10},  // 30% 冲突率
		// {ConflictRate: 0.5, TotalTxCount: 1000, GoroutineCount: 10},  // 50% 冲突率
		// {ConflictRate: 1.0, TotalTxCount: 1000, GoroutineCount: 10},  // 100% 冲突率
	}

	// 执行测试场景
	for i, config := range testConfigs {
		fmt.Printf("\n\n==================== 测试场景 %d ====================\n", i+1)
		fmt.Printf("冲突率: %.1f%%\n", config.ConflictRate*100)
		fmt.Printf("总交易数: %d\n", config.TotalTxCount)
		fmt.Printf("并发数: %d goroutines\n", config.GoroutineCount)
		fmt.Println("==================================================\n")

		runPerfTest(client, config)

		// 场景之间暂停一下
		if i < len(testConfigs)-1 {
			fmt.Println("\n等待 3 秒后开始下一个测试场景...")
			time.Sleep(3 * time.Second)
		}
	}

	fmt.Println("\n\n====================== 所有测试完成 ======================")
}

// ensureContractDeployed 确保合约已部署
func ensureContractDeployed(client *sdk.ChainClient) {
	fmt.Println("====================== 检查合约状态 ======================")

	contract, err := client.GetContractInfo(testRwsetContractName)
	if err != nil {
		if strings.Contains(err.Error(), "contract not exist") {
			fmt.Printf("合约[%s]不存在，开始部署...\n", testRwsetContractName)
			deployContract(client)
		} else {
			panicErr(err)
		}
	} else {
		fmt.Printf("✓ 合约已存在: %s (version: %s)\n\n", contract.Name, contract.Version)
	}
}

// deployContract 部署合约
func deployContract(client *sdk.ChainClient) {
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

	fmt.Printf("✓ 合约部署成功 - BlockHeight:%d, TxId:%s\n\n", resp.TxBlockHeight, resp.TxId)
}

// runPerfTest 运行性能测试
func runPerfTest(client *sdk.ChainClient, config PerfTestConfig) {
	// 1. 生成交易
	fmt.Println("步骤 1/3: 生成交易...")
	transactions := generateTransactions(config)
	fmt.Printf("✓ 已生成 %d 笔交易\n\n", len(transactions))

	// 2. 打乱交易顺序
	fmt.Println("步骤 2/3: 打乱交易顺序...")
	shuffleTransactions(transactions)
	fmt.Printf("✓ 交易顺序已打乱\n\n")

	// 3. 发送交易
	fmt.Println("步骤 3/3: 发送交易到链上...")
	fmt.Println("------------------------------------------------------------")

	// 分配交易到各个 goroutine
	txPerGoroutine := len(transactions) / config.GoroutineCount
	var wg sync.WaitGroup

	for i := 0; i < config.GoroutineCount; i++ {
		start := i * txPerGoroutine
		end := start + txPerGoroutine
		if i == config.GoroutineCount-1 {
			end = len(transactions) // 最后一个 goroutine 处理剩余的所有交易
		}

		wg.Add(1)
		go func(goroutineId int, txs []Transaction) {
			defer wg.Done()
			sendTransactions(client, txs, goroutineId)
		}(i, transactions[start:end])
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	fmt.Println("\n------------------------------------------------------------")
	fmt.Println("✓ 所有交易发送完成")
	fmt.Println("------------------------------------------------------------")
}

// generateTransactions 生成交易
// 冲突率计算：假设有 N 笔交易，冲突率为 R
// keyPoolSize = N * (1 - R)，这样 N 笔交易访问 keyPoolSize 个不同的 key
func generateTransactions(config PerfTestConfig) []Transaction {
	transactions := make([]Transaction, config.TotalTxCount)

	// 计算 key pool 大小
	// 冲突率越高，key pool 越小，交易访问相同 key 的概率越大
	keyPoolSize := int(float64(blockTxCapacity) * (1.0 - config.ConflictRate))
	if keyPoolSize < 1 {
		keyPoolSize = 1 // 至少有 1 个 key
	} // todo:keyPoolSize控制在1到1000之间，因为一个区块的交易容量是1000

	fmt.Printf("  - Key Pool 大小: %d", keyPoolSize)

	// 生成 key pool
	keyPool := make([]string, keyPoolSize)
	for i := 0; i < keyPoolSize; i++ {
		keyPool[i] = fmt.Sprintf("key_%d", i)
	}

	// 生成交易
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < config.TotalTxCount; i++ {
		// 从 key pool 中随机选择 key
		readKey := keyPool[rand.Intn(keyPoolSize)]
		writeKey := keyPool[rand.Intn(keyPoolSize)]

		transactions[i] = Transaction{
			ReadKey:    readKey,
			ReadField:  "", // 空，使用合约默认值 "data"
			WriteKey:   writeKey,
			WriteField: "", // 空，使用合约默认值 "data"
			WriteValue: fmt.Sprintf("value_%d_%d", i, time.Now().UnixNano()),
		}
	}

	return transactions
}

// shuffleTransactions 打乱交易顺序（Fisher-Yates 洗牌算法）
func shuffleTransactions(transactions []Transaction) {
	rand.Seed(time.Now().UnixNano())
	for i := len(transactions) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		transactions[i], transactions[j] = transactions[j], transactions[i]
	}
}

// sendTransactions 发送交易
func sendTransactions(client *sdk.ChainClient, transactions []Transaction, goroutineId int) {
	for i, tx := range transactions {
		// 构造参数
		kvs := []*common.KeyValuePair{
			{Key: "read_key", Value: []byte(tx.ReadKey)},
			{Key: "read_field", Value: []byte(tx.ReadField)},
			{Key: "write_key", Value: []byte(tx.WriteKey)},
			{Key: "write_field", Value: []byte(tx.WriteField)},
			{Key: "write_value", Value: []byte(tx.WriteValue)},
		}

		// 发送交易（异步，不等待上链结果以提高吞吐量）
		resp, err := client.InvokeContract(
			testRwsetContractName,
			"test_rwset",
			"", // txId 自动生成
			kvs,
			-1,    // timeout
			false, // withSyncResult = false (异步，不等待上链)
		)

		if err != nil {
			if i < 5 { // 只打印前几个错误，避免刷屏
				fmt.Printf("[Goroutine %d] ❌ 交易发送失败: %v\n", goroutineId, err)
			}
			continue
		}

		if resp.Code != common.TxStatusCode_SUCCESS {
			if i < 5 {
				fmt.Printf("[Goroutine %d] ❌ 交易失败 - Code:%d, Msg:%s\n",
					goroutineId, resp.Code, resp.Message)
			}
			continue
		}
	}
}

func panicErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
