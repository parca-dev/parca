<p align="center">
  <a href="#contributors-" target="_blank">
    <img src="https://img.shields.io/github/all-contributors/parca-dev/parca?style=flat" alt="contributors">
  </a>
  <a href="https://discord.com/invite/ZgUpYgpzXy" target="_blank">
    <img alt="Discord" src="https://img.shields.io/discord/877547706334199818?label=Discord">
  </a>
</p>
<p align="center">
  <img src="ui/packages/shared/icons/src/assets/logo.svg" alt="Parca: Continuous profiling for analysis of CPU, memory usage over time, and down to the line number." height="75">
</p>

<p align="center">Continuous profiling for analysis of CPU, memory usage over time, and down to the line number. Saving infrastructure cost, improving performance, and increasing reliability.</p>

<p align="center"><img src="screenshot.png" alt="Screenshot of Parca"></p>

## Features

- [**eBPF Profiler**](https://www.parca.dev/docs/parca-agent): A single profiler, using eBPF, automatically discovering targets from Kubernetes or systemd across the entire infrastructure with very low overhead. Supports C, C++, Rust, Go, and more!
- **[Open Standards](https://www.parca.dev/docs/concepts/#pprof)**: Both producing pprof formatted profiles with the eBPF based profiler, and ingesting any pprof formatted profiles allowing for wide language adoption and interoperability with existing tooling.

- [**Optimized Storage & Querying**](https://www.parca.dev/docs/storage): Efficiently storing profiling data while retaining raw data and allowing slicing and dicing of data through a label-based search. Aggregate profiling data infrastructure wide, view single profiles in time or compare on any dimension.

## Why?

- **Save Money**: Many organizations have 20-30% of resources wasted with easily optimized code paths. The Parca Agent aims to lower the entry bar by requiring 0 instrumentation for the whole infrastructure. Deploy in your infrastructure and get started!
- **Improve Performance**: Using profiling data collected over time, Parca can with confidence and statistical significance determine hot paths to optimize. Additionally it can show differences between any label dimension, such as deploys, versions, and regions.
- **Understand Incidents**: Profiling data provides unique insight and depth into what a process executed over time. Memory leaks, but also momentary spikes in CPU or I/O causing unexpected behavior, is traditionally difficult to troubleshoot are a breeze with continuous profiling.

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

### Configuration

Flags:

<!-- prettier-ignore-start -->
[embedmd]:# (tmp/help.txt)
```txt
Usage: parca

Flags:
  -h, --help                    Show context-sensitive help.
      --config-path="parca.yaml"
                                Path to config file.
      --mode="all"              Scraper only runs a scraper that sends to a
                                remote gRPC endpoint. All runs all components.
      --http-address=":7070"    Address to bind HTTP server to.
      --port=""                 (DEPRECATED) Use http-address instead.
      --log-level="info"        Log level.
      --log-format="logfmt"     Configure if structured logging as JSON or as
                                logfmt
      --cors-allowed-origins=CORS-ALLOWED-ORIGINS,...
                                Allowed CORS origins.
      --otlp-address=STRING     OpenTelemetry collector address to send traces
                                to.
      --version                 Show application version.
      --path-prefix=""          Path prefix for the UI
      --mutex-profile-fraction=0
                                Fraction of mutex profile samples to collect.
      --block-profile-rate=0    Sample rate for block profile.
      --enable-persistence      Turn on persistent storage for the metastore and
                                profile storage.
      --storage-granule-size=26265625
                                Granule size in bytes for storage.
      --storage-active-memory=536870912
                                Amount of memory to use for active storage.
                                Defaults to 512MB.
      --storage-path="data"     Path to storage directory.
      --storage-enable-wal      Enables write ahead log for profile storage.
      --storage-row-group-size=8192
                                Number of rows in each row group during
                                compaction and persistence. Setting to <= 0
                                results in a single row group per file.
      --symbolizer-demangle-mode="simple"
                                Mode to demangle C++ symbols. Default mode
                                is simplified: no parameters, no templates,
                                no return type
      --symbolizer-number-of-tries=3
                                Number of tries to attempt to symbolize an
                                unsybolized location
      --debuginfo-cache-dir="/tmp"
                                Path to directory where debuginfo is cached.
      --debuginfo-upload-max-size=1000000000
                                Maximum size of debuginfo upload in bytes.
      --debuginfo-upload-max-duration=15m
                                Maximum duration of debuginfo upload.
      --debuginfo-uploads-signed-url
                                Whether to use signed URLs for debuginfo
                                uploads.
      --debuginfod-upstream-servers=https://debuginfod.elfutils.org,...
                                Upstream debuginfod servers. Defaults to
                                https://debuginfod.elfutils.org. It is an
                                ordered list of servers to try. Learn more at
                                https://sourceware.org/elfutils/Debuginfod.html
      --debuginfod-http-request-timeout=5m
                                Timeout duration for HTTP request to upstream
                                debuginfod server. Defaults to 5m
      --metastore="badger"      Which metastore implementation to use
      --profile-share-server="api.pprof.me:443"
                                gRPC address to send share profile requests to.
      --store-address=STRING    gRPC address to send profiles and symbols to.
      --bearer-token=STRING     Bearer token to authenticate with store.
      --bearer-token-file=STRING
                                File to read bearer token from to authenticate
                                with store.
      --insecure                Send gRPC requests via plaintext instead of TLS.
      --insecure-skip-verify    Skip TLS certificate verification.
      --external-label=KEY=VALUE;...
                                Label(s) to attach to all profiles in
                                scraper-only mode.
      --experimental-arrow      EXPERIMENTAL: Enables Arrow ingestion, this will
                                reduce CPU usage but will increase memory usage.
```
<!-- prettier-ignore-end -->

## Credits

Parca was originally developed by [Polar Signals](https://polarsignals.com/). Read the announcement blog post: https://www.polarsignals.com/blog/posts/2021/10/08/introducing-parca-we-got-funded/

## Contributing

Check out our [Contributing Guide](CONTRIBUTING.md) to get started!
It explains how compile Parca, run it with Tilt as container in Kubernetes and send a Pull Request.

## Contributors ✨

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/docs/en/emoji-key)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tbody>
    <tr>
      <td align="center"><a href="https://brancz.com/"><img src="https://avatars.githubusercontent.com/u/4546722?v=4?s=100" width="100px;" alt="Frederic Branczyk"/><br /><sub><b>Frederic Branczyk</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=brancz" title="Code">💻</a> <a href="https://github.com/parca-dev/parca/commits?author=brancz" title="Documentation">📖</a> <a href="#infra-brancz" title="Infrastructure (Hosting, Build-Tools, etc)">🚇</a></td>
      <td align="center"><a href="https://github.com/thorfour"><img src="https://avatars.githubusercontent.com/u/8681572?v=4?s=100" width="100px;" alt="Thor"/><br /><sub><b>Thor</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=thorfour" title="Code">💻</a> <a href="https://github.com/parca-dev/parca/commits?author=thorfour" title="Documentation">📖</a> <a href="#infra-thorfour" title="Infrastructure (Hosting, Build-Tools, etc)">🚇</a></td>
      <td align="center"><a href="https://matthiasloibl.com/"><img src="https://avatars.githubusercontent.com/u/872251?v=4?s=100" width="100px;" alt="Matthias Loibl"/><br /><sub><b>Matthias Loibl</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=metalmatze" title="Code">💻</a> <a href="https://github.com/parca-dev/parca/commits?author=metalmatze" title="Documentation">📖</a> <a href="#infra-metalmatze" title="Infrastructure (Hosting, Build-Tools, etc)">🚇</a></td>
      <td align="center"><a href="https://kakkoyun.me/"><img src="https://avatars.githubusercontent.com/u/536449?v=4?s=100" width="100px;" alt="Kemal Akkoyun"/><br /><sub><b>Kemal Akkoyun</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=kakkoyun" title="Code">💻</a> <a href="https://github.com/parca-dev/parca/commits?author=kakkoyun" title="Documentation">📖</a></td>
      <td align="center"><a href="https://github.com/Sylfrena"><img src="https://avatars.githubusercontent.com/u/35404119?v=4?s=100" width="100px;" alt="Sumera Priyadarsini"/><br /><sub><b>Sumera Priyadarsini</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=Sylfrena" title="Code">💻</a> <a href="https://github.com/parca-dev/parca/commits?author=Sylfrena" title="Documentation">📖</a></td>
    </tr>
    <tr>
      <td align="center"><a href="https://github.com/jessicalins"><img src="https://avatars.githubusercontent.com/u/6627121?v=4?s=100" width="100px;" alt="Jéssica Lins "/><br /><sub><b>Jéssica Lins </b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=jessicalins" title="Documentation">📖</a></td>
      <td align="center"><a href="http://moiji-mobile.com/"><img src="https://avatars.githubusercontent.com/u/390178?v=4?s=100" width="100px;" alt="Holger Freyther"/><br /><sub><b>Holger Freyther</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=zecke" title="Code">💻</a></td>
      <td align="center"><a href="https://github.com/s-urbaniak"><img src="https://avatars.githubusercontent.com/u/375856?v=4?s=100" width="100px;" alt="Sergiusz Urbaniak"/><br /><sub><b>Sergiusz Urbaniak</b></sub></a><br /><a href="#infra-s-urbaniak" title="Infrastructure (Hosting, Build-Tools, etc)">🚇</a></td>
      <td align="center"><a href="https://pawel.krupa.net.pl/"><img src="https://avatars.githubusercontent.com/u/3531758?v=4?s=100" width="100px;" alt="Paweł Krupa"/><br /><sub><b>Paweł Krupa</b></sub></a><br /><a href="#infra-paulfantom" title="Infrastructure (Hosting, Build-Tools, etc)">🚇</a></td>
      <td align="center"><a href="https://yeya24.github.io/"><img src="https://avatars.githubusercontent.com/u/25150124?v=4?s=100" width="100px;" alt="Ben Ye"/><br /><sub><b>Ben Ye</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=yeya24" title="Code">💻</a> <a href="#infra-yeya24" title="Infrastructure (Hosting, Build-Tools, etc)">🚇</a></td>
    </tr>
    <tr>
      <td align="center"><a href="https://github.com/fpuc"><img src="https://avatars.githubusercontent.com/u/1822814?v=4?s=100" width="100px;" alt="Felix"/><br /><sub><b>Felix</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=fpuc" title="Code">💻</a> <a href="https://github.com/parca-dev/parca/commits?author=fpuc" title="Documentation">📖</a> <a href="#infra-fpuc" title="Infrastructure (Hosting, Build-Tools, etc)">🚇</a></td>
      <td align="center"><a href="https://cbrgm.net/"><img src="https://avatars.githubusercontent.com/u/24737434?v=4?s=100" width="100px;" alt="Christian Bargmann"/><br /><sub><b>Christian Bargmann</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=cbrgm" title="Code">💻</a></td>
      <td align="center"><a href="https://github.com/yomete"><img src="https://avatars.githubusercontent.com/u/9016992?v=4?s=100" width="100px;" alt="Yomi Eluwande"/><br /><sub><b>Yomi Eluwande</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=yomete" title="Code">💻</a> <a href="https://github.com/parca-dev/parca/commits?author=yomete" title="Documentation">📖</a></td>
      <td align="center"><a href="https://responsively.app/"><img src="https://avatars.githubusercontent.com/u/1283424?v=4?s=100" width="100px;" alt="Manoj Vivek"/><br /><sub><b>Manoj Vivek</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=manojVivek" title="Code">💻</a> <a href="https://github.com/parca-dev/parca/commits?author=manojVivek" title="Documentation">📖</a></td>
      <td align="center"><a href="http://thepolishamerican.com/"><img src="https://avatars.githubusercontent.com/u/14791956?v=4?s=100" width="100px;" alt="Monica Wojciechowska"/><br /><sub><b>Monica Wojciechowska</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=monicawoj" title="Code">💻</a> <a href="https://github.com/parca-dev/parca/commits?author=monicawoj" title="Documentation">📖</a></td>
    </tr>
    <tr>
      <td align="center"><a href="https://github.com/mrueg"><img src="https://avatars.githubusercontent.com/u/489370?v=4?s=100" width="100px;" alt="Manuel Rüger"/><br /><sub><b>Manuel Rüger</b></sub></a><br /><a href="#infra-mrueg" title="Infrastructure (Hosting, Build-Tools, etc)">🚇</a></td>
      <td align="center"><a href="https://github.com/avinashupadhya99"><img src="https://avatars.githubusercontent.com/u/52544819?v=4?s=100" width="100px;" alt="Avinash Upadhyaya K R"/><br /><sub><b>Avinash Upadhyaya K R</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=avinashupadhya99" title="Code">💻</a></td>
      <td align="center"><a href="https://bandism.net/"><img src="https://avatars.githubusercontent.com/u/22633385?v=4?s=100" width="100px;" alt="Ikko Ashimine"/><br /><sub><b>Ikko Ashimine</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=eltociear" title="Code">💻</a></td>
      <td align="center"><a href="https://maxbru.net"><img src="https://avatars.githubusercontent.com/u/32458727?v=4?s=100" width="100px;" alt="Maxime Brunet"/><br /><sub><b>Maxime Brunet</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=maxbrunet" title="Code">💻</a> <a href="#infra-maxbrunet" title="Infrastructure (Hosting, Build-Tools, etc)">🚇</a></td>
      <td align="center"><a href="https://www.diru.tech/"><img src="https://avatars.githubusercontent.com/u/39561007?v=4?s=100" width="100px;" alt="rohit"/><br /><sub><b>rohit</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=me-diru" title="Code">💻</a></td>
    </tr>
    <tr>
      <td align="center"><a href="http://importhuman.github.io"><img src="https://avatars.githubusercontent.com/u/69148722?v=4?s=100" width="100px;" alt="Ujjwal Goyal"/><br /><sub><b>Ujjwal Goyal</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=importhuman" title="Documentation">📖</a></td>
      <td align="center"><a href="http://hondu.co"><img src="https://avatars.githubusercontent.com/u/959128?v=4?s=100" width="100px;" alt="Javier Honduvilla Coto"/><br /><sub><b>Javier Honduvilla Coto</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=javierhonduco" title="Code">💻</a></td>
      <td align="center"><a href="http://marselester.com"><img src="https://avatars.githubusercontent.com/u/823099?v=4?s=100" width="100px;" alt="Marsel Mavletkulov"/><br /><sub><b>Marsel Mavletkulov</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=marselester" title="Code">💻</a></td>
      <td align="center"><a href="http://bit.ly/2XvWly1"><img src="https://avatars.githubusercontent.com/u/24803604?v=4?s=100" width="100px;" alt="Kautilya Tripathi"/><br /><sub><b>Kautilya Tripathi</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=knrt10" title="Code">💻</a></td>
      <td align="center"><a href="http://jnsgr.uk"><img src="https://avatars.githubusercontent.com/u/668505?v=4?s=100" width="100px;" alt="Jon Seager"/><br /><sub><b>Jon Seager</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=jnsgruk" title="Code">💻</a></td>
    </tr>
    <tr>
      <td align="center"><a href="https://github.com/PhilipGough"><img src="https://avatars.githubusercontent.com/u/5781491?v=4?s=100" width="100px;" alt="Philip Gough"/><br /><sub><b>Philip Gough</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=PhilipGough" title="Code">💻</a></td>
      <td align="center"><a href="http://www.boranseref.com/"><img src="https://avatars.githubusercontent.com/u/20660506?v=4?s=100" width="100px;" alt="Boran Seref"/><br /><sub><b>Boran Seref</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=boranx" title="Code">💻</a></td>
      <td align="center"><a href="https://heylongdacoder.github.io/"><img src="https://avatars.githubusercontent.com/u/79215152?v=4?s=100" width="100px;" alt="Wen Long"/><br /><sub><b>Wen Long</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=heylongdacoder" title="Code">💻</a></td>
      <td align="center"><a href="http://www.dashen.tech"><img src="https://avatars.githubusercontent.com/u/15921519?v=4?s=100" width="100px;" alt="cui fliter"/><br /><sub><b>cui fliter</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=cuishuang" title="Documentation">📖</a></td>
      <td align="center"><a href="http://www.asubiotto.com"><img src="https://avatars.githubusercontent.com/u/10560359?v=4?s=100" width="100px;" alt="Alfonso Subiotto Marqués"/><br /><sub><b>Alfonso Subiotto Marqués</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=asubiotto" title="Code">💻</a></td>
    </tr>
    <tr>
      <td align="center"><a href="https://github.com/TomHellier"><img src="https://avatars.githubusercontent.com/u/4739623?v=4?s=100" width="100px;" alt="TomHellier"/><br /><sub><b>TomHellier</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=TomHellier" title="Code">💻</a></td>
      <td align="center"><a href="https://stefan.vanburen.xyz"><img src="https://avatars.githubusercontent.com/u/622527?v=4?s=100" width="100px;" alt="Stefan VanBuren"/><br /><sub><b>Stefan VanBuren</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=stefanvanburen" title="Code">💻</a></td>
      <td align="center"><a href="https://cpanato.dev"><img src="https://avatars.githubusercontent.com/u/4115580?v=4?s=100" width="100px;" alt="Carlos Tadeu Panato Junior"/><br /><sub><b>Carlos Tadeu Panato Junior</b></sub></a><br /><a href="#infra-cpanato" title="Infrastructure (Hosting, Build-Tools, etc)">🚇</a></td>
      <td align="center"><a href="http://danielqsj.github.io/"><img src="https://avatars.githubusercontent.com/u/7528864?v=4?s=100" width="100px;" alt="Daniel (Shijun) Qian"/><br /><sub><b>Daniel (Shijun) Qian</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=danielqsj" title="Code">💻</a></td>
      <td align="center"><a href="https://github.com/avestuk"><img src="https://avatars.githubusercontent.com/u/28152118?v=4?s=100" width="100px;" alt="Alex Vest"/><br /><sub><b>Alex Vest</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=avestuk" title="Documentation">📖</a></td>
    </tr>
    <tr>
      <td align="center"><a href="https://github.com/ShubhamPalriwala"><img src="https://avatars.githubusercontent.com/u/55556994?v=4?s=100" width="100px;" alt="Shubham Palriwala"/><br /><sub><b>Shubham Palriwala</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=ShubhamPalriwala" title="Code">💻</a></td>
      <td align="center"><a href="https://github.com/fabxc"><img src="https://avatars.githubusercontent.com/u/4948210?v=4?s=100" width="100px;" alt="Fabian Reinartz"/><br /><sub><b>Fabian Reinartz</b></sub></a><br /><a href="https://github.com/parca-dev/parca/commits?author=fabxc" title="Code">💻</a></td>
    </tr>
  </tbody>
</table>

<!-- markdownlint-restore -->
<!-- prettier-ignore-end -->

<!-- ALL-CONTRIBUTORS-LIST:END -->

This project follows the [all-contributors](https://github.com/all-contributors/all-contributors) specification. Contributions of any kind welcome!
