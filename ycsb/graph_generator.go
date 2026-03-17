package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// TxNode 表示交易图中的一个节点
type TxNode struct {
	ID        int      // 交易ID (0-999)
	ReadKeys  []string // 读取的key
	WriteKeys []string // 写入的key
}

// TxGraph 表示交易依赖图
type TxGraph struct {
	Nodes []*TxNode     // 所有交易节点
	Edges map[int][]int // 边: from -> []to (A依赖B，则A->B)
}

// NewTxGraph 创建新的交易图
func NewTxGraph() *TxGraph {
	return &TxGraph{
		Nodes: make([]*TxNode, 0),
		Edges: make(map[int][]int),
	}
}

//使用方法
//# 使用均匀分布生成1000笔交易的依赖图
//./ycsb_test --generate-graph --txcount=1000 --dist=uniform --records=100
//# 使用 zipfian 分布生成1000笔交易的依赖图
//./ycsb_test --generate-graph --txcount=1000 --dist=zipfian --records=100 --skew=0.99

// addEdge 添加边（避免重复边）
func (g *TxGraph) addEdge(from, to int) {
	// 检查是否已存在该边
	for _, existingTo := range g.Edges[from] {
		if existingTo == to {
			return // 边已存在，不重复添加
		}
	}
	g.Edges[from] = append(g.Edges[from], to)
}

// BuildDependencyGraph 构建完整的读写依赖图
// 对于每个交易的读操作，找到所有写入该 key 的交易（无论顺序），建立边
func BuildDependencyGraph(transactions []Transaction) *TxGraph {
	graph := NewTxGraph()

	// 第一步：收集所有 key 的写入者
	// keyWriters[key] = []txID (所有写入该 key 的交易ID列表)
	keyWriters := make(map[string][]int)

	for txID, tx := range transactions {
		// 添加节点
		node := &TxNode{
			ID:        txID,
			ReadKeys:  tx.ReadKeys,
			WriteKeys: tx.WriteKeys,
		}
		graph.Nodes = append(graph.Nodes, node)

		// 记录该交易写入的所有 key
		for _, writeKey := range tx.WriteKeys {
			keyWriters[writeKey] = append(keyWriters[writeKey], txID)
		}
	}

	// 第二步：建立读写依赖边
	// 对于每个交易的每个读 key，找到所有写入该 key 的交易
	for txID, tx := range transactions {
		for _, readKey := range tx.ReadKeys {
			// 找到所有写入该 key 的交易
			if writers, exists := keyWriters[readKey]; exists {
				for _, writerTxID := range writers {
					// 从读者指向写者：txID -> writerTxID
					// 注意：即使 writerTxID == txID（自己读自己写），也建立边
					graph.addEdge(txID, writerTxID)
				}
			}
		}
	}

	return graph
}

// GenerateDOT 生成 DOT 格式的图描述文件
func (g *TxGraph) GenerateDOT(outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建DOT文件失败: %v", err)
	}
	defer file.Close()

	// 写入 DOT 文件头
	fmt.Fprintf(file, "digraph TransactionDependencyGraph {\n")
	fmt.Fprintf(file, "  rankdir=TB;\n")
	fmt.Fprintf(file, "  node [shape=circle, style=filled, fillcolor=lightblue];\n")
	fmt.Fprintf(file, "  edge [color=gray];\n\n")

	// 写入所有节点
	for _, node := range g.Nodes {
		fmt.Fprintf(file, "  tx%d [label=\"Tx%d\"];\n", node.ID, node.ID)
	}
	fmt.Fprintf(file, "\n")

	// 写入所有边
	for from, toList := range g.Edges {
		for _, to := range toList {
			fmt.Fprintf(file, "  tx%d -> tx%d;\n", from, to)
		}
	}

	fmt.Fprintf(file, "}\n")
	return nil
}

