# ChainMaker YCSB 性能测试工具

## 功能说明

基于 YCSB (Yahoo! Cloud Serving Benchmark) 基准测试的 ChainMaker 性能测试工具。使用 `test_rwset_contract` 合约进行性能测试，支持两种键分布模式：

- **Uniform（均匀分布）**：所有 key 被访问的概率相同
- **Zipfian（齐普夫分布）**：少数热点 key 被高频访问，符合现实世界的访问模式

## 测试场景

### 交易结构

每笔交易包含：
- **读集**：3个不同的 key（保证唯一性）
- **写集**：3个不同的 key（保证唯一性）
- **读集和写集之间的 key 可以重复**（符合真实场景）

### 实验一：Uniform 分布

键的选择服从均匀分布，所有 key 被访问的概率相同。

**冲突控制**：通过调整 `RecordCount`（键空间大小）来制造不同的冲突场景
- `RecordCount` 越大 → 冲突越少（key 空间大，碰撞概率低）
- `RecordCount` 越小 → 冲突越多（key 空间小，碰撞概率高）

**示例**：
```bash
# 低冲突场景（键空间大）
go run ycsb/*.go -dist uniform -records 10000 -txcount 10000

# 中等冲突场景
go run ycsb/*.go -dist uniform -records 1000 -txcount 10000

# 高冲突场景（键空间小）
go run ycsb/*.go -dist uniform -records 100 -txcount 10000
```

### 实验二：Zipfian 分布

键的选择服从齐普夫分布，少数热点 key 被频繁访问，符合 80/20 法则。

**冲突控制**：通过调整 `Skew` 参数来制造不同的冲突场景
- `Skew` 越小（趋向 0）→ 接近均匀分布，冲突较少
- `Skew` 越大（趋向 1 或大于 1）→ 高度倾斜，热点集中，冲突增多

**Skew 参数说明**：
- `0.0`：完全均匀分布
- `0.5`：轻度倾斜
- `0.99`：中等倾斜（YCSB 默认值）
- `1.0`：高度倾斜
- `1.5` 或更高：极度倾斜，绝大多数访问集中在极少数 key 上

**示例**：
```bash
# 轻度倾斜（接近均匀分布）
go run ycsb/*.go -dist zipfian -records 1000 -txcount 10000 -skew 0.3

# 中等倾斜（默认值）
go run ycsb/*.go -dist zipfian -records 1000 -txcount 10000 -skew 0.99

# 高度倾斜（热点明显）
go run ycsb/*.go -dist zipfian -records 1000 -txcount 10000 -skew 1.5
```

tail -f system.log | grep -i tps
scp -r root@192.168.1.10:/data/logs ~/Downloads/
Add tx failed, TxPool is full

## 使用方法

### 命令行参数

| 参数 | 说明 | 默认值 | 示例 |
|------|------|--------|------|
| `-dist` | 分布类型：`uniform` 或 `zipfian` | `uniform` | `-dist zipfian` |
| `-records` | 键空间大小（RecordCount） | `1000` | `-records 10000` |
| `-txcount` | 总交易数 | `10000` | `-txcount 100000` |
| `-goroutines` | 并发 goroutine 数量 | `10` | `-goroutines 100` |
| `-skew` | Zipfian 分布的偏斜参数（仅用于 zipfian） | `0.99` | `-skew 1.5` |

### 快速开始

#### 1. 运行 Uniform 分布测试

```bash
# 基本测试（默认参数）
go run ycsb/*.go

# 自定义参数
go run ycsb/*.go -dist uniform -records 5000 -txcount 50000 -goroutines 50
```

#### 2. 运行 Zipfian 分布测试

```bash
# 默认 Zipfian 参数（skew=0.99）
go run ycsb/*.go -dist zipfian -records 1000 -txcount 10000

# 高度倾斜场景（skew=1.5）
go run ycsb/*.go -dist zipfian -records 1000 -txcount 10000 -skew 1.5 -goroutines 50
```

## 测试流程

1. **检查合约**：自动检查 `test_rwset_contract` 合约是否已部署，未部署则自动部署
2. **生成交易**：根据分布类型和参数生成指定数量的交易
   - 每笔交易包含 3 个唯一的读 key 和 3 个唯一的写 key
   - 使用对应的分布算法选择 key
3. **发送交易**：启动多个 goroutine 并发异步发送交易到链上

## 输出示例

```
====================== ChainMaker YCSB 性能测试工具 ======================
合约: test_rwset_contract
方法: test_rwset (读3个key，写3个key)
========================================================================

====================== 检查合约状态 ======================
✓ 合约已存在: test_rwset_contract (version: 1.0.0)

==================== 测试配置 ====================
分布类型: zipfian
键空间大小 (RecordCount): 1000
总交易数: 10000
并发数: 10 goroutines
Zipfian Skew 参数: 0.99
=================================================

步骤 1/2: 生成交易...
  - 使用齐普夫分布 (Zipfian, skew=0.99)
✓ 已生成 10000 笔交易

步骤 2/2: 发送交易到链上...
------------------------------------------------------------

------------------------------------------------------------
✓ 所有交易发送完成
------------------------------------------------------------


====================== 测试完成 ======================
```

## 实验建议

