<p align="center">
  <img src="ui/packages/shared/icons/src/assets/logo.svg" alt="Parca: Continuous profiling for analysis of CPU, memory usage over time, and down to the line number." height="75">
</p>


<p align="center">Continuous profiling for analysis of CPU, memory usage over time, and down to the line number. Saving infrastructure cost, improving performance, and increasing reliability.</p>



<p align="center"><img src="screenshot.png" alt="Screenshot of Parca"></p>

## Features

* [**eBPF Profiler**](https://www.parca.dev/docs/parca-agent/): A single profiler, using eBPF, automatically discovering targets from Kubernetes or systemd across the entire infrastructure with very low overhead. Supports C, C++, Rust, Go, and more!
* **[Open Standards](https://www.parca.dev/docs/concepts/#pprof)**: Both producing pprof formatted profiles with the eBPF based profiler, and ingesting any pprof formatted profiles allowing for wide language adoption and interoperability with existing tooling.

* [**Optimized Storage & Querying**](https://www.parca.dev/docs/storage/): Efficiently storing profiling data while retaining raw data and allowing slicing and dicing of data through a label-based search. Aggregate profiling data infrastructure wide, view single profiles in time or compare on any dimension.

## Why?
* **Save Money**: Many organizations have 20-30% of resources wasted with easily optimized code paths. The Parca Agent aims to lower the entry bar by requiring 0 instrumentation for the whole infrastructure. Deploy in your infrastructure and get started!
* **Improve Performance**: Using profiling data collected over time, Parca can with confidence and statistical significance determine hot paths to optimize. Additionally it can show differences between any label dimension, such as deploys, versions, and regions.
* **Understand Incidents**: Profiling data provides unique insight and depth into what a process executed over time. Memory leaks, but also momentary spikes in CPU or I/O causing unexpected behavior, is traditionally difficult to troubleshoot are a breeze with continuous profiling.

## Feedback & Support

If you have any feedback, please open a discussion in the GitHub Discussions of this project.  
We would love to learn what you think!

## Installation & Documentation

Check Parca's website for updated and in-depth installation guides and documentation!

[parca.dev](https://www.parca.dev/)

## Development

You need to have [Go](https://golang.org/), [Node](https://nodejs.org/en/download/) and [Yarn](https://classic.yarnpkg.com/en/) installed.

Clone the project

```bash
git clone https://github.com/parca-dev/parca.git
```

Go to the project directory

```bash
cd parca
```

Build the UI and compile the Go binaries

```bash
make build
```

### Running the compiled Parca binary

The binary was compiled to `bin/parca` .

```
./bin/parca
```

Now Parca is running locally and its web UI is available on http://localhost:7070/.

By default Parca is scraping it's own pprof endpoints and you should see profiles show up over time. 
The scrape configuration can be changed in the `parca.yaml` in the root of the repository. 

