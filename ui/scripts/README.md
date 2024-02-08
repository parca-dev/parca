# Benchmark Runner

It is a simple benchmark tool that loads the given React component in the browser and measures the time it takes to render the component. It also has support for running dataPopulation scripts that can be used to populate the data from the remote api that the components need to render.

## Usage

```bash
$ yarn benchmark
```

## Options

```bash
yarn benchmark --pattern "ProfileIcicleGraph*"
```

To run only the benchmarks that match the given pattern.

```bash
yarn benchmark --pattern "ProfileIcicleGraph*" --name flamegraphBefore
```

To run only the benchmarks that match the given pattern and save the result with the given name.

```bash
yarn benchmark --pattern "ProfileIcicleGraph*" --compare "flamegraphBefore"
```

To run only the benchmarks that match the given pattern and compare them to the given result name.

```bash
yarn benchmark --apiEndpoint='https://api.example.com'
```

To run the dataPopulation script against a different API endpoint.

```bash
export GRPC_METADATA='{"key":"value","authorization":"Bearer ...."}'
yarn benchmark --apiEndpoint='https://api.example.com'
```

To run the dataPopulation script against a different API endpoint with custom headers.
