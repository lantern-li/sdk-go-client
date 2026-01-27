package main

import (
	"flag"
	"fmt"
	"log"
)

var (
	inputDot  = flag.String("input", "", "输入的 DOT 文件路径")
	outputDot = flag.String("output", "acyclic_graph.dot", "输出的无环图 DOT 文件路径")
)

func main() {
	flag.Parse()

	if *inputDot == "" {
		log.Fatal("错误: 必须指定 -input 参数（输入的 DOT 文件路径）")
	}

	fmt.Println("====================== 交易依赖图环检测与破环工具 ======================")
	fmt.Printf("输入文件: %s\n", *inputDot)
	fmt.Printf("输出文件: %s\n", *outputDot)
	fmt.Println("========================================================================\n")

	// 1. 解析 DOT 文件
	fmt.Println("步骤 1/4: 解析 DOT 文件...")
	graph, err := ParseDOTFile(*inputDot)
	if err != nil {
		log.Fatalf("解析 DOT 文件失败: %v", err)
	}
	fmt.Printf("✓ 图解析完成: %d 个节点, %d 条边\n\n", len(graph.Nodes), graph.EdgeCount())

	// 2. 检测环
	fmt.Println("步骤 2/4: 检测图中的环...")
	hasCycle := graph.HasCycle()
	if hasCycle {
		fmt.Println("✓ 检测到图中存在环")
	} else {
		fmt.Println("✓ 图中不存在环，已经是 DAG（有向无环图）")
		fmt.Println("\n无需破环，程序结束。")
		return
	}

	// 3. 破环
	fmt.Println("\n步骤 3/4: 移除顶点以破环...")
	removedNodes := graph.BreakCycles()
	fmt.Printf("✓ 已移除 %d 个顶点\n", len(removedNodes))
	fmt.Printf("  移除的顶点: %v\n", removedNodes)

	// 4. 验证并输出
	fmt.Println("\n步骤 4/4: 验证并生成无环图...")
	if graph.HasCycle() {
		log.Fatal("错误: 破环后图中仍然存在环")
	}
	fmt.Printf("✓ 验证通过: 图已变为 DAG\n")
	fmt.Printf("  剩余节点: %d 个\n", len(graph.Nodes))
	fmt.Printf("  剩余边: %d 条\n\n", graph.EdgeCount())

	// 生成 DOT 文件
	if err := graph.GenerateDOT(*outputDot); err != nil {
		log.Fatalf("生成 DOT 文件失败: %v", err)
	}
	fmt.Printf("✓ 无环图已保存到: %s\n", *outputDot)

	fmt.Println("\n========================================================================")
	fmt.Println("✓ 处理完成")
	fmt.Println("========================================================================")
}
