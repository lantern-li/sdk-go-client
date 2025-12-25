# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a ChainMaker SDK Go demo repository that demonstrates various use cases of the ChainMaker blockchain SDK. ChainMaker is a blockchain framework with support for different runtime types (WASMER for WASM contracts, EVM for Solidity contracts) and cryptographic modes (standard and GM/GuoMi).

## Project Structure

The repository contains multiple standalone demo applications, each in its own directory:

- **`main.go`** - Basic contract deployment and invocation example
- **`simple/`** - Extended example with contract lifecycle management, trust root updates, and query operations
- **`simple-gm/`** - Same as simple but configured for GM (GuoMi/Chinese national cryptographic standards)
- **`1000cert/`** - Performance testing demo with multiple concurrent goroutines sending transactions
- **`1000cert_evm/`** - EVM contract performance testing with ABI encoding/decoding
- **`spv/`** - SPV (Simplified Payment Verification) service client demo using gRPC
- **`config/`** - Shared configuration directory containing:
  - `sdk_config.yml` - SDK configuration (chain ID, org ID, node addresses, certificates)
  - `crypto-config/` - Certificate and key files for multiple organizations
  - `rust-fact-2.0.0.wasm` - WASM contract bytecode

Each demo directory may have its own `config/` subdirectory with demo-specific configurations.

## Building and Running

Each demo is a standalone Go application with its own `main.go`. To run a specific demo:

```bash
# Run the basic demo
go run main.go

# Run the simple demo
go run simple/main.go

# Run the GM demo
go run simple-gm/main.go

# Run the 1000 cert performance test
go run 1000cert/main.go

# Run the EVM performance test
go run 1000cert_evm/main.go

# Run the SPV demo
go run spv/main.go
```

The project uses Go 1.18+ and requires the following main dependencies:
- `chainmaker.org/chainmaker/sdk-go/v2` - ChainMaker Go SDK
- `chainmaker.org/chainmaker/pb-go/v2` - Protocol buffer definitions
- `chainmaker.org/chainmaker/common/v2` - Common utilities and crypto functions

## Key Concepts

### SDK Client Initialization

All demos create a ChainMaker client using the SDK config file:

```go
client, err := sdk.NewChainClient(
    sdk.WithConfPath(sdkConfigOrg1Client1Path),
)
```

The SDK config (`sdk_config.yml`) contains:
- Chain ID and organization ID
- User certificates and private keys (both TLS and signing)
- Node addresses with connection counts
- Trust root CA paths for TLS verification

### Contract Lifecycle

1. **Create Contract**: Deploy a new smart contract to the blockchain
   - Supports WASMER (WASM) and EVM runtime types
   - Requires contract name, version, bytecode path, and runtime type
   - Returns transaction response with block height and tx ID

2. **Invoke Contract**: Execute a contract method that modifies state
   - Accepts method name and key-value pairs as parameters
   - Can be synchronous (wait for block inclusion) or asynchronous
   - Returns transaction response

3. **Query Contract**: Execute a read-only contract method
   - Does not modify blockchain state
   - Faster than invoke operations

### Certificate Mode and Custom Signers

The SDK supports using custom signers (`CertModeSigner`) to sign transactions with specific certificates. This is useful for:
- Performance testing with multiple user identities (see `1000cert/` examples)
- Trust root management operations requiring multi-org endorsements
- Dynamic certificate selection at runtime

### Performance Testing Patterns

The `1000cert/` demos demonstrate high-throughput transaction patterns:
- Multiple goroutines sending transactions concurrently
- Atomic counters for round-robin certificate selection across multiple orgs
- TPS (transactions per second) monitoring with periodic reporting
- Asynchronous transaction submission (`withSyncResult: false`) for maximum throughput

### EVM Contract Interaction

The `1000cert_evm/` demo shows how to work with EVM contracts:
- Load and parse contract ABI from JSON file
- Use `abi.Pack()` to encode method calls with parameters
- Encode packed data as hex string for the "data" parameter
- Both invoke and query operations use the same ABI encoding pattern

## Configuration Notes

When adding new demos or modifying existing ones:
- Each demo references its own `sdk_config.yml` with appropriate relative paths
- Certificate paths must match the directory structure under `crypto-config/`
- The repository includes certs for multiple organizations (wx-org1 through wx-org5)
- Node addresses are configured in `sdk_config.yml` - update these to match your ChainMaker deployment
- Contract names must be unique per chain (e.g., "claim001", "claim003", "proof_evm")

## Common Operations

### Contract Creation with Error Handling

Check if contract exists before attempting to create it:

```go
contract, err := client.GetContractInfo(contractName)
if err != nil {
    if strings.Contains(err.Error(), "contract not exist") {
        // Create contract
    }
}
```

### Multi-Organization Endorsement

For chain configuration updates (e.g., trust root changes), collect endorsements from multiple admin users:

```go
endorsers := make([]*common.EndorsementEntry, 0)
endorser1, _ := sdkutils.MakeEndorserWithPath(admin1KeyPath, admin1CertPath, payload)
endorser2, _ := sdkutils.MakeEndorserWithPath(admin2KeyPath, admin2CertPath, payload)
endorser3, _ := sdkutils.MakeEndorserWithPath(admin3KeyPath, admin3CertPath, payload)
endorsers = append(endorsers, endorser1, endorser2, endorser3)
client.SendChainConfigUpdateRequest(payload, endorsers, -1, true)
```

### Block and Transaction Queries

The SDK provides methods to query blockchain data:
- `GetTxByTxId(txId)` - Get transaction details by transaction ID
- `GetBlockByHeight(height, withRWSet)` - Get block data by height
- `GetChainConfig()` - Get current chain configuration
