package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/tursodatabase/libsql-client-go/libsql"
	_ "github.com/ncruces/go-sqlite3/driver"
)

func main() {
	// URL format:
	//   Default ns:  http://127.0.0.1:8080
	//   Named ns:    http://127.0.0.1:9090/<namespace>
	//
	// When a namespace is provided, it's encoded as a path prefix.
	// The library constructs the pipeline URL as {base}/v2/pipeline,
	// so "http://127.0.0.1:9090/foo" becomes "http://127.0.0.1:9090/foo/v2/pipeline".
	sqldURL := "http://127.0.0.1:8080"
	namespace := ""

	if len(os.Args) > 1 {
		sqldURL = os.Args[1]
	}
	if len(os.Args) > 2 {
		namespace = os.Args[2]
	}

	if namespace != "" {
		sqldURL = sqldURL + "/" + namespace
	}

	connector, err := libsql.NewConnector(sqldURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connector: %s\n", err)
		os.Exit(1)
	}
	db := sql.OpenDB(connector)
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ping: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("connected to sqld  ns=%s\n", namespace)

	db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS visits (ts TEXT, page TEXT)")
	db.ExecContext(ctx, "INSERT INTO visits VALUES (datetime('now'), 'home')")
	db.ExecContext(ctx, "INSERT INTO visits VALUES (datetime('now'), 'about')")

	rows, err := db.QueryContext(ctx, "SELECT * FROM visits ORDER BY ts DESC")
	if err != nil {
		fmt.Fprintf(os.Stderr, "query: %s\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	for rows.Next() {
		var ts, page string
		if err := rows.Scan(&ts, &page); err != nil {
			fmt.Fprintf(os.Stderr, "scan: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("  %-24s %s\n", ts, page)
	}
	if err := rows.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "rows: %s\n", err)
		os.Exit(1)
	}
}
