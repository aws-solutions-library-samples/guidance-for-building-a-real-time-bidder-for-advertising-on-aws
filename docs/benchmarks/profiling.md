---
title: How to profile the bidder
---

Bidder application exposes usual profiler endpoints as defined by the
[net/http/pprof](https://pkg.go.dev/net/http/pprof) package. The load generator application can access them to capture
CPU profiling data during a benchmark run.

## Collecting profiling data

No special configuration of the bidder is needed: just make sure you have access to the `/debug/pprof/` endpoints.

Running the load generator, set the `LOAD_TEST_PROFILER_OUTPUT` environment variable to the path where the profile
result should be written (alternatively use the `--profiler-output` command line parameter).

Profile data is collected approximately for the duration of the benchmark (as set via `LOAD_TEST_DURATION` or
`--duration`) and includes samples from all goroutines of one bidder process accessible at the benchmark target. (If
benchmarking multiple bidder processes behind a load balancer, only one process is profiled, while all should handle
bid requests.)

You can also set the `LOAD_GENERATOR_START_DELAY` environment variable (or `--start-delay` argument) to make 
the load generator wait for specified amount of time before starting the benchmark. This may help establishing 
connection to the profiler when profiling the bidder under heavy load.

Run a long enough benchmark to collect useful data and choose QPS for a realistic ratio of bid request processing and
asynchronous bidder activity. If the benchmark is too long, the load balancer might cancel the profiler request waiting
during the data collection.

If running load generator in a container, copy the saved profiler output to your computer.

## Analyzing the results

Run e.g.:

    go tool pprof -http localhost:8080 pprof.pb.gz

which starts a service and opens in the browser presenting data from the `pprof.pb.gz` file saved in the previous step.

The most useful features are the (default) graph view and flame graph view available in the View menu.

See <https://blog.golang.org/pprof> and <https://github.com/google/pprof/blob/master/doc/README.md> for other ways of
using `pprof`.
