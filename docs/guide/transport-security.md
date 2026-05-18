# Transport Security

The API can serve HTTPS directly for small/self-hosted deployments, or run behind a TLS-terminating reverse proxy.

## Direct TLS

Set:

```bash
USE_SSL=true
APP_BASE_DIR=/app
PATH_SSL_CERT=/certs/fullchain.pem
PATH_SSL_KEY=/certs/privkey.pem
```

`cmd/app` enforces TLS 1.2+ when `USE_SSL=true`.

## Reverse Proxy

For production, terminate TLS at a managed load balancer or reverse proxy and forward only private network traffic to `flag-gen`.

Required proxy headers:

- `X-Forwarded-Proto: https`
- `X-Forwarded-For`
- `Host`

Keep `CORS_ALLOWED_ORIGINS` pinned to the dashboard origin. Do not use wildcard origins with credentialed dashboard routes.
