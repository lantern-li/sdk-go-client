package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
)

const (
	createContractTimeout = 5
	smallbankContractName = "smallbank_contract"
	smallbankVersion      = "1.0.0"
	smallbankByteCodePath = "./config/smallbank.wasm"
	sdkConfigPath         = "./config/sdk_config.yml"
)

// Transaction type probabilities
const (
	ProbDepositChecking = 0.15 // 15%
	ProbTransactSaving  = 0.15 // 15%
	ProbAmalgamate      = 0.15 // 15%
	ProbWriteCheck      = 0.15 // 15%
	ProbSendPayment     = 0.40 // 40%
)

// Transaction amounts
const (
	AmountDepositChecking = 200
	AmountTransactSaving  = 500
	AmountWriteCheck      = 50
	AmountSendPayment     = 50
	InitialBalance        = 1000000
)

// Command line flags
var (
	accountPoolSize = flag.Int("accounts", 10000, "Number of accounts to create")
	goroutineCount  = flag.Int("goroutines", 10, "Number of concurrent goroutines")
	distribution    = flag.String("dist", "uniform", "Distribution type: uniform or zipfian")
	zipfParam       = flag.Float64("zipf", 0.9, "Zipfian distribution parameter (0.0-1.0)")
	txCount         = flag.Int("txcount", 100000, "Number of transactions to send")
)

// Statistics
var (
	totalTx           int64
	successTx         int64
	failedTx          int64
	txTypeCount       [5]int64 // Count for each transaction type
	startTime         time.Time
	testRunning       int32 = 1
	accountNamePrefix       = "Alice_"
)

// Transaction types
const (
	TxDepositChecking = iota
	TxTransactSaving
	TxAmalgamate
	TxWriteCheck
	TxSendPayment
)

func main() {
	flag.Parse()

	fmt.Println("==================== SmallBank Performance Test ====================")
	fmt.Printf("Configuration:\n")
	fmt.Printf("  - Account Pool Size: %d\n", *accountPoolSize)
	fmt.Printf("  - Concurrent Goroutines: %d\n", *goroutineCount)
	fmt.Printf("  - Transaction Count: %d\n", *txCount)
	fmt.Printf("  - Distribution: %s\n", *distribution)
	if *distribution == "zipfian" {
		fmt.Printf("  - Zipf Parameter: %.2f\n", *zipfParam)
	}
	fmt.Println("====================================================================")

	// Create SDK client
	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigPath),
	)
	panicErr(err)

	// Check and deploy contract if needed
	ensureContractDeployed(client)

	// Create accounts
	createAccounts(client, *accountPoolSize)

	// Run performance test
	runPerformanceTest(client)
}

func ensureContractDeployed(client *sdk.ChainClient) {
	fmt.Println("\n==================== Checking Contract ====================")
	contract, err := client.GetContractInfo(smallbankContractName)
	if err != nil {
		if strings.Contains(err.Error(), "contract not exist") {
			fmt.Printf("Contract [%s] does not exist, deploying...\n", smallbankContractName)
			deployContract(client)
		} else {
			panicErr(err)
		}
	} else {
		fmt.Printf("Contract already exists: %s (version: %s)\n", contract.Name, contract.Version)
	}
}

func deployContract(client *sdk.ChainClient) {
	fmt.Println("==================== Deploying SmallBank Contract ====================")
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

	fmt.Printf("✓ Contract deployed successfully!\n")
	fmt.Printf("  - Block Height: %d\n", resp.TxBlockHeight)
	fmt.Printf("  - Tx ID: %s\n", resp.TxId)
}

