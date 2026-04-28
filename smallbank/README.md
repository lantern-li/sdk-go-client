# SmallBank Performance Test Tool

## Overview

This is a ChainMaker performance testing tool based on the SmallBank benchmark. SmallBank is a standard blockchain performance benchmark that simulates banking workloads and supports two key distribution modes:

- **Uniform**: all accounts are accessed with equal probability
- **Zipfian**: a small number of hot accounts are accessed more frequently, matching real-world access patterns

## Data Model

The SmallBank contract maintains three tables:

- `account_{name}` → `customer_id` (mapping from account name to customer ID)
- `saving_{id}` → `i64 balance` (savings account balance)
- `checking_{id}` → `i64 balance` (checking account balance)

## Transaction Types

The benchmark includes 5 transaction types with different read/write characteristics:

| Transaction | Read/Write Pattern | Call Probability | Amount | Description |
|------------|---------------------|------------------|--------|-------------|
| `deposit_checking` | R2W1 | 15% | 200 | Deposit into a checking account |
| `transact_saving` | R2W1 | 15% | 500 | Deposit/withdraw from a savings account |
| `amalgamate` | R5W3 | 15% | N/A | Move all funds from one account to another |
| `write_check` | R3W1 | 15% | 50 | Write a check (charge a penalty if funds are insufficient) |
| `send_payment` | R4W2 | 40% | 50 | Transfer between accounts |

**Note**: R2W1 means 2 reads and 1 write; R5W3 means 5 reads and 3 writes; and so on.

## Experiment Setup

### Initial State

Before each run:
- Create X accounts (configured by `--accounts`)
- Initialize each account with:
  - Savings balance: 1,000,000 tokens
  - Checking balance: 1,000,000 tokens

**Tip**: Control the number of concurrent transaction-sending goroutines via `-goroutines`.

## Usage

### Prerequisites

1. ChainMaker blockchain is running
2. The SmallBank contract WASM file is located at `config/smallbank.wasm`
3. The SDK config file is located at `config/sdk_config.yml`

### CLI Arguments

```
-accounts int       Number of accounts to create (default: 10000)
-goroutines int     Number of concurrent goroutines (default: 10)
-createAccountGoroutines int  Concurrent goroutines for account creation (default: 15)
-dist string        Distribution type: uniform or zipfian (default: "uniform")
-zipf float         Zipfian parameter 0.0-1.0 (default: 0.9)
-txcount int        Number of transactions to send (default: 100000)
```

### Run the Benchmark

#### Basic Run

#### Run with Zipfian Distribution
```bash
go run smallbank/*.go -accounts 100000 -createAccountGoroutines 5 -goroutines 25 -txcount 500000 -dist zipfian -zipf 0.1
```

### Example Output

**Final Summary**:
```
==================== Final Summary ====================
Duration: 45.23 seconds
Total transactions: 50000
Succeeded: 49995
Failed: 5

Transaction mix:
  - deposit_checking (R2W1): 7500 (15.0%)
  - transact_saving  (R2W1): 7498 (15.0%)
  - amalgamate       (R5W3): 7502 (15.0%)
  - write_check      (R3W1): 7500 (15.0%)
  - send_payment     (R4W2): 20000 (40.0%)
=======================================================
```

## Understanding Zipfian Distribution

Zipfian distribution models real-world access patterns where some items are accessed much more frequently than others (e.g., hot accounts in a banking system).

- **Zipf = 0.0**: close to uniform (low contention)
- **Zipf = 0.5**: moderately skewed (medium contention)
- **Zipf = 0.9**: highly skewed (high contention / hot accounts)
- **Zipf = 0.99**: extremely skewed (very high contention)

Higher Zipf values lead to more transaction conflicts, which helps stress-test the blockchain's concurrency control.

## Technical Details

### Distribution Implementation

#### Uniform
- Uses Go's standard library RNG to generate uniformly distributed random numbers
- Each account is selected with equal probability: P(k) = 1 / AccountCount

#### Zipfian
- Implements the classic Zipfian algorithm (YCSB-style)
- Probability mass function: P(k) = (1/k^s) / H(N,s)
  - k: rank of the account (starting from 1)
  - s: skew parameter (`zipf`)
  - H(N,s): normalization constant (harmonic series)
- Samples using the inverse CDF method (Inverse CDF)

### Concurrency Safety

- All distribution generators protect internal state with `sync.Mutex`
- Each goroutine uses an independent RNG instance
- Counters are updated via atomic operations

## Notes

1. **Async send**: to improve throughput, transactions are sent asynchronously (`withSyncResult: false`) without waiting for commit results
2. **Account pool size**: `accounts` is recommended to be at least 10 to ensure sufficient selection diversity
3. **Zipf range**: recommended within 0.0-1.0; values > 1.0 may cause extreme concentration
4. **Tuning concurrency**: adjust `goroutines` based on chain performance and network bandwidth

## References

- [SmallBank Benchmark](https://github.com/ooibc88/blockbench)
- [Zipfian Distribution - Wikipedia](https://en.wikipedia.org/wiki/Zipf%27s_law)
- [YCSB (Yahoo! Cloud Serving Benchmark)](https://github.com/brianfrankcooper/YCSB)
- ChainMaker official documentation
