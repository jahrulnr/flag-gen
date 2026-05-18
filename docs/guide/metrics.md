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
| `redis_reconnect_events_total` | counter | Redis connection errors that started a background reconnect probe |
| `redis_reconnect_events_coalesced_total` | counter | Redis reconnect events skipped because a probe was already running |
| `redis_reconnect_attempts_total` | counter | Redis reconnect probe attempts |
| `redis_reconnect_success_total` | counter | Redis reconnect probes that observed Redis healthy again |
| `redis_reconnect_failures_total` | counter | Redis reconnect probes that exhausted backoff |
| `snapshot_file_corruption_total` | counter | Snapshot checksum/format/scope validation failures |
| `snapshot_outbox_triggers_total` | counter | Snapshot refresh triggers after a flag outbox event publishes |
| `snapshot_outbox_trigger_failures_total` | counter | Outbox-published snapshot triggers skipped because scope payload was invalid |
| `webhook_enqueued_total` | counter | Outbox events queued for webhook delivery |
| `webhook_queue_dropped_total` | counter | Webhook events dropped because the in-memory queue is full |
| `webhook_delivery_attempts_total` | counter | HTTP webhook delivery attempts |
| `webhook_delivery_success_total` | counter | Successful webhook deliveries |
| `webhook_delivery_failures_total` | counter | Failed webhook delivery attempts |

## Alert signals

Recommended first alerts from the implemented counters:

- DB/cache failover active: page or warn on sustained `scoped_cache_service_degraded_total` or `evaluate_service_degraded_total` increases.
- Cache degradation: warn on `redis_reconnect_failures_total`; page if paired with high evaluate latency or service degradation.
- File corruption: page on any `snapshot_file_corruption_total` increase.
- Broker publish failures: warn on `outbox_publish_failures_total`; page if the rate stays non-zero for multiple dispatcher intervals.
- Webhook delivery failures: warn on sustained `webhook_delivery_failures_total`, especially if paired with `webhook_queue_dropped_total`.

## Logs

- Outbox publish failures log `event_id`, aggregate type/id, `correlationId` (from payload when present), and error.
- Request logs should continue to include `X-Correlation-ID` via logmanager middleware.

## Future (Prometheus / OTLP)

Recommended next exports (not yet wired):

- `flaggen_outbox_pending` — gauge from `outbox_events` status counts
- Histogram buckets instead of sum/count pairs
- Push to OTel / Prometheus registry
