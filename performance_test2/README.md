# ChainMaker 性能压测工具（分散冲突模式）

## 功能说明

基于 `test_rwset_contract` 合约的性能压测工具，支持构造不同冲突率场景的交易进行性能测试。本工具采用**分散冲突模式**，将冲突交易分散到多个 key 上，更真实地模拟实际业务场景。

## 冲突率原理

**冲突率** 指的是在单个区块内确定性地产生冲突的交易比例，但与 `performance_test1` 不同的是，**冲突交易被分散到多个 key 上**，而不是全部访问同一个 key。

- **0% 冲突率**：所有交易访问各自独立的 key，无冲突
- **10% 冲突率**：10% 的交易产生冲突（分散到多个 key），90% 的交易访问各自独立的 key（无冲突）
- **50% 冲突率**：50% 的交易产生冲突（分散到多个 key），50% 的交易访问各自独立的 key（无冲突）
- **100% 冲突率**：所有交易产生冲突（分散到多个 key）

**实现方式**：
```
冲突交易数 = totalTxCount × conflictRate
无冲突交易数 = totalTxCount × (1 - conflictRate)
冲突交易分散到 conflictKeyCount 个不同的 key 上
```

### 示例：10% 冲突率，1000 笔交易，10 个冲突 key

- **冲突交易数**: 1000 × 0.1 = 100 笔
  - 这 100 笔交易**分散**到 10 个不同的 key：`conflict_key_0` ~ `conflict_key_9`
  - 每个冲突 key 约有 10 笔交易访问
  - 分配方式：轮询分配（Round-Robin）
    - 交易 0, 10, 20, 30, ... 访问 `conflict_key_0`
    - 交易 1, 11, 21, 31, ... 访问 `conflict_key_1`
    - ...
    - 交易 9, 19, 29, 39, ... 访问 `conflict_key_9`

- **无冲突交易数**: 1000 × 0.9 = 900 笔
  - 每笔交易访问唯一的 key：`unique_key_0`, `unique_key_1`, ..., `unique_key_899`
  - 每个 key 只被一笔交易访问，完全无冲突

- **交易顺序**: 生成后使用 Fisher-Yates 算法随机打乱，确保冲突交易和非冲突交易混合分布

### 可视化示例

假设有 1000 笔交易，10% 冲突率，10 个冲突 key：

```
无冲突交易（900 笔）:
unique_key_0    → 1 笔交易
unique_key_1    → 1 笔交易
unique_key_2    → 1 笔交易
...
unique_key_899  → 1 笔交易

冲突交易（100 笔）:
conflict_key_0  → 10 笔交易  ⚠️ 冲突
conflict_key_1  → 10 笔交易  ⚠️ 冲突
conflict_key_2  → 10 笔交易  ⚠️ 冲突
...
conflict_key_9  → 10 笔交易  ⚠️ 冲突
```

## 与其他压测工具的区别

| 特性 | performance_test<br/>（概率性冲突） | performance_test1<br/>（集中冲突） | **performance_test2<br/>（分散冲突）** |
|------|-------------------------------|-------------------------------|----------------------------------|
| **冲突模式** | 概率性：从固定大小的 key pool 随机选择 | 确定性：所有冲突交易访问同一个 key | **确定性：冲突交易分散到多个 key** |
| **10% 冲突率** | keyPoolSize = 900，随机选择 | 100 笔交易访问 `conflict_key` | **100 笔交易分散到 N 个 `conflict_key_i`** |
| **冲突分布** | 随机分布，冲突程度不确定 | 集中冲突，所有冲突交易在同一 key | **分散冲突，多个冲突点** |
| **可控性** | 较低，实际冲突率有波动 | 高，精确控制冲突交易数量 | **高，精确控制冲突数量和分散程度** |
| **真实性** | 中等，模拟随机访问 | 低，极端场景（单点热 key） | **高，模拟多个热 key 的真实场景** |
| **适用场景** | 模拟真实场景的随机访问模式 | 测试极端单点冲突性能 | **测试多点冲突的真实业务场景** |

## 使用方法

### 1. 运行默认场景（10% 冲突率，分散到 10 个 key）

```bash
go run performance_test2/main.go
```

默认配置：
- 冲突率：10%（ConflictRate: 0.1）
- 冲突 key 数量：10（ConflictKeyCount: 10）
- 总交易数：1,000,000
- 并发数：100 goroutines

### 2. 运行多个场景

编辑 `main.go` 第 62-72 行，取消注释想要测试的场景：

