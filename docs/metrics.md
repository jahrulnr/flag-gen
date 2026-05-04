# Metrics & operations (MVP)

The service exposes in-process counters and latency samples at **`GET /v1/system/metrics`** as plain text (`name value` per line).

## Implemented text metrics

| Name | Type | Description |
|------|------|-------------|
| `outbox_claimed_events_total` | counter | Events returned by `ClaimNext` per dispatcher tick |
| `outbox_publish_attempts_total` | counter | Publish attempts |
| `outbox_publish_success_total` | counter | Successful publishes |
| `outbox_publish_failures_total` | counter | Failed publishes (after which event is marked failed + backoff) |
| `outbox_publish_ms_sum` / `outbox_publish_ms_count` | latency | Publish duration (milliseconds) |
| `evaluate_requests_total` | counter | Evaluate API calls (POST + GET) |
| `evaluate_ms_sum` / `evaluate_ms_count` | latency | End-to-end evaluate handler latency |
| `evaluate_circuit_open_total` | counter | Evaluate calls that failed with breaker open |
| `evaluate_service_degraded_total` | counter | Evaluate calls that failed with `service_degraded` |
| `evaluate_service_unavailable_total` | counter | Other 503 evaluate outcomes |
| `scoped_cache_circuit_block_total` | counter | Scoped cache reads blocked while circuit open |
| `scoped_cache_service_degraded_total` | counter | Scoped cache reads that exhausted DB + snapshot and returned degraded |
| `circuit_breaker_open_total` | counter | Times the degraded breaker entered the open window |

## Logs

- Outbox publish failures log `event_id`, aggregate type/id, `correlationId` (from payload when present), and error.
- Request logs should continue to include `X-Correlation-ID` via logmanager middleware.

## Future (Prometheus / OTLP)

Recommended next exports (not yet wired):

- `flaggen_outbox_pending` — gauge from `outbox_events` status counts
- Histogram buckets instead of sum/count pairs
- Push to OTel / Prometheus registry
