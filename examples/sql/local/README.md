# local — Connect to sqld with optional namespace

A minimal example that connects to a local or remote sqld instance using the `libsql-client-go` driver.

## Usage

```bash
# Default namespace (sqld without --enable-namespaces)
go run .

# Specific namespace (sqld with --enable-namespaces)
go run . http://127.0.0.1:9090 my-namespace
```

## How namespace routing works

When `--enable-namespaces` is active, sqld identifies the namespace from the `Host` header subdomain.

This example uses `WithProxy` to decouple the TCP destination from the Host header:

```go
connector, _ := libsql.NewConnector(
    fmt.Sprintf("http://%s.127.0.0.1:8080", namespace),  // Host header
    libsql.WithProxy(sqldURL),                           // TCP destination
)
```

Passing just `http://127.0.0.1:8080` with no namespace arg connects without namespace routing.

## Requirements

- A running sqld instance (see `examples/gateway` for one-click setup)
- `ncruces/go-sqlite3` for local SQLite file support (used automatically for `file:` URLs)
