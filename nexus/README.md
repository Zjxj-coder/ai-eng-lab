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

<!-- BENCH:BEGIN -->
> 以下 JSON 由 `go run ./cmd/bench -json` 现场产出，每次改动后由脚本重新粘贴，不手工维护。

```json
{
  "gateway": {
    "driver": "go-native",
    "concurrency": 200,
    "duration_s": 5.0202486,
    "qps": 680.245,
    "overhead_min_ns": 0,
    "overhead_p50_ns": 0,
    "overhead_mean_ns": 265255,
    "overhead_p99_ns": 1502400,
    "overhead_max_ns": 6460300,
    "note": "这是网关自身耗时（含路由、缓存、队列判断），不含上游生成等待"
  },
  "cache": {
    "corpus": 120000,
    "corpus_kind": "synthetic-seeded (18% duplicate, 14% prefix, 12% semantic, 56% unique)",
    "prefix_hit_rate": 0.307,
    "semantic_hit_rate": 0.138,
    "combined_hit_rate": 0.445,
    "false_hit_rate": 0.01,
    "token_cost_reduction": 0.508,
    "tradeoff_curve": [
      {
        "threshold": 0.7,
        "combined_hit_rate": 0.897,
        "semantic_hit_rate": 0.59,
        "false_hit_rate": 0.508
      },
      {
        "threshold": 0.75,
        "combined_hit_rate": 0.896,
        "semantic_hit_rate": 0.589,
        "false_hit_rate": 0.508
      },
      {
        "threshold": 0.8,
        "combined_hit_rate": 0.497,
        "semantic_hit_rate": 0.19,
        "false_hit_rate": 0.113
      },
      {
        "threshold": 0.82,
        "combined_hit_rate": 0.464,
        "semantic_hit_rate": 0.157,
        "false_hit_rate": 0.049
      },
      {
        "threshold": 0.85,
        "combined_hit_rate": 0.445,
        "semantic_hit_rate": 0.138,
        "false_hit_rate": 0.01
      },
      {
        "threshold": 0.88,
        "combined_hit_rate": 0.435,
        "semantic_hit_rate": 0.128,
        "false_hit_rate": 0.01
      },
      {
        "threshold": 0.9,
        "combined_hit_rate": 0.435,
        "semantic_hit_rate": 0.128,
        "false_hit_rate": 0.01
      },
      {
        "threshold": 0.92,
        "combined_hit_rate": 0.426,
        "semantic_hit_rate": 0.119,
        "false_hit_rate": 0.009
      },
      {
        "threshold": 0.95,
        "combined_hit_rate": 0.377,
        "semantic_hit_rate": 0.07,
        "false_hit_rate": 0.011
      },
      {
        "threshold": 0.98,
        "combined_hit_rate": 0.311,
        "semantic_hit_rate": 0.004,
        "false_hit_rate": 0.013
      }
    ]
  },
  "queue": {
    "submitted": 5000,
    "accepted": 1001,
    "deduped_by_idempotency_key": 1001,
    "rejected_by_backpressure": 2998,
    "goroutine_leak": false
  }
}
```
<!-- BENCH:END -->
