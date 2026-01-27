# SmallBank 性能测试工具

## 功能说明

基于 SmallBank 基准测试的 ChainMaker 性能测试工具。SmallBank 是标准的区块链性能基准测试，模拟银行业务场景，支持两种键分布模式：

- **Uniform（均匀分布）**：所有账户被访问的概率相同
- **Zipfian（齐普夫分布）**：少数热点账户被高频访问，符合现实世界的访问模式

## 数据模型

SmallBank 合约维护 3 个表：

- `account_{name}` → `customer_id` （账户名到客户ID的映射）
- `saving_{id}` → `i64 balance` （储蓄账户余额）
- `checking_{id}` → `i64 balance` （支票账户余额）

## 交易类型

测试包含 5 种交易类型，具有不同的读写特征：

| 交易类型 | 读写模式 | 调用概率 | 金额 | 说明 |
|---------|---------|---------|------|------|
| `deposit_checking` | R2W1 | 15% | 200 | 向支票账户存款 |
| `transact_saving` | R2W1 | 15% | 500 | 储蓄账户存取款 |
| `amalgamate` | R5W3 | 15% | 无 | 将一个账户的所有资金转移到另一个账户 |
| `write_check` | R3W1 | 15% | 50 | 开支票（余额不足时扣除罚金） |
| `send_payment` | R4W2 | 40% | 50 | 账户间转账 |

**注意**：R2W1 表示 2 次读操作和 1 次写操作；R5W3 表示 5 次读操作和 3 次写操作，以此类推。

## 实验设置

### 初始条件

每次实验开始前：
- 创建 X 个账户（通过 `--accounts` 参数配置）
- 每个账户初始化为：
  - 储蓄账户余额：1,000,000 代币
  - 支票账户余额：1,000,000 代币

**说明**：客户端通过调整 `-goroutines` 参数控制并发发送交易的协程数。服务器端的 CPU 核心数由长安链配置决定。

## 使用方法

### 前置条件

1. ChainMaker 区块链必须正在运行
2. SmallBank 合约 WASM 文件必须位于 `config/smallbank.wasm`
3. SDK 配置文件必须位于 `config/sdk_config.yml`

### 命令行参数

```
-accounts int       要创建的账户数量（默认：10000）
-goroutines int     并发 goroutine 数量（默认：10）
-dist string        分布类型：uniform 或 zipfian（默认："uniform"）
-zipf float         Zipfian 分布参数 0.0-1.0（默认：0.9）
-txcount int        要发送的交易数量（默认：100000）
```

### 运行测试

#### 基本运行
```bash
go run smallbank/*.go -accounts 1000 -goroutines 10 -txcount 50000 -dist uniform
```

#### 使用 Zipfian 分布
```bash
go run smallbank/*.go -accounts 1000 -goroutines 10 -txcount 50000 -dist zipfian -zipf 0.9
```

### 输出示例

**最终统计**：
```
==================== 最终统计 ====================
测试时长: 45.23 秒
总交易数: 50000
成功交易: 49995
失败交易: 5

交易类型分布:
  - deposit_checking (R2W1): 7500 (15.0%)
  - transact_saving  (R2W1): 7498 (15.0%)
  - amalgamate       (R5W3): 7502 (15.0%)
  - write_check      (R3W1): 7500 (15.0%)
  - send_payment     (R4W2): 20000 (40.0%)
=================================================
```

## 配置文件

### SDK 配置

确保 SDK 配置位于根目录的 config 文件夹：

```bash
config/sdk_config.yml
```

配置应包含：
- 正确的链 ID 和组织 ID
- 有效的用户证书和私钥
- 正确的节点地址

### SmallBank 合约

确保 SmallBank WASM 合约可用：

```bash
config/smallbank.wasm
```

## 理解 Zipfian 分布

Zipfian 分布模拟真实世界的访问模式，其中某些项目被访问的频率远高于其他项目（例如，银行系统中的热门账户）。

- **Zipf = 0.0**：接近均匀分布（低冲突）
- **Zipf = 0.5**：中等倾斜（中等冲突）
- **Zipf = 0.9**：高度倾斜（高冲突，热点账户）
- **Zipf = 0.99**：极度倾斜（非常高的冲突）

更高的 Zipf 值会创造更多的交易冲突，测试区块链的并发控制机制。

## 数据收集

对于自动化数据收集和分析，将输出重定向到文件：

## 技术细节

### 分布算法实现

#### Uniform 分布
- 使用 Go 标准库的随机数生成器生成均匀分布的随机数
- 每个账户被选中的概率相同：P(k) = 1 / AccountCount

#### Zipfian 分布
- 实现了经典的 Zipfian 分布算法（参考 YCSB）
- 概率质量函数：P(k) = (1/k^s) / H(N,s)
  - k: 账户的排名（从 1 开始）
  - s: 偏斜参数（zipf）
  - H(N,s): 归一化常数（调和级数）
- 使用逆累积分布函数（Inverse CDF）方法进行采样

### 并发安全

- 所有分布生成器都使用 `sync.Mutex` 保护内部状态
- 每个 goroutine 使用独立的随机数生成器实例
- 使用原子操作进行计数器更新

## 注意事项

1. **异步发送**：为了提高吞吐量，交易采用异步方式发送（`withSyncResult: false`），不等待上链结果
2. **账户池限制**：`accounts` 建议不小于 10，以保证账户选择的多样性
3. **Zipf 参数范围**：建议在 0.0-1.0 之间，超过 1.0 可能导致极端的访问集中
4. **并发数调整**：根据链的性能和网络带宽调整 `goroutines` 参数
5. **服务器端配置**：多核可扩展性实验需要在长安链服务器端配置 CPU 核心数

## 参考资料

- [SmallBank Benchmark](https://github.com/ooibc88/blockbench)
- [Zipfian Distribution - Wikipedia](https://en.wikipedia.org/wiki/Zipf%27s_law)
- [YCSB (Yahoo! Cloud Serving Benchmark)](https://github.com/brianfrankcooper/YCSB)
- ChainMaker 官方文档
