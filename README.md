# Vendor
This repository vendors its Go dependencies under `vendor/` for reproducible/offline builds.
When using YCSB or SMALLBANK to send transactions, you must add the `-mod=vendor` flag to your Go run commands to use the vendored dependencies.

# Note 
Before sending transactions, please configure the target machine's IP address in `config/sdk_config.yml`. Please locate the `node_addr` field, update the `127.0.0.1` address while keeping the default port `12301`.

# Send YCSB PAYLOAD
Read the ycsb/README.md to send ycsb transactions.
## Quick Start
To get started quickly, run the following command to send a YCSB transaction workload:
```bash
go run ./ycsb -mod=vendor -dist zipfian -records 1000000 -txcount 500000 -skew 0.1 -goroutines 20
```

# Send SMALLBANK PAYLOAD
Read the smallbank/README.md to send smallbank transactions.
## Quick Start
To get started quickly, run the following command to send a Smallbank transaction workload:
```bash
go run smallbank/*.go -mod=vendor -accounts 100000 -createAccountGoroutines 5 -goroutines 25 -txcount 500000 -dist zipfian -zipf 0.1
```
**Note**: The value of -createAccountGoroutines should be kept relatively low to ensure all accounts are created successfully.
