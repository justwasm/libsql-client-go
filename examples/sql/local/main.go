package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"

	"github.com/tursodatabase/libsql-client-go/libsql"
	_ "github.com/ncruces/go-sqlite3/driver"
)

func main() {
	// sqld with --enable-namespaces identifies the namespace via the Host header.
	//
	//   Default ns:    http://127.0.0.1:8080
	//   Named ns:      http://<ns>.127.0.0.1:8080  (via WithProxy)
	//
	// Note: Because Go's HTTP client needs to resolve the hostname, we use
	// WithProxy to keep TCP going to 127.0.0.1:8080 while the Host header
	// carries "<ns>.127.0.0.1:8080" for sqld to route on.
	sqldURL := "http://127.0.0.1:8080"
	namespace := ""

	if len(os.Args) > 1 {
		sqldURL = os.Args[1]
	}
	if len(os.Args) > 2 {
		namespace = os.Args[2]
	}

	var connector driver.Connector
	var err error

	if namespace != "" {
		// Encode the namespace as a subdomain in the URL so the library
		// sends it as the Host header. WithProxy keeps TCP pointing at
		// 127.0.0.1 so DNS resolution works.
		connector, err = libsql.NewConnector(
			fmt.Sprintf("http://%s.127.0.0.1:8080", namespace),
			libsql.WithProxy(sqldURL),
		)
	} else {
		connector, err = libsql.NewConnector(sqldURL)
	}
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
