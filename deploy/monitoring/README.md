# Monitoring

LINKTOR exposes Prometheus metrics at `GET /metrics` (unauthenticated, like
`/health` — restrict it at the ingress/network layer if the router is public).

## Files

- `prometheus.yml` — minimal scrape config that targets the app's `/metrics` and
  loads the alert rules. Adjust the target host/port/TLS for your deployment.
- `alerts.yml` — alerting rules for availability, queues/DLQ, pipeline failures
  and HTTP. Thresholds are homologation starting points — tune to your traffic.

## Key metrics (namespace `linktor_`)

| Metric | Meaning |
| --- | --- |
| `linktor_nats_up` | 1 when the NATS/JetStream connection is live, else 0 |
| `linktor_inbound_messages_total{channel_type,result}` | Inbound processed / duplicate / failed |
| `linktor_outbound_messages_total{channel_type,result}` | Outbound sent / failed |
| `linktor_nats_publish_failures_total{kind}` | NATS publish failures by kind |
| `linktor_stream_messages{stream}` | Messages held per stream (`LINKTOR_DLQ` = dead-letter backlog) |
| `linktor_consumer_pending{stream,consumer}` | Delivery backlog (lag) |
| `linktor_consumer_ack_pending{stream,consumer}` | In-flight awaiting ack |
| `linktor_consumer_redelivered{stream,consumer}` | Redeliveries (poison-message signal) |
| `linktor_http_requests_total{method,route,status}` | HTTP request counts |
| `linktor_http_request_seconds_bucket` | HTTP latency histogram |
| `linktor_inbound_processing_seconds` / `linktor_outbound_send_seconds` | Pipeline latency histograms |

Plus the standard Go runtime / process collectors (`go_*`, `process_*`).

## Validate the rules

```sh
promtool check rules deploy/monitoring/alerts.yml
```

## Quick local run

Point a Prometheus at a running app (the compose network name `linktor` and port
`8081` are the defaults):

```sh
prometheus --config.file=deploy/monitoring/prometheus.yml
```
