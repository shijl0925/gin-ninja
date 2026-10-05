# Performance and Stability Report

[Docs Home](./README.md) | [中文](../zh/performance.md)

This page turns the existing hot-path benchmark coverage into a repeatable performance report for production evaluations. Numbers below are a baseline captured from the checked-in benchmarks, not a universal SLA; rerun the commands on your own hardware before capacity planning.

## Scope

The current benchmark suite in `hotpaths_benchmark_test.go` covers:

- gin-ninja route dispatch compared with native Gin.
- Request parameter/body binding overhead compared with native Gin.
- Multi-source binding across path, query, header, cookie, and JSON body.
- Response cache hit overhead compared with an equivalent Gin middleware.
- OpenAPI cold/warm generation for a synthetic large API.
- Response cache miss, conditional ETag, and large-body cache paths.
- Middleware-chain depth across 0, 1, 5, 10, and 20 layers.
- `RunParallel` coverage for routing, binding, and cache hits.
- Redis tag invalidation for 100, 1k, 10k, and 100k cached keys.

## How to Reproduce

Run from the repository root:

```bash
go test -run '^$' -bench '^BenchmarkHotpaths' -benchmem -count=5 .
```

For a narrower comparison:

```bash
go test -run '^$' -bench '^BenchmarkHotpathsRouting$' -benchmem -count=5 .
go test -run '^$' -bench '^BenchmarkHotpathsBinding$' -benchmem -count=5 .
go test -run '^$' -bench '^BenchmarkHotpathsCacheHit$' -benchmem -count=5 .
go test -run '^$' -bench '^BenchmarkHotpathsOpenAPI$' -benchmem -count=5 .
go test -run '^$' -bench '^BenchmarkHotpathsRedisTagInvalidation/keys-1000$' -benchmem -count=5 .
```

The Redis invalidation benchmarks deliberately include large cardinalities. Run the 10k and 100k cases separately when you need those signals because setup time and Redis command volume can dominate a full benchmark sweep.

When comparing changes, keep Go version, CPU model, `GOMAXPROCS`, benchmark command, and repository commit fixed, and compare medians rather than one-off runs.

## Current Baseline

Environment for this sample run:

- Date: 2026-05-29
- Command: `go test -run '^$' -bench '^BenchmarkHotpaths' -benchmem -count=5 .`
- GOOS/GOARCH: `linux/amd64`
- CPU: `AMD EPYC 9V74 80-Core Processor`

| Area | Benchmark | Median ns/op | B/op | allocs/op | Compared with Gin |
| --- | --- | ---: | ---: | ---: | ---: |
| Route dispatch | `BenchmarkHotpathsRouting/gin-ninja` | 4,542 | 6,635 | 27 | +23.3% ns/op |
| Route dispatch | `BenchmarkHotpathsRouting/gin` | 3,683 | 6,186 | 20 | baseline |
| Query + JSON binding | `BenchmarkHotpathsBinding/gin-ninja` | 7,971 | 8,436 | 45 | +5.7% ns/op |
| Query + JSON binding | `BenchmarkHotpathsBinding/gin` | 7,542 | 8,052 | 37 | baseline |
| Multi-source binding | `BenchmarkHotpathsBindingMultiSource` | 10,821 | 10,023 | 65 | no native Gin peer |
| Response cache hit | `BenchmarkHotpathsCacheHit/gin-ninja` | 6,003 | 7,386 | 42 | +7.4% ns/op |
| Response cache hit | `BenchmarkHotpathsCacheHit/gin` | 5,589 | 7,010 | 38 | baseline |

Interpretation:

- The framework overhead is visible but bounded in the current route, binding, and cache-hit microbenchmarks.
- Binding overhead is the smallest Gin-relative delta in the current set because both implementations perform validation and JSON decoding.
- Multi-source binding is intentionally heavier because it combines path, query, header, cookie, default-value handling, and JSON body extraction in a single typed input.
- Cache-hit measurements include response replay behavior and allocations from the benchmark HTTP request/recorder path.

## Stability Signals

The benchmark suite should be treated as a regression guard:

- Route dispatch overhead should not grow without a feature-specific explanation.
- Binding allocations are the most important signal for high-QPS JSON APIs.
- Cache-hit latency should remain close to the native Gin comparison because it runs on read-heavy endpoints.
- Any benchmark that adds locking, reflection, schema generation, or Redis I/O should include `-benchmem` output and a clear workload size.

## Benchmark Coverage Matrix

| Enterprise question | Current status | Recommended acceptance signal |
| --- | --- | --- |
| Overhead compared with native Gin | Covered by route, binding, and cache-hit Gin peers | Track median ns/op, B/op, and allocs/op in PRs that touch routing, binding, cache, or response writing |
| Parameter binding overhead | Covered for query + JSON and multi-source inputs | Add separate path-only, query-only, JSON-only, and form/file benchmarks if optimizing binders |
| OpenAPI generation cache effect | Covered by `BenchmarkHotpathsOpenAPI` cold/warm sub-benchmarks | Compare cold generation vs warm cached `openAPIBytes()` for the synthetic 200-route API |
| Route cache hit/miss performance | Covered by cache-hit peers plus `BenchmarkHotpathsCacheMiss`, `BenchmarkHotpathsCacheETag`, and `BenchmarkHotpathsCacheLargeBody` | Track identical TTL/cache-key behavior for hit/miss and conditional responses |
| Middleware chain overhead | Covered by `BenchmarkHotpathsMiddlewareDepth` | Measure 0, 1, 5, 10, and 20 middleware layers around a no-op endpoint |
| High-concurrency memory allocation | Covered by `BenchmarkHotpathsParallelRouting`, `BenchmarkHotpathsParallelBinding`, and `BenchmarkHotpathsParallelCacheHit` | Report allocs/op and consider pprof heap deltas for regressions |
| Redis cache tag invalidation at large key counts | Covered by `BenchmarkHotpathsRedisTagInvalidation` | Benchmark `InvalidateTags` with 100, 1k, 10k, and 100k keys against Redis/miniredis and record operation count and latency |

## Reporting Template

Use this structure when publishing benchmark results in a release note or PR:

1. Commit SHA, Go version, OS/architecture, CPU, and `GOMAXPROCS`.
2. Exact benchmark command.
3. Raw benchmark output or a link to the CI artifact.
4. Median table for ns/op, B/op, and allocs/op.
5. Gin-relative overhead where a native Gin peer exists.
6. Explanation for any regression above the agreed threshold.
7. Follow-up task for uncovered areas or unstable measurements.

## Next Benchmark Work

Priority order:

1. Capture fresh baseline numbers for the expanded benchmark suite on release hardware.
2. Add path-only, query-only, JSON-only, and form/file binding benchmarks if binder optimizations are planned.
3. Add multiple-tags-per-key Redis invalidation benchmarks if production workloads rely on dense tag fan-out.
