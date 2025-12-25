# ChainMaker 性能压测工具

## 功能说明

基于 `test_rwset_contract` 合约的性能压测工具，支持构造不同冲突率场景的交易进行性能测试。

## 冲突率原理

**冲突率** 指的是在单个区块内交易之间访问相同 key 的概率：

- **0% 冲突率**：区块内每笔交易访问不同的 key，无冲突
- **10% 冲突率**：区块内约 10% 的交易会与其他交易访问相同的 key
- **50% 冲突率**：区块内约 50% 的交易会与其他交易访问相同的 key
- **100% 冲突率**：区块内所有交易访问同一个 key，完全冲突

**实现方式**：
```
keyPoolSize = blockTxCapacity × (1 - conflictRate)
```

其中 `blockTxCapacity = 1000`（单个区块的交易容量上限）

例如：10% 冲突率
- keyPoolSize = 1000 × (1 - 0.1) = 900
- 所有交易从 900 个 key 的池子中随机选择读写 key
- 区块内平均每个 key 被访问约 1.11 次

## 使用方法

### 1. 运行默认场景（10% 冲突率）

```bash
go run performance_test/main.go
```

默认配置：
- 冲突率：10%（ConflictRate: 0.9）
- 总交易数：1,000,000
- 并发数：100 goroutines

### 2. 运行多个场景

编辑 `main.go` 第 61-68 行，取消注释想要测试的场景：

```go
testConfigs := []PerfTestConfig{
    {ConflictRate: 0.9, TotalTxCount: 1000000, GoroutineCount: 100}, // 10% 冲突率
    // 可以添加更多场景
    // {ConflictRate: 0.0, TotalTxCount: 1000, GoroutineCount: 10},  // 0% 冲突率
    // {ConflictRate: 0.3, TotalTxCount: 1000, GoroutineCount: 10},  // 30% 冲突率
    // {ConflictRate: 0.5, TotalTxCount: 1000, GoroutineCount: 10},  // 50% 冲突率
    // {ConflictRate: 1.0, TotalTxCount: 1000, GoroutineCount: 10},  // 100% 冲突率
}
```

### 3. 自定义参数

修改 `PerfTestConfig` 结构体参数：
- `ConflictRate`: 冲突率 (0.0 - 1.0)，注意值越大表示冲突率越低
  - 0.0 = 100% 冲突率（所有交易访问同一个 key）
  - 0.9 = 10% 冲突率（keyPoolSize = 900）
  - 1.0 = 0% 冲突率（keyPoolSize = 1000，无冲突）
- `TotalTxCount`: 总交易数
- `GoroutineCount`: 并发 goroutine 数量

## 测试流程

1. **检查合约**：自动检查 `test_rwset_contract` 合约是否已部署，未部署则自动部署
2. **生成交易**：根据冲突率计算 keyPoolSize，生成指定数量的交易
3. **打乱顺序**：使用 Fisher-Yates 算法随机打乱交易顺序
4. **发送交易**：启动多个 goroutine 并发异步发送交易到链上

## 交易结构

每笔交易包含以下参数：

| 参数 | 说明 | 取值 |
|------|------|------|
| `read_key` | 读取的 key | 从 key pool 中随机选择（如 `key_0`, `key_1`, ...） |
| `read_field` | 读取的 field | 空字符串（合约使用默认值 "data"） |
| `write_key` | 写入的 key | 从 key pool 中随机选择 |
| `write_field` | 写入的 field | 空字符串（合约使用默认值 "data"） |
| `write_value` | 写入的值 | `value_{txIndex}_{timestamp}` |

实际访问的完整 key 为：`{read_key/write_key}#data`

## 输出示例

```
====================== ChainMaker 性能压测工具 ======================
合约: test_rwset_contract
方法: test_rwset
场景: 不同冲突率下的性能测试
===================================================================


==================== 测试场景 1 ====================
冲突率: 10.0%
总交易数: 1000000
并发数: 100 goroutines
==================================================

步骤 1/3: 生成交易...
  - Key Pool 大小: 900
✓ 已生成 1000000 笔交易

步骤 2/3: 打乱交易顺序...
✓ 交易顺序已打乱

步骤 3/3: 发送交易到链上...
------------------------------------------------------------

------------------------------------------------------------
✓ 所有交易发送完成
------------------------------------------------------------


====================== 所有测试完成 ======================
```

## 注意事项

1. **异步发送**：为了提高吞吐量，交易采用异步方式发送（`withSyncResult: false`），不等待上链结果
2. **错误处理**：每个 goroutine 只打印前 5 个错误，避免刷屏
3. **冲突率参数**：`ConflictRate` 的值实际上表示"无冲突率"，即值越大，实际冲突越少
4. **Key Pool 范围**：keyPoolSize 控制在 1 到 1000 之间（由 `blockTxCapacity` 常量定义）
5. **调整并发**：可根据实际需求调整 `GoroutineCount` 参数来控制并发数

## 配置文件

- **SDK 配置**: `./config/sdk_config.yml`
- **合约字节码**: `./config/test_rwset.wasm`
- **证书路径**: 在 `sdk_config.yml` 中配置

## 合约信息

- **合约名称**: `test_rwset_contract`
- **合约版本**: `1.0.0`
- **运行时**: WASMER
- **调用方法**: `test_rwset`
- **部署超时**: 5 秒

## 常量配置

```go
const (
    createContractTimeout = 5      // 合约部署超时时间（秒）
    testRwsetContractName = "test_rwset_contract"
    testRwsetVersion      = "1.0.0"
    testRwsetByteCodePath = "./config/test_rwset.wasm"
    sdkConfigOrg1Client1Path = "./config/sdk_config.yml"
    blockTxCapacity       = 1000   // 区块交易容量
)
```