// RenderGraph 使用 graphviz 渲染图片
func (g *TxGraph) RenderGraph(dotPath, outputPath string) error {
	// 检查是否安装了 dot 命令
	_, err := exec.LookPath("dot")
	if err != nil {
		return fmt.Errorf("未找到 graphviz (dot 命令)，请先安装: brew install graphviz")
	}

	// 使用 dot 命令渲染图片
	cmd := exec.Command("dot", "-Tpng", dotPath, "-o", outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("渲染图片失败: %v, 输出: %s", err, string(output))
	}

	return nil
}

// GetStatistics 获取图的统计信息
func (g *TxGraph) GetStatistics() map[string]interface{} {
	totalEdges := 0
	for _, toList := range g.Edges {
		totalEdges += len(toList)
	}

	// 计算入度和出度
	inDegree := make(map[int]int)
	outDegree := make(map[int]int)
	for from, toList := range g.Edges {
		outDegree[from] = len(toList)
		for _, to := range toList {
			inDegree[to]++
		}
	}

	// 找出最大入度和出度
	maxInDegree := 0
	maxOutDegree := 0
	for _, degree := range inDegree {
		if degree > maxInDegree {
			maxInDegree = degree
		}
	}
	for _, degree := range outDegree {
		if degree > maxOutDegree {
			maxOutDegree = degree
		}
	}

	return map[string]interface{}{
		"total_nodes":    len(g.Nodes),
		"total_edges":    totalEdges,
		"max_in_degree":  maxInDegree,
		"max_out_degree": maxOutDegree,
	}
}

// GenerateTransactionGraph 生成交易依赖图的主函数
func GenerateTransactionGraph(config TestConfig, outputDir string) error {
	fmt.Println("====================== 生成交易依赖图 ======================")
	fmt.Printf("分布类型: %s\n", config.DistributionType)
	fmt.Printf("键空间大小: %d\n", config.RecordCount)
	fmt.Printf("交易数量: %d\n", config.TotalTxCount)
	if config.DistributionType == "zipfian" {
		fmt.Printf("Zipfian Skew: %.2f\n", config.Skew)
	}
	fmt.Println("============================================================")

	// 1. 生成交易
	fmt.Println("步骤 1/4: 生成交易...")
	transactions := generateTransactions(config)
	fmt.Printf("✓ 已生成 %d 笔交易\n\n", len(transactions))

	// 2. 构建依赖图（新逻辑：考虑所有读写依赖）
	fmt.Println("步骤 2/4: 分析交易依赖关系...")
	graph := BuildDependencyGraph(transactions)
	fmt.Printf("✓ 依赖图构建完成\n\n")

	// 3. 输出统计信息
	fmt.Println("步骤 3/4: 统计图信息...")
	stats := graph.GetStatistics()
	fmt.Printf("  - 总节点数: %d\n", stats["total_nodes"])
	fmt.Printf("  - 总边数: %d\n", stats["total_edges"])
	fmt.Printf("  - 最大入度: %d\n", stats["max_in_degree"])
	fmt.Printf("  - 最大出度: %d\n\n", stats["max_out_degree"])

	// 4. 生成 DOT 文件和图片
	fmt.Println("步骤 4/4: 生成图文件...")

	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 生成 DOT 文件
	dotPath := filepath.Join(outputDir, "tx_dependency_graph.dot")
	if err := graph.GenerateDOT(dotPath); err != nil {
		return err
	}
	fmt.Printf("✓ DOT 文件已生成: %s\n", dotPath)

	// 尝试渲染图片
	pngPath := filepath.Join(outputDir, "tx_dependency_graph.png")
	if err := graph.RenderGraph(dotPath, pngPath); err != nil {
		fmt.Printf("⚠ 图片渲染失败: %v\n", err)
		fmt.Printf("  提示: 你可以手动使用 graphviz 渲染: dot -Tpng %s -o %s\n", dotPath, pngPath)
	} else {
		fmt.Printf("✓ 图片已生成: %s\n", pngPath)
	}

	fmt.Println("\n============================================================")
	fmt.Println("✓ 交易依赖图生成完成")
	fmt.Println("============================================================")

	return nil
}
