# ChainMaker YCSB Performance Test Tool

## Overview

This is a ChainMaker performance testing tool based on the YCSB (Yahoo! Cloud Serving Benchmark) workload. It runs performance tests using the `test_rwset_contract` contract and supports two key distribution modes:

- **Uniform**: all keys are accessed with equal probability
- **Zipfian**: a small number of hot keys are accessed more frequently, matching real-world access patterns

## Workloads

### Transaction Shape

Each transaction contains:
- **Read set**: 5 distinct keys (uniqueness guaranteed)
- **Write set**: 5 distinct keys (uniqueness guaranteed)
- **Keys may overlap between the read set and write set** (closer to real scenarios)

### Quick Command
```bash
go run ./ycsb -dist zipfian -records 1000000 -txcount 500000 -skew 0.1 -goroutines 20
```

### Experiment 1: Uniform

Keys are selected uniformly, so every key has the same access probability.

**Controlling conflicts**: tune `RecordCount` (keyspace size) to create different contention levels:
- Larger `RecordCount` → fewer conflicts (bigger keyspace, lower collision probability)
- Smaller `RecordCount` → more conflicts (smaller keyspace, higher collision probability)

**Example**:
```bash
go run ./ycsb -dist uniform -records 10000 -txcount 500000 -goroutines 20
```

### Experiment 2: Zipfian

Keys are selected using a Zipfian distribution; a small number of hot keys are accessed frequently (the 80/20 rule).

**Controlling conflicts**: tune `Skew` to create different contention levels:
- Smaller `Skew` (towards 0) → closer to uniform, fewer conflicts
- Larger `Skew` (towards 1 or > 1) → more skewed, hotter keys, more conflicts

**Skew guide**:
- `0.0`: perfectly uniform
- `0.5`: slightly skewed
- `0.9`: moderately skewed (YCSB default)

**Example**:
```bash
go run ./ycsb -dist zipfian -records 10000 -txcount 500000 -skew 0.1 -goroutines 20
```

## Usage

### CLI Arguments

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `-dist` | Distribution: `uniform` or `zipfian` | `uniform` | `-dist zipfian` |
| `-records` | Keyspace size (`RecordCount`) | `1000` | `-records 1000000` |
| `-txcount` | Total transactions | `10000` | `-txcount 500000` |
| `-goroutines` | Number of concurrent goroutines | `10` | `-goroutines 100` |
| `-skew` | Zipfian skew parameter (zipfian only) | `0.99` | `-skew 1.5` |
| `-key-size` | Fixed key length in bytes; pad with `x` if shorter and truncate if longer; `0` means no limit | `8` | `-key-size=16` |
| `-value-size` | Fixed value length in bytes; pad with `x` if shorter and truncate if longer; `0` means no limit | `16` | `-value-size=64` |

## Test Flow

1. **Contract check**: automatically checks whether `test_rwset_contract` is deployed; deploys it if missing
2. **Generate transactions**: generates the configured number of transactions based on distribution and parameters
   - Each transaction contains 5 unique read keys and 5 unique write keys
   - Keys and values are padded/truncated to the fixed length specified by `-key-size` / `-value-size`
   - Keys are selected using the specified distribution
3. **Send transactions**: concurrently sends transactions to the chain asynchronously using multiple goroutines

## Example Output

```
====================== ChainMaker YCSB Performance Test Tool ======================
Contract: test_rwset_contract
Method: test_rwset (read 3 keys, write 3 keys)
==================================================================================

====================== Checking Contract Status ======================
✓ Contract exists: test_rwset_contract (version: 1.0.0)

==================== Test Configuration ====================
Distribution: zipfian
Keyspace size (RecordCount): 1000
Total transactions: 10000
Concurrency: 10 goroutines
Zipfian skew: 0.99
===========================================================

Step 1/2: Generating transactions...
  - Using Zipfian distribution (skew=0.99)
✓ Generated 10000 transactions

Step 2/2: Sending transactions...
------------------------------------------------------------

------------------------------------------------------------
✓ All transactions sent
------------------------------------------------------------

====================== Done ======================
```

## Technical Details

### Key / Value Size Control

Keys and values are numeric strings, and their lengths can be fixed using `-key-size` and `-value-size`:
- If shorter than the target length, pad with `x` to the target length
- If longer than the target length, truncate to the target length
- If set to `0`, no padding/truncation is performed and the original string is kept

**Why the defaults** (based on `-records 1000000 -txcount 500000`):
- Max key index is `999999` (6 digits); `key-size=8` covers all indices and leaves 2 bytes of headroom
- Value format is `{txIndex}_{j}`, up to ~8 bytes; `value-size=16` leaves ~2× headroom

### Distribution Implementation

#### Uniform
- Uses Go's `rand.Int63n()` to generate uniformly distributed random numbers
- Each key has equal probability: P(k) = 1 / RecordCount

#### Zipfian
- Implements the classic Zipfian algorithm (YCSB-style)
- Probability mass function: P(k) = (1/k^s) / H(N,s)
  - k: key rank (starting from 1)
  - s: skew parameter (`skew`)
  - H(N,s): normalization constant (harmonic series)
- Samples using the inverse CDF method

### Ensuring Key Uniqueness

- The 5 read keys within a transaction are guaranteed to be distinct
- The 5 write keys within a transaction are guaranteed to be distinct
- Deduplication is done via `SelectUniqueKeys()`
- Up to count×200 attempts are made to avoid infinite loops

### Concurrency Safety

- All distribution generators protect internal state with `sync.Mutex`
- Each goroutine uses an independent RNG instance

## Configuration Files

- **SDK config**: `../config/sdk_config.yml`
- **Contract bytecode**: `../config/test_rwset.wasm`
- **Certificate paths**: configured in the SDK config file

## Contract Info

- **Name**: `test_rwset_contract`
- **Version**: `1.0.0`
- **Runtime**: WASMER
- **Method**: `test_rwset`
- **Deploy timeout**: 5 seconds

## Notes

1. **Async send**: to improve throughput, transactions are sent asynchronously (`withSyncResult: false`) without waiting for commit results
2. **Keyspace size**: `RecordCount` is recommended to be at least 10 to ensure sufficient key diversity
3. **Skew range**: recommended within 0.0-2.0; values > 2.0 may cause extreme concentration
4. **Tuning concurrency**: adjust `goroutines` based on chain performance and network bandwidth

## Performance Tuning Tips

1. **Increase concurrency**: raising `-goroutines` can improve send throughput
2. **Batch runs**: use scripts to sweep different parameter combinations and observe performance
3. **Monitor chain state**: use monitoring tools to track block production speed and conflict rate
4. **Adjust keyspace**: choose an appropriate `RecordCount` for your workload

## References

- [YCSB (Yahoo! Cloud Serving Benchmark)](https://github.com/brianfrankcooper/YCSB)
- [Zipfian Distribution - Wikipedia](https://en.wikipedia.org/wiki/Zipf%27s_law)
- ChainMaker official documentation

## Graph Generation

```bash
go run ./ycsb --generate-graph=true --txcount=50 --dist=zipfian --records=10000 --skew=0.7
```

Note the difference between `go run ./ycsb`, `go run ./main.go`, and `go run ycsb/*.go`.
