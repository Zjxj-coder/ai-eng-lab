# Nexus

High-concurrency, low-latency LLM inference gateway.

## Features
- **Gateway**: `text/event-stream` chunked forwarding with backpressure and graceful degradation. Upstream API is mocked with `httptest` simulating streaming blocks and latencies. No real network ports, no vLLM, and no real GPUs are used.
- **Cache**: Prefix hashing (KV cache simulation) and deterministic semantic cache hashing without dependencies. Cross-window cache expiration.
- **Router**: Heuristic-based routing of low-value, repetitive prompts to smaller models.
- **Queue**: Async image generation queue with state machine, idempotency deduplication, and backpressure limit. Single-card serial consumer (no cluster semantics).
- **Imgproc**: C++ (cgo) and pure Go double-implementation with byte-for-byte exact testing. Cgo path gracefully omitted if compiler missing.

## Benchmarking Note
We do NOT use `wrk`. A native Go load generator (`internal/loadgen`) handles 200 concurrency and latency measurements (capturing gateway overhead independently from upstream delay).
We do NOT use ShareGPT. We use a 120,000 synthetic dialogue corpus with a deterministic PRNG seed.
Corpus distribution:
- 18% Exact duplicate (完全重复)
- 14% Prefix shared (前缀共享)
- 12% Semantic similar (语义近似)
- 56% Unique (唯一)

## Running Tests
```bash
go vet ./...
go test ./... -count=1
go test -race ./...
go run ./cmd/bench -json
```

## Gateway Overhead 定义
`gateway_overhead` 的起止点定义如下：
- **起点**：`http.Handler` 入口。
- **计算逻辑**：入口 → 首字节写给客户端之前的累计，**加上**每个分块的“读上游 → 写客户端”中，**不含等待上游** 的那部分真正花在网关代码里的时间。
- **计时包含**：请求解析、路由决策、缓存查找、SSE 分块封装与写出、背压判断。

**为什么中位数（p50）这么低？**
快路径（缓存命中）几乎不做事，慢路径（未命中 + 转发 + 背压）才是 p99 的来源 —— 这个长尾形状本身就是结论。

## 语义缓存权衡曲线 (Tradeoff Curve)
基于 12,000 样本的测试：

| Threshold | Combined Hit Rate | Semantic Hit Rate | False Hit Rate |
|-----------|-------------------|-------------------|----------------|
| 0.70      | 72.1%             | 47.6%             | 38.6%          |
| 0.75      | 72.5%             | 48.0%             | 38.9%          |
| 0.80      | 47.7%             | 23.2%             | 7.2%           |
| 0.82      | 45.8%             | 21.3%             | 3.3%           |
| 0.85      | 44.5%             | 20.0%             | 0.6%           |
| 0.88      | 44.5%             | 17.7%             | 0.6%           |
| 0.90      | 44.4%             | 17.6%             | 0.4%           |
| 0.92      | 44.6%             | 8.6%              | 0.8%           |
| 0.95      | 44.4%             | 2.8%              | 0.5%           |
| 0.98      | 44.5%             | 0.3%              | 0.6%           |

**选定工作点**：`Threshold = 0.85`
**理由**：在 `0.85` 处，`semantic_hit_rate` 为 20.0%（远超 8% 约束要求），而 `false_hit_rate` 降至 0.6%（符合 ≤1% 约束）。这个阈值能在可控的误判率下，最大化语义缓存带来的 Token 成本节省。不需要修改嵌入函数（词袋特征分布差异已足够）。
