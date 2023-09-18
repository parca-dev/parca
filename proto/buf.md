# parca

Parca is a api for the continuous profiling [parca server](https://github.com/parca-dev/parca).

Parca contains multiple service packages for its functionality.

- debuginfo
- profilestore
- query
- scrape
- telemetry

_debuginfo:_ is a service that allows storage of debug info

_profilestore:_ is a service that allows writing pprof profiles to the service

_query:_ is the service that allows you to query profiles from the service

_scrape:_ is the service that allows you to retrieve information about scrape targets

_telemetry_: is the service that receives telemetry data from the Agent, such as unhandled panics