func createAccounts(client *sdk.ChainClient, count int) {
	fmt.Printf("\n==================== Creating %d Accounts ====================\n", count)

	// Create accounts with multiple goroutines
	numWorkers := 5 //创建账户时，并发的协程数要低一些，避免打爆交易池，最终创建交易没被调度执行，落库等。
	accountChan := make(chan int, count)

	// Fill the channel with account indices
	for i := 0; i < count; i++ {
		accountChan <- i
	}
	close(accountChan)

	var wg sync.WaitGroup
	var created int64
	var failed int64

	// Start workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for accountIdx := range accountChan {
				accountName := fmt.Sprintf("%s%d", accountNamePrefix, accountIdx)
				customerId := fmt.Sprintf("C%d", accountIdx)

				kvs := []*common.KeyValuePair{
					{Key: "name", Value: []byte(accountName)},
					{Key: "customer_id", Value: []byte(customerId)},
				}

				// Use async mode for faster account creation
				_, err := client.InvokeContract(smallbankContractName, "create_account", "", kvs, -1, false)
				if err != nil {
					atomic.AddInt64(&failed, 1)
					fmt.Printf("Failed to create account %s: %v\n", accountName, err)
				} else {
					atomic.AddInt64(&created, 1)
				}
			}
		}(w)
	}

	wg.Wait()

	fmt.Printf("\n✓ Account creation completed!\n")
	fmt.Printf("  - Total: %d\n", count)
	fmt.Printf("  - Created: %d\n", created)
	fmt.Printf("  - Failed: %d\n", failed)

	// Wait a bit for transactions to be processed
	fmt.Println("\nWaiting 5 seconds for transactions to be processed...")
	time.Sleep(5 * time.Second)
}

func runPerformanceTest(client *sdk.ChainClient) {
	fmt.Printf("\n==================== 开始性能测试 ====================\n")
	fmt.Printf("将发送 %d 笔交易，使用 %d 个并发 goroutine\n", *txCount, *goroutineCount)

	// Create distribution
	var dist Distribution
	if *distribution == "zipfian" {
		dist = NewZipfianDistribution(*accountPoolSize, *zipfParam)
		fmt.Printf("使用 Zipfian 分布，参数 %.2f\n", *zipfParam)
	} else {
		dist = NewUniformDistribution(*accountPoolSize)
		fmt.Printf("使用 Uniform 分布\n")
	}

	// Start workers
	var wg sync.WaitGroup
	startTime = time.Now()

	for i := 0; i < *goroutineCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendTransactions(client, dist)
		}()
	}

	// Wait until we reach the transaction count
	for atomic.LoadInt64(&totalTx) < int64(*txCount) {
		time.Sleep(100 * time.Millisecond)
	}
	atomic.StoreInt32(&testRunning, 0)

	wg.Wait()

	// Print final statistics
	printFinalStatistics()
}

