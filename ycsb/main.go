package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"

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
)

// Transaction 交易数据结构
type Transaction struct {
	ReadKeys  []string // 5个读key
	WriteKeys []string // 5个写key
	Values    []string // 5个写value
}

// TestConfig 测试配置
type TestConfig struct {
	DistributionType string  // "uniform" 或 "zipfian"
	RecordCount      int64   // 键空间大小
	TotalTxCount     int     // 总交易数
	GoroutineCount   int     // 并发goroutine数量
	Skew             float64 // Zipfian分布的偏斜参数（仅用于zipfian）
	KeySize          int     // key的固定字节长度（0表示不限制）
	ValueSize        int     // value的固定字节长度（0表示不限制）
}

var (
	// 命令行参数
	distType       = flag.String("dist", "uniform", "分布类型: uniform 或 zipfian")
	recordCount    = flag.Int64("records", 1000, "键空间大小 (RecordCount)")
	totalTxCount   = flag.Int("txcount", 10000, "总交易数")
	goroutineCount = flag.Int("goroutines", 10, "并发goroutine数量")
	skew           = flag.Float64("skew", 0.99, "Zipfian分布的偏斜参数 (0.0-2.0)")
	generateGraph  = flag.Bool("generate-graph", false, "生成交易依赖图（不发送到链上）")
	keySize        = flag.Int("key-size", 8, "key的固定字节长度（0表示不限制）")
	valueSize      = flag.Int("value-size", 16, "value的固定字节长度（0表示不限制）")
)

