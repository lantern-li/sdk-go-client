# ChainMaker 性能压测工具（确定性冲突模式）

## 功能说明

基于 `test_rwset_contract` 合约的性能压测工具，支持构造不同冲突率场景的交易进行性能测试。本工具采用**确定性冲突模式**，与 `performance_test` 的概率性冲突不同。

## 冲突率原理

**冲突率** 指的是在单个区块内确定性地产生冲突的交易比例：

- **0% 冲突率**：所有交易访问各自独立的 key，无冲突
- **10% 冲突率**：10% 的交易访问相同的 key（完全冲突），90% 的交易访问各自独立的 key（无冲突）
- **50% 冲突率**：50% 的交易访问相同的 key（完全冲突），50% 的交易访问各自独立的 key（无冲突）
- **100% 冲突率**：所有交易访问同一个 key，完全冲突

**实现方式**：
```
冲突交易数 = totalTxCount × conflictRate
无冲突交易数 = totalTxCount × (1 - conflictRate)
```

### 示例：10% 冲突率，1000 笔交易

- **冲突交易数**: 1000 × 0.1 = 100 笔
  - 这 100 笔交易全部访问同一个 key：`conflict_key`
  - 读写同一个完整 key：`conflict_key#data`

- **无冲突交易数**: 1000 × 0.9 = 900 笔
  - 每笔交易访问唯一的 key：`unique_key_0`, `unique_key_1`, ..., `unique_key_899`
  - 每个 key 只被一笔交易访问，完全无冲突

- **交易顺序**: 生成后使用 Fisher-Yates 算法随机打乱，确保冲突交易和非冲突交易混合分布

## 与 performance_test 的区别

| 特性 | performance_test（概率性冲突） | performance_test1（确定性冲突） |
|------|-------------------------------|-------------------------------|
| **冲突模式** | 概率性：所有交易从固定大小的 key pool 随机选择 | 确定性：明确指定冲突和非冲突交易数量 |
| **10% 冲突率** | keyPoolSize = 900，所有交易从 900 个 key 中随机选择 | 100 笔交易访问相同 key，900 笔交易访问各自独立的 key |
| **冲突分布** | 随机分布，冲突程度不确定 | 确定分布，冲突交易完全冲突，非冲突交易完全独立 |
| **可控性** | 较低，实际冲突率有波动 | 高，精确控制冲突交易数量 |
| **适用场景** | 模拟真实场景的随机访问模式 | 精确测试特定冲突比例下的性能 |

## 使用方法

### 1. 运行默认场景（10% 冲突率）

```bash
go run performance_test1/main.go
```

默认配置：
- 冲突率：10%（ConflictRate: 0.1）
- 总交易数：1,000,000
- 并发数：100 goroutines

### 2. 运行多个场景

编辑 `main.go` 第 62-69 行，取消注释想要测试的场景：

```go
testConfigs := []PerfTestConfig{
    {ConflictRate: 0.1, TotalTxCount: 1000000, GoroutineCount: 100}, // 10% 冲突率
    // 可以添加更多场景
    // {ConflictRate: 0.0, TotalTxCount: 1000, GoroutineCount: 10},  // 0% 冲突率
    // {ConflictRate: 0.3, TotalTxCount: 1000, GoroutineCount: 10},  // 30% 冲突率
    // {ConflictRate: 0.5, TotalTxCount: 1000, GoroutineCount: 10},  // 50% 冲突率
    // {ConflictRate: 1.0, TotalTxCount: 1000, GoroutineCount: 10},  // 100% 冲突率
}
```

### 3. 自定义参数

修改 `PerfTestConfig` 结构体参数：
- `ConflictRate`: 冲突率 (0.0 - 1.0)
  - 0.0 = 0% 冲突率（所有交易访问各自独立的 key）
  - 0.1 = 10% 冲突率（10% 交易访问相同 key，90% 交易访问各自独立的 key）
  - 0.5 = 50% 冲突率（50% 交易访问相同 key，50% 交易访问各自独立的 key）
  - 1.0 = 100% 冲突率（所有交易访问同一个 key）
