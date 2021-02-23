# Instrument Your Application

The way Conprof works, is by collecting profiles in [pprof](https://github.com/google/pprof) format from HTTP endpoints. So all applications have to do is use one of the client libraries for pprof and expose an HTTP endpoint serving it. Pprof client libraries exist for various languages:

| Language/runtime | CPU  | Heap | Allocations | Blocking | Mutex Contention | Extra |
|---|---|---|---|---|---|---|
| [Go](https://golang.org/pkg/net/http/pprof/) | Yes | Yes | Yes | Yes | Yes | Goroutine, [`fgprof`](https://github.com/felixge/fgprof) |
| [Rust](https://github.com/tikv/pprof-rs) | Yes | No | No | No | No |  |
| [Python](https://pypi.org/project/pypprof/) | Yes | Yes  | No | No | No |  |
| [NodeJS](https://github.com/google/pprof-nodejs) | Yes | Yes | No | No | No |  |
| [JVM](https://github.com/papertrail/profiler) | Yes | No | No | No | No |  |

## Guides

* [Go: Instrument your Go app with pprof](./instrumenting/go.md)

## Generic Profiling

Additionally any [`perf`](https://perf.wiki.kernel.org/index.php/Main_Page) profile can be converted to pprof using [`perf_data_converter`](https://github.com/google/perf_data_converter), so even programs that do not have native support for pprof can benefit from continuous profiling with Conprof. We do, however, recommend to use native instrumentation when possible, as it allows language and runtime specific nuances to be encodede in the respective libraries.

Once there is an HTTP endpoint that serves profiles in pprof format, all that needs to be done is configure Conprof to collect the profile in a regular interval. See [`examples/conprof.yaml`](examples/conprof.yaml) for an example configuration.
