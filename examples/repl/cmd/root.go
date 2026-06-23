package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/libsql-client-go/examples/repl/repl"
	"github.com/tursodatabase/libsql-client-go/libsql"
)

var (
	dbURL   string
	dbToken string
)

// Execute is the entry point called from main.
func Execute() {
	root := &cobra.Command{
		Use:   "repl",
		Short: "Interactive SQL shell for sqld / Turso",
		Long: `An interactive SQL REPL that connects to a remote sqld (Turso) database.

Environment variables:
  TURSO_DATABASE_URL   (required if --url not set)
  TURSO_AUTH_TOKEN     (required if --token not set)

Examples:
  repl
  repl --url libsql://host.turso.io --token eyJ...
  TURSO_DATABASE_URL=libsql://host.turso.io TURSO_AUTH_TOKEN=eyJ... repl
`,
		RunE: run,
	}

	root.Flags().StringVarP(&dbURL, "url", "u", "", "sqld database URL (or TURSO_DATABASE_URL)")
	root.Flags().StringVarP(&dbToken, "token", "t", "", "auth token (or TURSO_AUTH_TOKEN)")

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Resolve from flag → env var.
	if dbURL == "" {
		dbURL = os.Getenv("TURSO_DATABASE_URL")
	}
	if dbToken == "" {
		dbToken = os.Getenv("TURSO_AUTH_TOKEN")
	}

	if dbURL == "" {
		return fmt.Errorf("database URL is required; set --url or TURSO_DATABASE_URL")
	}

	opts := []libsql.Option{}
	if dbToken != "" {
		opts = append(opts, libsql.WithAuthToken(dbToken))
	}
	connector, err := libsql.NewConnector(dbURL, opts...)
	if err != nil {
		return fmt.Errorf("create connector: %w", err)
	}

	db := sql.OpenDB(connector)
	defer db.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	repl.New(db).Run(ctx)
	return nil
}
