// Package app implements rowback: extracting rows from a (very large) mysqldump
// .sql file by running a pasted SQL query's WHERE clause as a filter, and
// emitting restore SQL — without loading the whole dump into memory.
//
// A one-time table index lets repeat queries seek straight to the referenced
// tables instead of re-reading the whole file.
//
// Run with -serve for the guided web UI, or pass flags for a one-shot CLI run.
package app

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

const defaultQuery = `DELETE FROM orders
WHERE customer_id IN (SELECT id FROM customers WHERE region_id = 42)
  AND order_no IN ('A-1001','A-1002')`

// Run parses flags and executes either the web server or a one-shot extraction.
// It returns a process exit code.
func Run() int {
	var (
		serve    bool
		addr     string
		dumpPath string
		outPath  string
		query    string
		format   string
		progress string
	)
	flag.BoolVar(&serve, "serve", false, "start the guided web UI instead of a one-shot run")
	flag.StringVar(&addr, "addr", "127.0.0.1:8765", "address for the web UI")
	flag.StringVar(&dumpPath, "dump", "dump.sql", "path to the mysqldump .sql file")
	flag.StringVar(&outPath, "out", "restore.sql", "output SQL file")
	flag.StringVar(&query, "query", defaultQuery, "SQL DELETE/SELECT whose WHERE clause selects the rows to extract")
	flag.StringVar(&format, "format", "insert", "output format: insert | raw")
	flag.StringVar(&progress, "progress", "human", "progress output: human | json")
	var insertMode string
	var batchSize int
	flag.StringVar(&insertMode, "insert-mode", "insert", "for -format insert: insert | ignore | replace")
	flag.IntVar(&batchSize, "batch", 0, "rows per INSERT statement (0/1 = one statement per row)")
	var related bool
	flag.BoolVar(&related, "related", false, "also extract child rows that reference the matched rows (transitive, via FK constraints)")
	var indexDir string
	flag.StringVar(&indexDir, "index-dir", "", "folder for the cached table index (default ~/.rowback/indexes)")
	flag.Parse()

	if serve {
		if err := serveUI(addr, dumpPath); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		return 0
	}

	cfg := Config{DumpPath: dumpPath, OutPath: outPath, Query: query, Format: format,
		InsertMode: insertMode, BatchSize: batchSize, Related: related, IndexDir: indexDir}

	enc := json.NewEncoder(os.Stderr)
	cb := func(p Progress) {
		switch progress {
		case "json":
			_ = enc.Encode(p)
		default:
			fmt.Fprintf(os.Stderr, "\r%-10s %6.2f%%  sub=%d main=%d matched=%d   ",
				p.Phase, pct(p.BytesRead, p.TotalBytes), p.SubResolved, p.MainScanned, p.Matched)
		}
	}

	final, err := Extract(cfg, cb, nil)
	if progress != "json" {
		fmt.Fprintln(os.Stderr)
	}
	if err != nil {
		if progress == "json" {
			_ = enc.Encode(Progress{Done: true, Phase: "error", Message: "error: " + err.Error()})
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	fmt.Fprintln(os.Stderr, final.Message)
	if final.Matched == 0 {
		fmt.Fprintln(os.Stderr, "WARNING: no rows matched — check the query, values, and column names")
	}
	return 0
}

func pct(a, b int64) float64 {
	if b <= 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}
