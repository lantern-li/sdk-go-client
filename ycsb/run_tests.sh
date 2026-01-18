#!/bin/bash

# ChainMaker YCSB 批量性能测试脚本

echo "============================================================"
echo "ChainMaker YCSB 批量性能测试"
echo "============================================================"
echo ""

# 配置参数
TXCOUNT=10000
GOROUTINES=50

# ============================================================
# 实验一：Uniform 分布 - 不同 RecordCount
# ============================================================
echo "开始实验一：Uniform 分布测试（不同键空间大小）"
echo "============================================================"

for records in 50 100 500 1000 5000 10000; do
    echo ""
    echo ">>> 测试 Uniform 分布, RecordCount=$records <<<"
    echo "------------------------------------------------------------"
    go run ycsb/*.go -dist uniform -records $records -txcount $TXCOUNT -goroutines $GOROUTINES

    echo ""
    echo "等待 3 秒后开始下一个测试..."
    sleep 3
done

echo ""
echo "============================================================"
echo "实验一完成！"
echo "============================================================"
echo ""
sleep 5

# ============================================================
# 实验二：Zipfian 分布 - 不同 Skew 参数
# ============================================================
echo "开始实验二：Zipfian 分布测试（不同倾斜度）"
echo "============================================================"

RECORDS=1000  # 固定键空间大小

for skew in 0.0 0.3 0.5 0.99 1.3 1.5 1.8; do
    echo ""
    echo ">>> 测试 Zipfian 分布, RecordCount=$RECORDS, Skew=$skew <<<"
    echo "------------------------------------------------------------"
    go run ycsb/*.go -dist zipfian -records $RECORDS -txcount $TXCOUNT -goroutines $GOROUTINES -skew $skew

    echo ""
    echo "等待 3 秒后开始下一个测试..."
    sleep 3
done

echo ""
echo "============================================================"
echo "实验二完成！"
echo "============================================================"
echo ""
sleep 5

# ============================================================
# 实验三：对比实验 - 相同键空间下的不同分布
# ============================================================
echo "开始实验三：对比实验（相同键空间，不同分布）"
echo "============================================================"

RECORDS=1000  # 固定键空间大小
TXCOUNT_LARGE=50000  # 更大的交易数量以获得更稳定的统计

echo ""
echo ">>> 1. Uniform 分布 (RecordCount=$RECORDS) <<<"
echo "------------------------------------------------------------"
go run ycsb/*.go -dist uniform -records $RECORDS -txcount $TXCOUNT_LARGE -goroutines $GOROUTINES

echo ""
echo "等待 3 秒..."
sleep 3

echo ""
echo ">>> 2. Zipfian 分布 - 轻度倾斜 (Skew=0.5) <<<"
echo "------------------------------------------------------------"
go run ycsb/*.go -dist zipfian -records $RECORDS -txcount $TXCOUNT_LARGE -goroutines $GOROUTINES -skew 0.5

echo ""
echo "等待 3 秒..."
sleep 3

echo ""
echo ">>> 3. Zipfian 分布 - 中等倾斜 (Skew=0.99) <<<"
echo "------------------------------------------------------------"
go run ycsb/*.go -dist zipfian -records $RECORDS -txcount $TXCOUNT_LARGE -goroutines $GOROUTINES -skew 0.99

echo ""
echo "等待 3 秒..."
sleep 3

echo ""
echo ">>> 4. Zipfian 分布 - 高度倾斜 (Skew=1.5) <<<"
echo "------------------------------------------------------------"
go run ycsb/*.go -dist zipfian -records $RECORDS -txcount $TXCOUNT_LARGE -goroutines $GOROUTINES -skew 1.5

echo ""
echo "============================================================"
echo "实验三完成！"
echo "============================================================"
echo ""

# ============================================================
# 完成
# ============================================================
echo ""
echo "============================================================"
echo "所有批量测试完成！"
echo "============================================================"
echo ""
echo "测试总结："
echo "  - 实验一：测试了 6 种不同的键空间大小（Uniform 分布）"
echo "  - 实验二：测试了 7 种不同的倾斜度（Zipfian 分布）"
echo "  - 实验三：对比了 4 种分布模式（相同键空间）"
echo ""
echo "建议查看输出日志，分析不同参数下的 TPS 和成功率。"
echo ""
