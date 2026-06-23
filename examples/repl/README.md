# repl — Interactive SQL shell for sqld / Turso

A lightweight, extensible SQL REPL that connects to a remote [sqld](https://github.com/tursodatabase/libsql) (Turso) database through the `libsql-client-go` driver.

## Quick start

```bash
# direnv
direnv allow
go run .

# or env vars
TURSO_DATABASE_URL="libsql://host.turso.io" TURSO_AUTH_TOKEN="eyJ..." go run .

# or CLI flags
go run . --url libsql://host.turso.io --token eyJ...
```

## Built-in commands

| Command | Description |
|---|---|
| `.help` | Show available commands |
| `.tables` | List all tables |
| `.quit` / `.exit` | Exit the REPL |

Everything else is treated as SQL and executed against the database.

## Structure

```
.
├── main.go         # Entry point
├── cmd/root.go     # Cobra command, flags, env fallback, connection setup
└── repl/repl.go    # REPL loop, dot commands, result rendering
```

### Adding a dot command

Add a case in `repl.go`'s `handleDot`:

```go
case ".schema":
    r.execSQL(ctx, "SELECT sql FROM sqlite_master WHERE type='table'")
```

### Switching the output format

Swap `printTable` in `repl.go` for another renderer (CSV, JSON, vertical, etc.).
