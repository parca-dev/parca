# Local tracing setup

This is a minimal local tracing setup with OpenTelemetry and Jaeger based on https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/examples/demo

Once you've started the setup with `docker-compose up -d` you can run Parca with the added flag:
```bash
./bin/parca --otlp-address=127.0.0.1:4317
```

Now check for traces on http://localhost:16686.