- `TotalTxCount`: 总交易数
- `GoroutineCount`: 并发 goroutine 数量

## 测试流程

1. **检查合约**：自动检查 `test_rwset_contract` 合约是否已部署，未部署则自动部署
2. **生成交易**：根据冲突率生成指定数量的冲突和非冲突交易
   - 无冲突交易：每笔交易使用唯一的 key（`unique_key_0`, `unique_key_1`, ...）
   - 冲突交易：所有交易使用相同的 key（`conflict_key`）
3. **打乱顺序**：使用 Fisher-Yates 算法随机打乱交易顺序
4. **发送交易**：启动多个 goroutine 并发异步发送交易到链上

## 交易结构

### 无冲突交易（每笔交易访问唯一的 key）

| 参数 | 说明 | 取值 |
|------|------|------|
| `read_key` | 读取的 key | `unique_key_0`, `unique_key_1`, `unique_key_2`, ... |
| `read_field` | 读取的 field | 空字符串（合约使用默认值 "data"） |
| `write_key` | 写入的 key | 与 `read_key` 相同 |
| `write_field` | 写入的 field | 空字符串（合约使用默认值 "data"） |
| `write_value` | 写入的值 | `value_{txIndex}_{timestamp}` |

实际访问的完整 key：`unique_key_{index}#data`

### 冲突交易（所有交易访问相同的 key）

| 参数 | 说明 | 取值 |
|------|------|------|
| `read_key` | 读取的 key | `conflict_key`（固定值） |
| `read_field` | 读取的 field | 空字符串（合约使用默认值 "data"） |
| `write_key` | 写入的 key | `conflict_key`（固定值） |
| `write_field` | 写入的 field | 空字符串（合约使用默认值 "data"） |
| `write_value` | 写入的值 | `value_{txIndex}_{timestamp}` |

实际访问的完整 key：`conflict_key#data`

## 输出示例

```
====================== ChainMaker 性能压测工具 (确定性冲突) ======================
合约: test_rwset_contract
方法: test_rwset
场景: 不同冲突率下的性能测试（确定性冲突模式）
==========================================================================


==================== 测试场景 1 ====================
冲突率: 10.0%
总交易数: 1000000
并发数: 100 goroutines
==================================================

步骤 1/3: 生成交易...
  - 无冲突交易数: 900000 (每笔交易访问唯一的 key)
  - 冲突交易数: 100000 (所有交易访问相同的 key)
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

1. **确定性冲突**：与 `performance_test` 不同，本工具采用确定性冲突模式，精确控制冲突交易数量
2. **异步发送**：为了提高吞吐量，交易采用异步方式发送（`withSyncResult: false`），不等待上链结果
3. **错误处理**：每个 goroutine 只打印前 5 个错误，避免刷屏
4. **冲突 key**：所有冲突交易访问同一个固定 key（`conflict_key`）
5. **无冲突 key**：每笔无冲突交易访问唯一的 key（`unique_key_{index}`）
6. **调整并发**：可根据实际需求调整 `GoroutineCount` 参数来控制并发数

## 配置文件

- **SDK 配置**: `./config/sdk_config.yml`
- **合约字节码**: `./config/test_rwset.wasm`
- **证书路径**: 在 `sdk_config.yml` 中配置

**重要提示**：必须从项目根目录运行程序，以便正确访问 `./config/` 目录：
```bash
# 正确的运行方式（在项目根目录）
cd /path/to/sdk-go-demo
go run performance_test1/main.go

# 错误的运行方式（在 performance_test1 目录内）
cd performance_test1
go run main.go  # 这样会找不到 ./config/ 目录
```

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
    blockTxCapacity       = 1000   // 区块交易容量（用于统计，不影响实际冲突计算）
)
```

## 使用场景

### 何时使用 performance_test1（确定性冲突）

- 需要精确控制冲突交易比例
- 测试极端冲突场景（如 100% 冲突）
- 评估完全冲突 vs 完全无冲突的性能差异
- 需要可重复的测试结果

### 何时使用 performance_test（概率性冲突）

- 模拟真实业务场景的随机访问模式
- 测试随机分布的冲突处理能力
- 评估平均情况下的系统性能