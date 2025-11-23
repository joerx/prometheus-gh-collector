# Prometheus GHA Collector

Minimal example showing how to export a Prometheus counter from events collected by a GitHub webhook

Build
```
make build
```

Run
```
./out/prometheus-gha-collector --repo example/sample-app
```

This starts an HTTP server on port 9101. Metrics are available at `http://localhost:9101/metrics`.

Testing endpoints:

- `POST /events`: Send an example deployment event payload

```sh
curl -v -XPOST -H "X-GitHub-Event:deployment" -d '{"action":"created", "deployment":{"environment":"deployment","id": 123}}' localhost:9101/events
```

- `GET /metrics`: Get the metrics produced so far

```sh
curl localhost:9101/metrics | grep gha
```