```go
testConfigs := []PerfTestConfig{
    {ConflictRate: 0.1, ConflictKeyCount: 10, TotalTxCount: 1000000, GoroutineCount: 100}, // 10% 冲突率，分散到 10 个 key
    // 可以添加更多场景
    // {ConflictRate: 0.0, ConflictKeyCount: 1, TotalTxCount: 1000, GoroutineCount: 10},   // 0% 冲突率
    // {ConflictRate: 0.1, ConflictKeyCount: 5, TotalTxCount: 1000, GoroutineCount: 10},   // 10% 冲突率，分散到 5 个 key
    // {ConflictRate: 0.3, ConflictKeyCount: 10, TotalTxCount: 1000, GoroutineCount: 10},  // 30% 冲突率，分散到 10 个 key
    // {ConflictRate: 0.5, ConflictKeyCount: 20, TotalTxCount: 1000, GoroutineCount: 10},  // 50% 冲突率，分散到 20 个 key
    // {ConflictRate: 1.0, ConflictKeyCount: 50, TotalTxCount: 1000, GoroutineCount: 10},  // 100% 冲突率，分散到 50 个 key
}
```

### 3. 自定义参数

修改 `PerfTestConfig` 结构体参数：
- `ConflictRate`: 冲突率 (0.0 - 1.0)
  - 0.0 = 0% 冲突率（所有交易访问各自独立的 key）
  - 0.1 = 10% 冲突率（10% 交易产生冲突）
  - 0.5 = 50% 冲突率（50% 交易产生冲突）
  - 1.0 = 100% 冲突率（所有交易产生冲突）
- `ConflictKeyCount`: **冲突 key 数量**（冲突交易分散到多少个不同的 key 上）
  - 数值越小，每个 key 上的冲突越集中
  - 数值越大，冲突越分散
  - 建议设置为区块容量的 1%-10%（例如区块容量 1000，设置 10-100）
- `TotalTxCount`: 总交易数
- `GoroutineCount`: 并发 goroutine 数量

### 4. 调整冲突 key 数量的影响

假设有 1000 笔交易，10% 冲突率（100 笔冲突交易）：

| ConflictKeyCount | 每个冲突 key 的交易数 | 冲突密度 | 适用场景 |
|------------------|---------------------|---------|----------|
| 1 | 100 | 极高 | 等同于 performance_test1，单点热 key |
| 5 | 20 | 高 | 少量热 key，高冲突场景 |
| 10 | 10 | 中 | 多个热 key，真实业务场景 |
| 20 | 5 | 低 | 冲突分散，接近真实分布 |
| 50 | 2 | 极低 | 大量 key，轻微冲突 |

## 测试流程

1. **检查合约**：自动检查 `test_rwset_contract` 合约是否已部署，未部署则自动部署
2. **生成交易**：根据冲突率和冲突 key 数量生成交易
   - 无冲突交易：每笔交易使用唯一的 key（`unique_key_0`, `unique_key_1`, ...）
   - 冲突交易：使用轮询方式分配到多个冲突 key（`conflict_key_0`, `conflict_key_1`, ...）
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

### 冲突交易（分散到多个冲突 key）

| 参数 | 说明 | 取值 |
|------|------|------|
| `read_key` | 读取的 key | `conflict_key_0`, `conflict_key_1`, ..., `conflict_key_{N-1}`<br/>（轮询分配） |
| `read_field` | 读取的 field | 空字符串（合约使用默认值 "data"） |
| `write_key` | 写入的 key | 与 `read_key` 相同 |
| `write_field` | 写入的 field | 空字符串（合约使用默认值 "data"） |
| `write_value` | 写入的值 | `value_{txIndex}_{timestamp}` |

实际访问的完整 key：`conflict_key_{i}#data`（i = 0 ~ N-1）

**分配算法**：
```go
conflictKeyIndex := i % config.ConflictKeyCount
conflictKey := fmt.Sprintf("conflict_key_%d", conflictKeyIndex)
```

## 输出示例

