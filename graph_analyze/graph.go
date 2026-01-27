package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Graph 表示有向图
type Graph struct {
	Nodes   map[int]bool  // 节点集合
	Edges   map[int][]int // 邻接表: from -> []to
	Removed map[int]bool  // 已移除的节点
}

// NewGraph 创建新图
func NewGraph() *Graph {
	return &Graph{
		Nodes:   make(map[int]bool),
		Edges:   make(map[int][]int),
		Removed: make(map[int]bool),
	}
}

//./graph_analyze -input "zipfian/skew0.9record10000/1000笔/tx_dependency_graph.dot" -output "zipfian/skew0.9record10000/1000笔/acyclic_graph.dot"

// AddNode 添加节点
func (g *Graph) AddNode(node int) {
	g.Nodes[node] = true
}

// AddEdge 添加边
func (g *Graph) AddEdge(from, to int) {
	g.Edges[from] = append(g.Edges[from], to)
}

// EdgeCount 计算边的数量
func (g *Graph) EdgeCount() int {
	count := 0
	for from := range g.Edges {
		if g.Removed[from] {
			continue
		}
		for _, to := range g.Edges[from] {
			if !g.Removed[to] {
				count++
			}
		}
	}
	return count
}

// RemoveNode 移除节点（标记为已移除）
func (g *Graph) RemoveNode(node int) {
	g.Removed[node] = true
}

// ParseDOTFile 解析 DOT 文件
func ParseDOTFile(filename string) (*Graph, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	graph := NewGraph()
	scanner := bufio.NewScanner(file)

	// 正则表达式匹配节点和边
	// 节点: tx123 [label="Tx123"];
	nodeRegex := regexp.MustCompile(`tx(\d+)\s*\[`)
	// 边: tx123 -> tx456;
	edgeRegex := regexp.MustCompile(`tx(\d+)\s*->\s*tx(\d+)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 匹配边
		if matches := edgeRegex.FindStringSubmatch(line); matches != nil {
			from, _ := strconv.Atoi(matches[1])
			to, _ := strconv.Atoi(matches[2])
			graph.AddNode(from)
			graph.AddNode(to)
			graph.AddEdge(from, to)
			continue
		}

		// 匹配节点
		if matches := nodeRegex.FindStringSubmatch(line); matches != nil {
			node, _ := strconv.Atoi(matches[1])
			graph.AddNode(node)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	return graph, nil
}

// HasCycle 检测图中是否存在环（使用 DFS）
// 检测当前图（去除 Removed 节点之后）是否存在有向环，只要存在一个环就返回 true。
func (g *Graph) HasCycle() bool {
	// 0: 未访问, 1: 正在访问, 2: 已完成
	visited := make(map[int]int)

	// 对每个未访问的节点进行 DFS
	for node := range g.Nodes {
		if g.Removed[node] {
			continue
		}
		if visited[node] == 0 {
			if g.hasCycleDFS(node, visited) {
				return true
			}
		}
	}
	return false
}

// hasCycleDFS DFS 辅助函数
func (g *Graph) hasCycleDFS(node int, visited map[int]int) bool {
	visited[node] = 1 // 标记为正在访问

	// 遍历所有邻居
	for _, neighbor := range g.Edges[node] {
		if g.Removed[neighbor] {
			continue
		}

		if visited[neighbor] == 1 {
			// 发现回边，存在环
			return true
		}

		if visited[neighbor] == 0 {
			if g.hasCycleDFS(neighbor, visited) {
				return true
			}
		}
	}

	visited[node] = 2 // 标记为已完成
	return false
}

// BreakCycles 破环：移除节点使图变为 DAG
// 只要判定为有环，就删除入度 + 出度 最大的节点
func (g *Graph) BreakCycles() []int {
	removed := []int{}

	// 反复检测和破环，直到没有环
	for g.HasCycle() {
		// 计算每个节点的入度和出度
		inDegree := make(map[int]int)
		outDegree := make(map[int]int)

		for node := range g.Nodes {
			if g.Removed[node] {
				continue
			}
			outDegree[node] = 0
			inDegree[node] = 0
		}

		for from, neighbors := range g.Edges {
			if g.Removed[from] {
				continue
			}
			for _, to := range neighbors {
				if g.Removed[to] {
					continue
				}
				outDegree[from]++
				inDegree[to]++
			}
		}

		// 找到入度+出度最大的节点
		maxDegree := -1
		maxNode := -1
		for node := range g.Nodes {
			if g.Removed[node] {
				continue
			}
			degree := inDegree[node] + outDegree[node]
			if degree > maxDegree {
				maxDegree = degree
				maxNode = node
			}
		}

		// 移除该节点
		if maxNode != -1 {
			g.RemoveNode(maxNode)
			removed = append(removed, maxNode)
		} else {
			break // 没有可移除的节点
		}
	}

	return removed
}

// GenerateDOT 生成 DOT 格式文件
func (g *Graph) GenerateDOT(outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()

	// 写入 DOT 文件头
	fmt.Fprintf(file, "digraph AcyclicTransactionGraph {\n")
	fmt.Fprintf(file, "  rankdir=TB;\n")
	fmt.Fprintf(file, "  node [shape=circle, style=filled, fillcolor=lightgreen];\n")
	fmt.Fprintf(file, "  edge [color=gray];\n\n")

	// 写入所有未移除的节点
	for node := range g.Nodes {
		if !g.Removed[node] {
			fmt.Fprintf(file, "  tx%d [label=\"Tx%d\"];\n", node, node)
		}
	}
	fmt.Fprintf(file, "\n")

	// 写入所有未移除的边
	for from, neighbors := range g.Edges {
		if g.Removed[from] {
			continue
		}
		for _, to := range neighbors {
			if !g.Removed[to] {
				fmt.Fprintf(file, "  tx%d -> tx%d;\n", from, to)
			}
		}
	}

	fmt.Fprintf(file, "}\n")
	return nil
}
