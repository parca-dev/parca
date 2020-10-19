# conprof - Continuous Profiling

> NOTE: Conprof is already used on production however it's still in active devlopment phase. APIs are prone to change. Otherwise you are welcome to use Conprof and don't forget to give us feedback!

Continuous profiling is the act of taking profiles of programs in a systematic way. Conprof is based on a lot of principles and even code of [Prometheus](https://prometheus.io), the service discovery mechanism and configuration works very similar to Prometheus and the general functionality is similar, as consecutive profiles of the same type and the same process behave similar to time-series, as in that they are related events of the same origin thus they are in the same series. Only that sample values in Conprof are not float64, but an arbitrary byte array.

Currently only collecting [pprof](https://github.com/google/pprof) profiles from HTTP endpoints is supported.

### Why?

Have you ever been in the situation where you know your application has a memory leak or was OOMKilled, but you of course don't have the memory profile from right before that happened? This is why continuous profiling is important, it allows answering these questions even in retrospect.

Conprof is most useful when used together with other systems such as [Prometheus](https://prometheus.io), as Prometheus can be used to identify when something happened and Conprof can be used to investigate the particular incident.

This project is inspired by the [Google-Wide Profiling: A Continuous Profiling Infrastructure for Data Centers
](https://ai.google/research/pubs/pub36575) paper.

### Quickstart

Build conprof binary from the root of the repo directory:

```bash
git clone git@github.com:conprof/conprof.git
GO111MODULE=on GOPROXY=https://proxy.golang.org go install -v
```

Run the example:

```bash
conprof all --config.file examples/conprof.yaml --http-address :8080 --storage.tsdb.path ./data
```

Open `http://localhost:8080/` and write a query like `{job="conprof"}` which after a short amount of time (1 minute should show some data point that can be clicked on). This is conprof profiling itself so the you run it the more data you get.

Here's a screenshot of an instance of conprof running for a couple of minutes, and having run the query `{job="conprof", profile_path="/debug/pprof/heap"}`, plotting samples of heap profiles taken over time.

![conprof screenshot](https://raw.githubusercontent.com/conprof/conprof/master/screenshot.png)

When clicking on a sample the [pprof UI](https://rakyll.org/pprof-ui/) included in the [`pprof`](https://github.com/google/pprof) toolchain will be opened, served by conprof. For example:

![pprof UI screenshot](https://raw.githubusercontent.com/conprof/conprof/master/pprofui.png)

### Building the UI

- Run `npm install` and `npm run build` once in the web directory.
- Run `make assets` to inline the assets into a go file.
- Run `make conprof` to build the app.

Note: For UI development you can run `npm start` in the web directory and it will proxy requests to the backend.
