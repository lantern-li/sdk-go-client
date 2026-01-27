## 使用
```bash
# 破环
go run ./graph_analyze -input "" -output "/Users/denglongli/workspace/sdk-go-demo/graph_analyze/acyclic_graph.dot"
# 生成图片PNG
dot -Tpng ./graph_analyze/acyclic_graph.dot -o ./graph_analyze/acyclic_graph.png
```