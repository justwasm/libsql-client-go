package repl

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
)

// REPL wraps an interactive SQL shell session.
type REPL struct {
	db      *sql.DB
	input   *bufio.Scanner
	output  io.Writer
	prompt  string
}

// New creates a REPL connected to the given database.
func New(db *sql.DB) *REPL {
	return &REPL{
		db:     db,
		input:  bufio.NewScanner(os.Stdin),
		output: os.Stdout,
		prompt: "sqld> ",
	}
}

// Run starts the read-execute-print loop. It exits on EOF, ".quit", or ".exit".
func (r *REPL) Run(ctx context.Context) {
	fmt.Fprint(r.output, `Connected to sqld.
Type ".help" for help.
`)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		fmt.Fprint(r.output, r.prompt)

		if !r.input.Scan() {
			break // EOF / Ctrl-D
		}
		line := strings.TrimSpace(r.input.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, ".") {
			if r.handleDot(line) {
				return
			}
			continue
		}

		r.execSQL(ctx, line)
	}

	fmt.Fprintln(r.output)
}

func (r *REPL) handleDot(cmd string) (exit bool) {
	switch strings.Fields(cmd)[0] {
	case ".quit", ".exit":
		return true
	case ".help":
		r.printHelp()
	case ".tables":
		r.execSQL(context.Background(), "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	default:
		fmt.Fprintf(r.output, "Unknown command: %s\n", cmd)
	}
	return false
}

func (r *REPL) printHelp() {
	fmt.Fprint(r.output, `.help     show this message
.tables   list tables
.quit     exit the REPL
.exit     exit the REPL
`)
}

func (r *REPL) execSQL(ctx context.Context, sql string) {
	rows, err := r.db.QueryContext(ctx, sql)
	if err != nil {
		fmt.Fprintf(r.output, "Error: %s\n", err)
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		fmt.Fprintf(r.output, "Error: %s\n", err)
		return
	}
	if len(cols) == 0 {
		fmt.Fprintln(r.output, "OK")
		return
	}

	printTable(r.output, cols, rows)
}

// --------------------------------------------------------------------------
// Simple ASCII-table renderer (sqlite3-style).
// --------------------------------------------------------------------------

func printTable(w io.Writer, cols []string, rows *sql.Rows) {
	// Collect all rows first so we can compute column widths.
	type rowData []string
	var data []rowData

	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &nullableScanner{ptr: &vals[i]}
		}
		if err := rows.Scan(ptrs...); err != nil {
			fmt.Fprintf(w, "Error: %s\n", err)
			return
		}
		row := make(rowData, len(cols))
		for i, v := range vals {
			if v == nil {
				row[i] = "NULL"
			} else {
				row[i] = fmt.Sprint(v)
			}
		}
		data = append(data, row)
	}
	if err := rows.Err(); err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}

	if len(data) == 0 {
		fmt.Fprintln(w, "Empty set")
		return
	}

	// Compute widths.
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len(c)
	}
	for _, row := range data {
		for i, v := range row {
			if len(v) > widths[i] {
				widths[i] = len(v)
			}
		}
	}

	// Separator.
	sep := "+"
	for _, w := range widths {
		sep += strings.Repeat("-", w+2) + "+"
	}

	// Header.
	fmt.Fprintln(w, sep)
	line := "|"
	for i, c := range cols {
		line += fmt.Sprintf(" %-*s |", widths[i], c)
	}
	fmt.Fprintln(w, line)
	fmt.Fprintln(w, sep)

	// Rows.
	for _, row := range data {
		line := "|"
		for i, v := range row {
			line += fmt.Sprintf(" %-*s |", widths[i], v)
		}
		fmt.Fprintln(w, line)
	}
	fmt.Fprintln(w, sep)

	fmt.Fprintf(w, "(%d row%s)\n", len(data), plural(len(data)))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// nullableScanner wraps a *any so that sql.Scan writes nil into it for NULLs.
type nullableScanner struct {
	ptr *any
}

func (ns *nullableScanner) Scan(src any) error {
	*ns.ptr = src
	return nil
}
