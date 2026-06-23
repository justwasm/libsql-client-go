# gateway — Auto-provisioning namespace proxy for sqld

A reverse proxy that wraps `sqld --enable-namespaces`, lazily creates namespaces on first use, and routes them via a path prefix.

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
client → gateway :9090/foo/v2/pipeline
          1. extracts ns=foo from /foo/v2/pipeline
          2. auto-creates ns via admin API (once)
          3. strips /foo prefix, proxies to sqld with Host: foo.127.0.0.1:8080
          ← sqld responds ←
```

The namespace is encoded as the first path segment.  The libsql Go library
constructs `{url}/v2/pipeline`, so `http://127.0.0.1:9090/my-ns` becomes
`http://127.0.0.1:9090/my-ns/v2/pipeline`.  The gateway splits it back.

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

```go
connector, _ := libsql.NewConnector("http://127.0.0.1:9090/" + ns)
```

Or via the `sql/local` example:

```bash
go run ../sql/local http://127.0.0.1:9090 any-namespace-name
```
