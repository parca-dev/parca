# Instrument your Go app with pprof

Go supports profiling in `pprof` format via the standard library. This guide features two approaches: Global and non-Global.

The [`pprof-example-app-go`](https://github.com/polarsignals/pprof-example-app-go) repository also demonstrates both of these approaches.

## Global

While we do not recommend it, using the global http server uses the least lines of code to get any Go app to expose pprof profiling endpoints.

```go
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()

    // Your application code...
}
```

> A full example with some code that produces some activity can be found [here](https://github.com/polarsignals/pprof-example-app-go/blob/main/global/main.go).

## Non-global

Whereever it makes sense in your program

```go
mux := http.NewServeMux()
mux.HandleFunc("/debug/pprof/", pprof.Index)
mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
go func() { log.Fatal(http.ListenAndServe("localhost:6060", mux)) }()
```

> A full example with some code that produces some activity can be found [here](https://github.com/polarsignals/pprof-example-app-go/blob/main/main.go).
