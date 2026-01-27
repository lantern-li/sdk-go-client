package main

import (
	"fmt"
	"path/filepath"
	"testing"
)

// TestBuildDependencyGraph_Complex 测试复杂场景
func TestBuildDependencyGraph_Complex(t *testing.T) {
	transactions := []Transaction{
		// tx0
		{
			ReadKeys:  []string{"key1", "key2"},
			WriteKeys: []string{"key3"},
		},
		// tx1
		{
			ReadKeys:  []string{"key3"},
			WriteKeys: []string{"key1", "key4"},
		},
		// tx2
		{
			ReadKeys:  []string{"key4"},
			WriteKeys: []string{"key2"},
		},
		// tx3
		{
			ReadKeys:  []string{"key3"},
			WriteKeys: []string{"key2"},
		},
	}

	graph := BuildDependencyGraph(transactions)

	// 生成 DOT 文件和图片
	fmt.Println("生成图文件...")

	// 生成 DOT 文件
	dotPath := filepath.Join("./", "tx_dependency_graph.dot")
	if err := graph.GenerateDOT(dotPath); err != nil {
		fmt.Println(err)
	}
	fmt.Printf("✓ DOT 文件已生成: %s\n", dotPath)

	// 尝试渲染图片
	pngPath := filepath.Join("./", "tx_dependency_graph.png")
	if err := graph.RenderGraph(dotPath, pngPath); err != nil {
		fmt.Printf("⚠ 图片渲染失败: %v\n", err)
		fmt.Printf("  提示: 你可以手动使用 graphviz 渲染: dot -Tpng %s -o %s\n", dotPath, pngPath)
	} else {
		fmt.Printf("✓ 图片已生成: %s\n", pngPath)
	}

	fmt.Println("✓ 交易依赖图生成完成")
	fmt.Println("============================================================")

}