func main() {
	flag.Parse()

	fmt.Println("====================== ChainMaker YCSB 性能测试工具 ======================")
	fmt.Println("合约: test_rwset_contract")
	fmt.Println("方法: test_rwset (读5个key，写5个key)")
	fmt.Println("========================================================================")

	// 验证参数
	if *distType != "uniform" && *distType != "zipfian" {
		log.Fatal("错误: -dist 参数必须是 'uniform' 或 'zipfian'")
	}

	if *recordCount < 10 {
		log.Fatal("错误: -records 参数必须 >= 10")
	}

	if *totalTxCount < 1 {
		log.Fatal("错误: -txcount 参数必须 >= 1")
	}

	if *goroutineCount < 1 {
		log.Fatal("错误: -goroutines 参数必须 >= 1")
	}

	// 构建测试配置
	config := TestConfig{
		DistributionType: *distType,
		RecordCount:      *recordCount,
		TotalTxCount:     *totalTxCount,
		GoroutineCount:   *goroutineCount,
		Skew:             *skew,
		KeySize:          *keySize,
		ValueSize:        *valueSize,
	}

	// 如果指定了 --generate-graph 参数，则生成图而不是发送交易
	if *generateGraph {
		err := GenerateTransactionGraph(config, "./")
		if err != nil {
			log.Fatalf("生成交易依赖图失败: %v", err)
		}
		return
	}

	// 否则，执行正常的性能测试流程
	// 创建客户端
	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	// 检查并部署合约
	ensureContractDeployed(client)

	// 运行测试
	runTest(client, config)

	fmt.Println("\n\n====================== 测试完成 ======================")
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

// runTest 运行测试
func runTest(client *sdk.ChainClient, config TestConfig) {
	fmt.Println("==================== 测试配置 ====================")
	fmt.Printf("分布类型: %s\n", config.DistributionType)
	fmt.Printf("键空间大小 (RecordCount): %d\n", config.RecordCount)
	fmt.Printf("总交易数: %d\n", config.TotalTxCount)
	fmt.Printf("并发数: %d goroutines\n", config.GoroutineCount)
	if config.DistributionType == "zipfian" {
		fmt.Printf("Zipfian Skew 参数: %.2f\n", config.Skew)
	}
	fmt.Println("=================================================")

	// 1. 生成交易
	fmt.Println("步骤 1/2: 生成交易...")
	transactions := generateTransactions(config)
	fmt.Printf("✓ 已生成 %d 笔交易\n\n", len(transactions))

	// 2. 发送交易
	fmt.Println("步骤 2/2: 发送交易到链上...")
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
		go func(txs []Transaction) {
			defer wg.Done()
			sendTransactions(client, txs)
		}(transactions[start:end])
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	fmt.Println("\n------------------------------------------------------------")
	fmt.Println("✓ 所有交易发送完成")
	fmt.Println("------------------------------------------------------------")
}

// generateTransactions 生成交易
func generateTransactions(config TestConfig) []Transaction {
	transactions := make([]Transaction, config.TotalTxCount)

	// 创建分布
	var dist Distribution
	if config.DistributionType == "uniform" {
		dist = NewUniformDistribution(config.RecordCount)
		fmt.Printf("  - 使用均匀分布 (Uniform)\n")
	} else {
		dist = NewZipfianDistribution(config.RecordCount, config.Skew)
		fmt.Printf("  - 使用齐普夫分布 (Zipfian, skew=%.2f)\n", config.Skew)
	}

	selector := NewKeySelector(dist)

	// 生成每笔交易
	for i := 0; i < config.TotalTxCount; i++ {
		// 选择5个不同的读key
		readKeyIndexes := selector.SelectUniqueKeys(5)
		readKeys := make([]string, 5)
		for j := 0; j < 5; j++ {
			readKeys[j] = padOrTrunc(fmt.Sprintf("%d", readKeyIndexes[j]), config.KeySize)
		}

		// 选择5个不同的写key
		writeKeyIndexes := selector.SelectUniqueKeys(5)
		writeKeys := make([]string, 5)
		values := make([]string, 5)
		for j := 0; j < 5; j++ {
			writeKeys[j] = padOrTrunc(fmt.Sprintf("%d", writeKeyIndexes[j]), config.KeySize)
			values[j] = padOrTrunc(fmt.Sprintf("%d_%d", i, j), config.ValueSize)
		}

		transactions[i] = Transaction{
			ReadKeys:  readKeys,
			WriteKeys: writeKeys,
			Values:    values,
		}
	}

	return transactions
}

// sendTransactions 发送交易
func sendTransactions(client *sdk.ChainClient, transactions []Transaction) {
	for _, tx := range transactions {
		// 构造参数 - 5个读key和5个写key
		kvs := []*common.KeyValuePair{
			// 第一组读参数
			{Key: "read_key", Value: []byte(tx.ReadKeys[0])},
			{Key: "read_field", Value: []byte("")},

			// 第二组读参数
			{Key: "read_key2", Value: []byte(tx.ReadKeys[1])},
			{Key: "read_field2", Value: []byte("")},

			// 第三组读参数
			{Key: "read_key3", Value: []byte(tx.ReadKeys[2])},
			{Key: "read_field3", Value: []byte("")},

			// 第四组读参数
			{Key: "read_key4", Value: []byte(tx.ReadKeys[3])},
			{Key: "read_field4", Value: []byte("")},

			// 第五组读参数
			{Key: "read_key5", Value: []byte(tx.ReadKeys[4])},
			{Key: "read_field5", Value: []byte("")},

			// 第一组写参数
			{Key: "write_key", Value: []byte(tx.WriteKeys[0])},
			{Key: "write_field", Value: []byte("")},
			{Key: "write_value", Value: []byte(tx.Values[0])},

			// 第二组写参数
			{Key: "write_key2", Value: []byte(tx.WriteKeys[1])},
			{Key: "write_field2", Value: []byte("")},
			{Key: "write_value2", Value: []byte(tx.Values[1])},

			// 第三组写参数
			{Key: "write_key3", Value: []byte(tx.WriteKeys[2])},
			{Key: "write_field3", Value: []byte("")},
			{Key: "write_value3", Value: []byte(tx.Values[2])},

			// 第四组写参数
			{Key: "write_key4", Value: []byte(tx.WriteKeys[3])},
			{Key: "write_field4", Value: []byte("")},
			{Key: "write_value4", Value: []byte(tx.Values[3])},

			// 第五组写参数
			{Key: "write_key5", Value: []byte(tx.WriteKeys[4])},
			{Key: "write_field5", Value: []byte("")},
			{Key: "write_value5", Value: []byte(tx.Values[4])},
		}

		// 发送交易（异步，不等待上链结果以提高吞吐量）
		_, _ = client.InvokeContract(
			testRwsetContractName,
			"test_rwset",
			"", // txId 自动生成
			kvs,
			-1,    // timeout
			false, // withSyncResult = false (异步，不等待上链)
		)
	}
}

func panicErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

// padOrTrunc 将字符串填充或截断到指定字节长度，size<=0时原样返回
func padOrTrunc(s string, size int) string {
	if size <= 0 {
		return s
	}
	b := []byte(s)
	if len(b) >= size {
		return string(b[:size])
	}
	padded := make([]byte, size)
	copy(padded, b)
	for i := len(b); i < size; i++ {
		padded[i] = 'x'
	}
	return string(padded)
}
