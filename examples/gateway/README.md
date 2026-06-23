# gateway — Auto-provisioning namespace proxy for sqld

A reverse proxy that wraps `sqld --enable-namespaces`, lazily creates namespaces on first use, and routes them via the `Host` header subdomain.

## Quick start

```bash
# Start gateway (starts sqld as a subprocess automatically)
go run .

# In another terminal, connect with any namespace name
go run ../sql/local http://127.0.0.1:9090 my-first-ns
```

The namespace `my-first-ns` is created automatically when the first request arrives. No manual admin API calls needed.

## How it works

```
client → gateway :9090 → auto-creates ns via admin API (once)
                        → proxies request to sqld :8080
                        ← sqld responds ←
```

The gateway extracts the namespace from the `Host` header subdomain (e.g. `my-ns.127.0.0.1:9090` → `my-ns`), creates it via `POST /v1/namespaces/:ns/create` if unseen, then proxies the request to sqld.

## Configuration

All via environment variables:

| Variable | Default | Description |
|---|---|---|
| `SQLD_DB_PATH` | `data.sqld` | sqld data directory (persists across restarts) |
| `SQLD_HTTP_LISTEN_ADDR` | `127.0.0.1:8080` | sqld HTTP port |
| `SQLD_ADMIN_LISTEN_ADDR` | `127.0.0.1:8082` | sqld admin API port |
| `LISTEN` | `:9090` | Gateway's public address |

Example with custom data dir:

```bash
SQLD_DB_PATH=/tmp/my-data go run .
```

## Data persistence

sqld data is stored in `SQLD_DB_PATH` (`data.sqld` by default). Restarting the gateway restarts sqld with the same data — all namespaces and their data survive.

## Client usage

Use `WithProxy` to point the Go client at the gateway:

```go
connector, _ := libsql.NewConnector(
    "http://my-ns.127.0.0.1:8080",       // Host header = my-ns.127.0.0.1:8080
    libsql.WithProxy("http://127.0.0.1:9090"), // TCP goes here
)
```

Or use the `sql/local` example:

```bash
go run ../sql/local http://127.0.0.1:9090 any-namespace-name
```
