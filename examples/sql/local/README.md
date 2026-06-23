# local — Connect to sqld with optional namespace

A minimal example that connects to a local or remote sqld instance using the `libsql-client-go` driver.

## Usage

```bash
# Default namespace (sqld without --enable-namespaces)
go run .

# Specific namespace (sqld with --enable-namespaces)
go run . http://127.0.0.1:9090 my-namespace
```

The namespace is encoded as a path prefix.  The library constructs
`{url}/v2/pipeline`, so `http://127.0.0.1:9090/my-namespace` becomes
`http://127.0.0.1:9090/my-namespace/v2/pipeline`.

## Requirements

- A running sqld instance (see `examples/gateway` for one-click setup)
- `ncruces/go-sqlite3` for local SQLite file support (used automatically for `file:` URLs)
