# Performance and Stability Report

[Docs Home](./README.md) | [中文](../zh/performance.md)

This page turns the existing hot-path benchmark coverage into a repeatable performance report for production evaluations. Numbers below are a baseline captured from the checked-in benchmarks, not a universal SLA; rerun the commands on your own hardware before capacity planning.

## Scope

The current benchmark suite in `hotpaths_benchmark_test.go` covers:

- gin-ninja route dispatch compared with native Gin.
- Request parameter/body binding overhead compared with native Gin.
- Multi-source binding across path, query, header, cookie, and JSON body.
- Response cache hit overhead compared with an equivalent Gin middleware.

OpenAPI generation cache, route cache miss behavior, middleware-chain depth, high-concurrency allocation pressure, and Redis tag invalidation at large key counts are explicitly tracked as follow-up benchmark areas below.

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
```

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
| OpenAPI generation cache effect | Not yet covered by a benchmark | Compare cold generation vs warm cached `OpenAPI()`/`/openapi.json` access for a large route set |
| Route cache hit/miss performance | Response cache hit is covered; miss path is not isolated | Add paired cache-hit/cache-miss benchmarks with identical payload size and TTL |
| Middleware chain overhead | Not yet covered by a benchmark | Measure 0, 1, 5, 10, and 20 middleware layers around a no-op endpoint |
| High-concurrency memory allocation | Not yet covered by a benchmark | Add `RunParallel` benchmarks for routing, binding, and cache hits; report allocs/op and pprof heap deltas |
| Redis cache tag invalidation at large key counts | Functional behavior is tested; large-cardinality performance is not benchmarked | Benchmark `InvalidateTags` with 100, 1k, 10k, and 100k keys against Redis/miniredis and record operation count and latency |

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

1. Add OpenAPI cold/warm generation benchmarks with a synthetic large API.
2. Split response cache benchmarks into hit, miss, conditional `ETag`, and large body scenarios.
3. Add middleware depth benchmarks to document the fixed per-layer cost.
4. Add `RunParallel` benchmarks for route dispatch, binding, and cache hits.
5. Add Redis tag invalidation benchmarks for large tag sets and multiple tags per key.
