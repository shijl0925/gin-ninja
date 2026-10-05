# 性能与稳定性报告

[文档首页](./README.md) | [English](../en/performance.md)

本文把现有 hot path 基准测试整理成可复现的性能报告，方便企业采用前评估。下面的数字是基于仓库内现有基准测试的一次基线结果，不是通用 SLA；容量规划前应在自己的硬件和部署环境中重新运行。

## 覆盖范围

当前 `hotpaths_benchmark_test.go` 覆盖：

- gin-ninja 路由分发与原生 Gin 对比。
- 请求参数和请求体绑定开销与原生 Gin 对比。
- 同时包含 path、query、header、cookie、JSON body 的多来源绑定。
- Response cache 命中与等价 Gin middleware 的对比。
- 大型合成 API 下 OpenAPI 冷生成/热缓存。
- Response cache miss、条件 `ETag`、大响应体缓存路径。
- 0、1、5、10、20 层中间件链路深度。
- 路由、绑定、cache hit 的 `RunParallel` 并发覆盖。
- 100、1k、10k、100k cached keys 的 Redis tag invalidation。

## 复现方式

在仓库根目录运行：

```bash
go test -run '^$' -bench '^BenchmarkHotpaths' -benchmem -count=5 .
```

也可以只运行单项对比：

```bash
go test -run '^$' -bench '^BenchmarkHotpathsRouting$' -benchmem -count=5 .
go test -run '^$' -bench '^BenchmarkHotpathsBinding$' -benchmem -count=5 .
go test -run '^$' -bench '^BenchmarkHotpathsCacheHit$' -benchmem -count=5 .
go test -run '^$' -bench '^BenchmarkHotpathsOpenAPI$' -benchmem -count=5 .
go test -run '^$' -bench '^BenchmarkHotpathsRedisTagInvalidation/keys-1000$' -benchmem -count=5 .
```

Redis invalidation 基准刻意覆盖大基数场景。需要 10k 和 100k 信号时建议单独运行，因为 setup 时间和 Redis 命令量会显著影响完整 benchmark sweep 的耗时。

比较不同版本时，应固定 Go 版本、CPU 型号、`GOMAXPROCS`、基准测试命令和仓库 commit，并优先比较多次运行的中位数。

## 当前基线

本次样例运行环境：

- 日期：2026-05-29
- 命令：`go test -run '^$' -bench '^BenchmarkHotpaths' -benchmem -count=5 .`
- GOOS/GOARCH：`linux/amd64`
- CPU：`AMD EPYC 9V74 80-Core Processor`

| 场景 | Benchmark | 中位数 ns/op | B/op | allocs/op | 相比 Gin |
| --- | --- | ---: | ---: | ---: | ---: |
| 路由分发 | `BenchmarkHotpathsRouting/gin-ninja` | 4,542 | 6,635 | 27 | +23.3% ns/op |
| 路由分发 | `BenchmarkHotpathsRouting/gin` | 3,683 | 6,186 | 20 | 基线 |
| Query + JSON 绑定 | `BenchmarkHotpathsBinding/gin-ninja` | 7,971 | 8,436 | 45 | +5.7% ns/op |
| Query + JSON 绑定 | `BenchmarkHotpathsBinding/gin` | 7,542 | 8,052 | 37 | 基线 |
| 多来源绑定 | `BenchmarkHotpathsBindingMultiSource` | 10,821 | 10,023 | 65 | 无原生 Gin 对照 |
| Response cache 命中 | `BenchmarkHotpathsCacheHit/gin-ninja` | 6,003 | 7,386 | 42 | +7.4% ns/op |
| Response cache 命中 | `BenchmarkHotpathsCacheHit/gin` | 5,589 | 7,010 | 38 | 基线 |

解读：

- 当前路由、绑定、缓存命中的微基准中，框架开销可见但处于有界范围。
- 绑定场景的 Gin 相对增量最小，因为两边都执行验证和 JSON 解码。
- 多来源绑定会更重，因为它在一个类型化输入中同时处理 path、query、header、cookie、默认值和 JSON body。
- Cache hit 指标包含 response replay 以及 benchmark 中 HTTP request/recorder 路径带来的分配。

## 稳定性信号

基准测试应作为回归保护：

- 路由分发开销不应在没有明确功能原因的情况下增长。
- 对高 QPS JSON API，绑定分配次数是最重要的观察指标之一。
- Cache hit 延迟应接近原生 Gin 对照，因为它面向读多写少接口。
- 新增涉及锁、反射、schema 生成或 Redis I/O 的基准测试时，应始终包含 `-benchmem`，并明确工作负载规模。

## 基准覆盖矩阵

| 企业评估问题 | 当前状态 | 建议验收信号 |
| --- | --- | --- |
| 和原生 Gin 相比的 overhead | 已通过路由、绑定、cache hit Gin 对照覆盖 | 修改 routing、binding、cache、response writing 时跟踪中位数 ns/op、B/op、allocs/op |
| 参数绑定开销 | 已覆盖 query + JSON 和多来源绑定 | 优化 binder 时补充 path-only、query-only、JSON-only、form/file 基准 |
| OpenAPI 生成缓存效果 | 已由 `BenchmarkHotpathsOpenAPI` 冷/热子基准覆盖 | 对合成 200-route API 比较冷生成与热缓存 `openAPIBytes()` |
| route cache hit/miss 性能 | 已覆盖 cache-hit 对照，以及 `BenchmarkHotpathsCacheMiss`、`BenchmarkHotpathsCacheETag`、`BenchmarkHotpathsCacheLargeBody` | 跟踪相同 TTL/cache-key 行为下的 hit/miss 和条件响应 |
| middleware 链路开销 | 已由 `BenchmarkHotpathsMiddlewareDepth` 覆盖 | 测量 0、1、5、10、20 层 middleware 包裹 no-op endpoint 的固定成本 |
| 高并发下内存分配 | 已由 `BenchmarkHotpathsParallelRouting`、`BenchmarkHotpathsParallelBinding`、`BenchmarkHotpathsParallelCacheHit` 覆盖 | 报告 allocs/op，出现回归时考虑补充 pprof heap delta |
| Redis cache tag invalidation 大 key 数量表现 | 已由 `BenchmarkHotpathsRedisTagInvalidation` 覆盖 | 针对 100、1k、10k、100k keys 的 `InvalidateTags` 进行 Redis/miniredis 基准，记录操作次数和延迟 |

## 报告模板

发布 release note 或 PR 性能结果时建议包含：

1. Commit SHA、Go 版本、OS/architecture、CPU、`GOMAXPROCS`。
2. 完整 benchmark 命令。
3. 原始 benchmark 输出或 CI artifact 链接。
4. ns/op、B/op、allocs/op 中位数表格。
5. 存在原生 Gin 对照时，给出 Gin-relative overhead。
6. 超过阈值的回归说明。
7. 对未覆盖或波动较大指标的后续任务。

## 后续基准测试优先级

1. 在 release 硬件上为扩展后的 benchmark suite 采集新基线。
2. 如计划优化 binder，补充 path-only、query-only、JSON-only、form/file 绑定基准。
3. 如生产负载依赖高密度 tag fan-out，补充多 tag/key Redis invalidation 基准。
