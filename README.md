# conprof - Continuous Profiling

> NOTE: Conprof is already used in production, however, it's still in active development. APIs are prone to change. Otherwise, you are welcome to use Conprof, and don't forget to give us feedback!

Continuous profiling is the act of taking profiles of programs in a systematic way. Conprof collects, stores and makes profiles available to be queried over time. This is especially useful in post-mortem analysis, when it is too late to take measurements.

### Why?

Have you ever been in the situation where you know your application has a memory leak and was OOMKilled, but you of course don't have the memory profile from right before that happened? Or you experienced a latency spike in your application and you wish you had a CPU profile from exactly that point in time, but don't have it? This is why continuous profiling is important, it allows answering these questions even in retrospect, without going on a search for the needle in the haystack after the fact.

Conprof is most useful when paired with a metrics system such as [Prometheus](https://prometheus.io), as Prometheus can be used to identify a situation based on metrics and Conprof can be used to investigate the particular incident.

### How can I use it?

The way Conprof works, is by collecting profiles in [pprof](https://github.com/google/pprof) format from HTTP endpoints. So all applications have to do is use one of the client libraries for pprof and expose an HTTP endpoint serving it. Pprof client libraries exist for various languages:

* [Go](https://golang.org/pkg/net/http/pprof/) (part of the standard lib, it supports profiling memory allocations, heap memory, CPU, goroutine blocking and mutex contention)
* [Rust](https://github.com/tikv/pprof-rs) (supports CPU profiling)
* [Python](https://pypi.org/project/pypprof/) (supports CPU, heap memory and thread profiling)
* [NodeJS](https://github.com/google/pprof-nodejs) (supports CPU wall time profiling and heap memory profiling)
* [JVM](https://github.com/papertrail/profiler) (supports CPU profiling)

Any HTTP endpoint, that exposes a valid pprof profile is supported, so even custom profiles can be created and collected and viewed by Conprof, such as [`fgprof`](https://github.com/felixge/fgprof).

Additionally any [`perf`](https://perf.wiki.kernel.org/index.php/Main_Page) profile can be converted to pprof using [`perf_data_converter`](https://github.com/google/perf_data_converter), so even programs that do not have native support for pprof can benefit from continuous profiling with Conprof. We do, however, recommend to use native instrumentation when possible, as it allows language and runtime specific nuances to be encodede in the respective libraries.

Once there is an HTTP endpoint that serves profiles in pprof format, all that needs to be done is configure Conprof to collect the profile in a regular interval. See [`examples/conprof.yaml`](examples/conprof.yaml) for an example configuration.

### Inspiration

If this sounds a lot like [Prometheus](https://prometheus.io/), then that's no accident. The creator author of Conprof (@brancz) is also a Prometheus maintainer. Conprof is based on a lot of principles and even code of [Prometheus](https://prometheus.io), the service discovery mechanism and configuration works very similar to Prometheus and the general functionality is similar, as consecutive profiles of the same type and the same process behave similar to time-series, as in that they are related events of the same origin thus they are in the same series. Only that sample values in Conprof are not float64, but an arbitrary byte array.

Additionally, Google has written about continuous profiling in their whitepaper: [Google-Wide Profiling: A Continuous Profiling Infrastructure for Data Centers](https://ai.google/research/pubs/pub36575)".

### Quickstart

Pre-built container images (for linux-amd64, linux-arm64, linux-armv7) can be found at: https://quay.io/repository/conprof/conprof .

Run the pre-built docker container (here we're using the `master-2021-02-15-56c07ca` tag, but choose any version you want to use from the above container image repo):

```bash
docker run --network host --rm -it -v /etc/passwd:/etc/passwd -u `id -u`:`id -g` -v `pwd`:`pwd`:z -w `pwd` quay.io/conprof/conprof:master-2021-02-15-56c07ca all --config.file examples/conprof.yaml --http-address :10902 --storage.tsdb.path ./data
```

You can also build the conprof binary yourself from the root of the repo directory:

```bash
git clone git@github.com:conprof/conprof.git
GO111MODULE=on GOPROXY=https://proxy.golang.org go install -v
```

Run the example with the binary you built:

```bash
conprof all --config.file examples/conprof.yaml --http-address :10902 --storage.tsdb.path ./data
```

Whether you use the simple process or docker, open `http://localhost:10902/` and write a query like `{job="conprof"}` which after a short amount of time (1 minute should show some data point that can be clicked on). This is conprof profiling itself so the you run it the more data you get.

Here's a screenshot of an instance of conprof running for a couple of minutes, and having run the query `{job="conprof", profile_path="/debug/pprof/heap"}`, plotting samples of heap profiles taken over time.

![conprof screenshot](https://raw.githubusercontent.com/conprof/conprof/master/screenshot.png)

When clicking on a sample the [pprof UI](https://rakyll.org/pprof-ui/) included in the [`pprof`](https://github.com/google/pprof) toolchain, that we know and love, will be opened, served by conprof. For example:

![pprof UI screenshot](https://raw.githubusercontent.com/conprof/conprof/master/pprofui.png)

### Building the UI

- Run `npm install` and `npm run build` once in the web directory.
- Run `make assets` to inline the assets into a go file.
- Run `make conprof` to build the app.

Note: For UI development you can run `npm start` in the web directory and it will proxy requests to the backend.
