#!/bin/bash

# 快速测试脚本 - 用于验证 YCSB 工具是否正常工作

echo "============================================================"
echo "ChainMaker YCSB 快速测试"
echo "============================================================"
echo ""

echo "测试 1: Uniform 分布（小规模）"
echo "------------------------------------------------------------"
go run ycsb/*.go -dist uniform -records 100 -txcount 100 -goroutines 5

echo ""
echo "等待 2 秒..."
sleep 2
echo ""

echo "测试 2: Zipfian 分布（小规模）"
echo "------------------------------------------------------------"
go run ycsb/*.go -dist zipfian -records 100 -txcount 100 -goroutines 5 -skew 0.99

echo ""
echo "============================================================"
echo "快速测试完成！"
echo "============================================================"
echo ""
echo "如果上述测试成功，可以运行完整测试："
echo "  ./ycsb/run_tests.sh"
echo ""