func sendTransactions(client *sdk.ChainClient, dist Distribution) {
	for atomic.LoadInt32(&testRunning) == 1 {
		// Check if we've reached the transaction count limit
		if atomic.LoadInt64(&totalTx) >= int64(*txCount) {
			break
		}

		// Select transaction type based on probability
		txType := selectTransactionType()

		// Generate transaction parameters
		var kvs []*common.KeyValuePair
		var method string

		switch txType {
		case TxDepositChecking:
			accountIdx := dist.Next()
			accountName := fmt.Sprintf("%s%d", accountNamePrefix, accountIdx)
			kvs = []*common.KeyValuePair{
				{Key: "name", Value: []byte(accountName)},
				{Key: "amount", Value: []byte(strconv.Itoa(AmountDepositChecking))},
			}
			method = "deposit_checking"

		case TxTransactSaving:
			accountIdx := dist.Next()
			accountName := fmt.Sprintf("%s%d", accountNamePrefix, accountIdx)
			kvs = []*common.KeyValuePair{
				{Key: "name", Value: []byte(accountName)},
				{Key: "amount", Value: []byte(strconv.Itoa(AmountTransactSaving))},
			}
			method = "transact_saving"

		case TxAmalgamate:
			accountIdx1 := dist.Next()
			accountIdx2 := dist.Next()
			// Ensure different accounts
			for accountIdx2 == accountIdx1 {
				accountIdx2 = dist.Next()
			}
			accountName1 := fmt.Sprintf("%s%d", accountNamePrefix, accountIdx1)
			accountName2 := fmt.Sprintf("%s%d", accountNamePrefix, accountIdx2)
			kvs = []*common.KeyValuePair{
				{Key: "name1", Value: []byte(accountName1)},
				{Key: "name2", Value: []byte(accountName2)},
			}
			method = "amalgamate"

		case TxWriteCheck:
			accountIdx := dist.Next()
			accountName := fmt.Sprintf("%s%d", accountNamePrefix, accountIdx)
			kvs = []*common.KeyValuePair{
				{Key: "name", Value: []byte(accountName)},
				{Key: "amount", Value: []byte(strconv.Itoa(AmountWriteCheck))},
			}
			method = "write_check"

		case TxSendPayment:
			accountIdx1 := dist.Next()
			accountIdx2 := dist.Next()
			// Ensure different accounts
			for accountIdx2 == accountIdx1 {
				accountIdx2 = dist.Next()
			}
			sendName := fmt.Sprintf("%s%d", accountNamePrefix, accountIdx1)
			destName := fmt.Sprintf("%s%d", accountNamePrefix, accountIdx2)
			kvs = []*common.KeyValuePair{
				{Key: "send_name", Value: []byte(sendName)},
				{Key: "dest_name", Value: []byte(destName)},
				{Key: "amount", Value: []byte(strconv.Itoa(AmountSendPayment))},
			}
			method = "send_payment"
		}

		// Send transaction (async mode for maximum throughput)
		_, err := client.InvokeContract(smallbankContractName, method, "", kvs, -1, false)
		if err != nil {
			atomic.AddInt64(&failedTx, 1) // todo:什么时候会走到这里？
		} else {
			atomic.AddInt64(&successTx, 1)
			atomic.AddInt64(&txTypeCount[txType], 1)
		}
		atomic.AddInt64(&totalTx, 1)
	}
}

func selectTransactionType() int {
	r := rand.Float64()
	if r < ProbDepositChecking {
		return TxDepositChecking
	} else if r < ProbDepositChecking+ProbTransactSaving {
		return TxTransactSaving
	} else if r < ProbDepositChecking+ProbTransactSaving+ProbAmalgamate {
		return TxAmalgamate
	} else if r < ProbDepositChecking+ProbTransactSaving+ProbAmalgamate+ProbWriteCheck {
		return TxWriteCheck
	}
	return TxSendPayment
}

func printFinalStatistics() {
	elapsed := time.Since(startTime).Seconds()

	fmt.Println("\n==================== 最终统计 ====================")
	fmt.Printf("测试时长: %.2f 秒\n", elapsed)
	fmt.Printf("总交易数: %d\n", totalTx)
	fmt.Printf("成功交易: %d\n", successTx)
	fmt.Printf("失败交易: %d\n", failedTx)
	fmt.Println("\n交易类型分布:")
	fmt.Printf("  - deposit_checking (R2W1): %d (%.1f%%)\n", txTypeCount[TxDepositChecking], float64(txTypeCount[TxDepositChecking])/float64(totalTx)*100)
	fmt.Printf("  - transact_saving  (R2W1): %d (%.1f%%)\n", txTypeCount[TxTransactSaving], float64(txTypeCount[TxTransactSaving])/float64(totalTx)*100)
	fmt.Printf("  - amalgamate       (R5W3): %d (%.1f%%)\n", txTypeCount[TxAmalgamate], float64(txTypeCount[TxAmalgamate])/float64(totalTx)*100)
	fmt.Printf("  - write_check      (R3W1): %d (%.1f%%)\n", txTypeCount[TxWriteCheck], float64(txTypeCount[TxWriteCheck])/float64(totalTx)*100)
	fmt.Printf("  - send_payment     (R4W2): %d (%.1f%%)\n", txTypeCount[TxSendPayment], float64(txTypeCount[TxSendPayment])/float64(totalTx)*100)
	fmt.Println("=================================================")
}

func panicErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