### Uniform 分布实验

测试不同 RecordCount 下的性能表现：

```bash
# 实验 1: 低冲突（大键空间）
go run ycsb/*.go -dist uniform -records 10000 -txcount 50000 -goroutines 50

# 实验 2: 中等冲突
go run ycsb/*.go -dist uniform -records 1000 -txcount 50000 -goroutines 50

# 实验 3: 高冲突（小键空间）
go run ycsb/*.go -dist uniform -records 100 -txcount 50000 -goroutines 50

# 实验 4: 极高冲突
go run ycsb/*.go -dist uniform -records 50 -txcount 50000 -goroutines 50
```

### Zipfian 分布实验

测试不同 Skew 参数下的性能表现：

```bash
# 实验 1: 轻度倾斜（接近均匀）
go run ycsb/*.go -dist zipfian -records 1000 -txcount 50000 -goroutines 50 -skew 0.3

# 实验 2: 中等倾斜（YCSB 默认）
go run ycsb/*.go -dist zipfian -records 1000 -txcount 50000 -goroutines 50 -skew 0.99

# 实验 3: 高度倾斜
go run ycsb/*.go -dist zipfian -records 1000 -txcount 50000 -goroutines 50 -skew 1.3

# 实验 4: 极度倾斜（热点明显）
go run ycsb/*.go -dist zipfian -records 1000 -txcount 50000 -goroutines 50 -skew 1.8
```

### 组合实验

固定键空间，比较不同分布：

```bash
# Uniform 分布
go run ycsb/*.go -dist uniform -records 1000 -txcount 100000 -goroutines 100

# Zipfian 分布（中等倾斜）
go run ycsb/*.go -dist zipfian -records 1000 -txcount 100000 -goroutines 100 -skew 0.99

# Zipfian 分布（高度倾斜）
go run ycsb/*.go -dist zipfian -records 1000 -txcount 100000 -goroutines 100 -skew 1.5
```

## 技术细节

### 分布算法实现

#### Uniform 分布
- 使用 Go 标准库的 `rand.Int63n()` 生成均匀分布的随机数
- 每个 key 被选中的概率相同：P(k) = 1 / RecordCount

#### Zipfian 分布
- 实现了经典的 Zipfian 分布算法（参考 YCSB）
- 概率质量函数：P(k) = (1/k^s) / H(N,s)
  - k: key 的排名（从 1 开始）
  - s: 偏斜参数（skew）
  - H(N,s): 归一化常数（调和级数）
- 使用逆累积分布函数（Inverse CDF）方法进行采样

### Key 唯一性保证

- 每笔交易的 3 个读 key 保证互不相同
- 每笔交易的 3 个写 key 保证互不相同
- 使用 `SelectUniqueKeys()` 方法进行去重选择
- 最多尝试 count×10 次，避免死循环

### 并发安全

- 所有分布生成器都使用 `sync.Mutex` 保护内部状态
- 每个 goroutine 使用独立的随机数生成器实例

## 配置文件

- **SDK 配置**: `../config/sdk_config.yml`
- **合约字节码**: `../config/test_rwset.wasm`
- **证书路径**: 在 SDK 配置文件中指定

## 合约信息

- **合约名称**: `test_rwset_contract`
- **合约版本**: `1.0.0`
- **运行时**: WASMER
- **调用方法**: `test_rwset`
- **部署超时**: 5 秒

## 注意事项

1. **异步发送**：为了提高吞吐量，交易采用异步方式发送（`withSyncResult: false`），不等待上链结果
2. **键空间限制**：`RecordCount` 建议不小于 10，以保证 key 选择的多样性
3. **Skew 参数范围**：建议在 0.0-2.0 之间，超过 2.0 可能导致极端的访问集中
4. **并发数调整**：根据链的性能和网络带宽调整 `goroutines` 参数

## 性能优化建议

1. **提高并发数**：增加 `-goroutines` 参数可以提高发送速度
2. **批量测试**：使用脚本批量运行不同参数组合，观察链的性能表现
3. **监控链状态**：配合链的监控工具观察区块生成速度和交易冲突率
4. **调整键空间**：根据实际业务场景选择合适的 `RecordCount`

## 示例脚本

创建批量测试脚本 `run_tests.sh`：

```bash
#!/bin/bash

echo "开始批量测试..."

# Uniform 分布测试
for records in 100 500 1000 5000; do
    echo "=== Uniform 分布, RecordCount=$records ==="
    go run ycsb/*.go -dist uniform -records $records -txcount 10000 -goroutines 50
    sleep 5
done

# Zipfian 分布测试
for skew in 0.5 0.99 1.3 1.8; do
    echo "=== Zipfian 分布, Skew=$skew ==="
    go run ycsb/*.go -dist zipfian -records 1000 -txcount 10000 -goroutines 50 -skew $skew
    sleep 5
done

echo "测试完成！"
```

运行脚本：
```bash
chmod +x run_tests.sh
./run_tests.sh
```

## 参考资料

- [YCSB (Yahoo! Cloud Serving Benchmark)](https://github.com/brianfrankcooper/YCSB)
- [Zipfian Distribution - Wikipedia](https://en.wikipedia.org/wiki/Zipf%27s_law)
- ChainMaker 官方文档
