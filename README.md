# conprof - Continuous Profiling

> Note this repository is very much a proof of concept, it only works on a very basic level.

Continuous profiling is the act of taking profiles of programs in a systematic way. conprof is based on a lot of principles and even code of Prometheus, the service discovery mechanism and configuration works very similar to Prometheus and the general functionality is similar, as consecutive profiles of the same type and the same process behave similar to time-series, as in that they are related events of the same origin thus they are in the same series.

### Quickstart

Build conprof binary from the root of the repo directory:

```bash
go build
```

Run the example:

```bash
cd examples && mkdir data
../conprof all
```

Open http://localhost:8080/ and keep refreshing to gather the profile data. It's profiling itself so more you run it the more data you get.