```
====================== ChainMaker 性能压测工具 (分散冲突) ======================
合约: test_rwset_contract
方法: test_rwset
场景: 不同冲突率下的性能测试（分散冲突模式）
==========================================================================


==================== 测试场景 1 ====================
冲突率: 10.0%
冲突 key 数量: 10
总交易数: 1000000
并发数: 100 goroutines
==================================================

步骤 1/3: 生成交易...
  - 无冲突交易数: 900000 (每笔交易访问唯一的 key)
  - 冲突交易数: 100000 (分散到 10 个冲突 key，平均每个 key 约 10000 笔交易)
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

1. **分散冲突**：与 `performance_test1` 不同，本工具将冲突交易分散到多个 key 上，更真实地模拟实际业务场景
2. **异步发送**：为了提高吞吐量，交易采用异步方式发送（`withSyncResult: false`），不等待上链结果
3. **错误处理**：每个 goroutine 只打印前 5 个错误，避免刷屏
4. **冲突 key 数量**：建议根据实际业务场景设置合理的冲突 key 数量
5. **轮询分配**：冲突交易使用取模运算轮询分配到各个冲突 key 上
6. **调整并发**：可根据实际需求调整 `GoroutineCount` 参数来控制并发数

## 配置文件

- **SDK 配置**: `../config/sdk_config.yml`
- **合约字节码**: `../config/test_rwset.wasm`
- **证书路径**: 在 `sdk_config.yml` 中配置

**重要提示**：必须从项目根目录运行程序，以便正确访问 `../config/` 目录：
```bash
# 正确的运行方式（在项目根目录）
cd /path/to/sdk-go-demo
go run performance_test2/main.go

# 错误的运行方式（在 performance_test2 目录内）
cd performance_test2
go run main.go  # 这样会找不到 ../config/ 目录
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
    testRwsetByteCodePath = "../config/test_rwset.wasm"
    sdkConfigOrg1Client1Path = "../config/sdk_config.yml"
    blockTxCapacity       = 1000   // 区块交易容量（用于参考）
)
```

## 使用场景

### 何时使用 performance_test2（分散冲突）

✅ **推荐使用**：
- 模拟真实业务场景的多热点 key 冲突
- 测试多个账户/资源同时被访问的情况
- 评估分布式冲突处理能力
- 需要可重复且真实的测试结果
- 测试链在多点冲突下的并发处理能力

### 何时使用 performance_test1（集中冲突）

- 测试极端单点热 key 场景
- 评估完全集中冲突的最差性能
- 压力测试单 key 的最大吞吐量

### 何时使用 performance_test（概率性冲突）

- 模拟完全随机访问模式
- 测试随机分布的冲突处理能力
- 评估平均情况下的系统性能

## 参数建议

根据不同测试目的，建议的参数配置：

### 1. 真实业务模拟（推荐）
```go
{ConflictRate: 0.1, ConflictKeyCount: 10, TotalTxCount: 10000, GoroutineCount: 10}
```
- 10% 冲突率，分散到 10 个 key
- 模拟真实的多热点场景

### 2. 高冲突场景测试
```go
{ConflictRate: 0.3, ConflictKeyCount: 5, TotalTxCount: 10000, GoroutineCount: 10}
```
- 30% 冲突率，分散到 5 个 key
- 每个冲突 key 约有 600 笔交易

### 3. 极端冲突测试
```go
{ConflictRate: 0.5, ConflictKeyCount: 1, TotalTxCount: 10000, GoroutineCount: 10}
```
- 50% 冲突率，集中到 1 个 key
- 等同于 performance_test1 的 50% 冲突场景

### 4. 分散冲突测试
```go
{ConflictRate: 0.2, ConflictKeyCount: 20, TotalTxCount: 10000, GoroutineCount: 10}
```
- 20% 冲突率，分散到 20 个 key
- 每个冲突 key 约有 100 笔交易

### 5. 无冲突基线测试
```go
{ConflictRate: 0.0, ConflictKeyCount: 1, TotalTxCount: 10000, GoroutineCount: 10}
```
- 0% 冲突率
- 所有交易访问各自独立的 key，测试最佳性能

## 性能分析建议

使用不同的 `ConflictKeyCount` 值进行对比测试，分析系统在不同冲突密度下的表现：

```go
testConfigs := []PerfTestConfig{
    // 对比测试：相同冲突率，不同冲突密度
    {ConflictRate: 0.1, ConflictKeyCount: 1, TotalTxCount: 10000, GoroutineCount: 10},   // 高密度
    {ConflictRate: 0.1, ConflictKeyCount: 5, TotalTxCount: 10000, GoroutineCount: 10},   // 中密度
    {ConflictRate: 0.1, ConflictKeyCount: 10, TotalTxCount: 10000, GoroutineCount: 10},  // 低密度
    {ConflictRate: 0.1, ConflictKeyCount: 20, TotalTxCount: 10000, GoroutineCount: 10},  // 极低密度
}
```

观察 TPS、区块大小、确认时间等指标，找到系统的性能拐点。
