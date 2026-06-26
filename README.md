# rowback

[![CI](https://github.com/albertovincenzi/rowback/actions/workflows/ci.yml/badge.svg)](https://github.com/albertovincenzi/rowback/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Surgically restore deleted rows from a massive MySQL/MariaDB dump.**

You ran a `DELETE` you regret, and all you have is last night's multi‑gigabyte
`mysqldump`. `rowback` reverses that `DELETE` as a *filter*: paste the query, and
it extracts exactly the rows it would have removed — plus their foreign‑key‑linked
children — and emits replayable restore SQL. No staging database, no full restore.

- **Streams the dump** — never loads it into memory, so a 30 GB file is fine.
- **One‑time table index** — the first run records each table's byte range
  (cached in `~/.rowback/indexes`); every later query seeks straight to the
  tables it needs instead of re‑reading the whole file.
- **Follows foreign keys** — optionally pulls every child row that references the
  matched rows (transitively), emitted parents‑first so the restore loads cleanly.
- **Two UIs** — a guided web UI and a native desktop app, plus a scriptable CLI.
- **Preview & curate** — inspect the matched rows in a table grid, filter them,
  remove rows, bulk‑ or single‑cell‑edit a column, then export the curated set.

---

## How it works

A `mysqldump` stores each table's data in one contiguous block, in alphabetical
order, with no row‑level index. `rowback`:

1. **Indexes** the dump once — a fast marker scan that records every table's
   `[start,end)` byte range, saved next to nothing you care about
   (`~/.rowback/indexes/<dump>-<hash>.idx.json`).
2. **Parses your query** (a `DELETE`/`SELECT`/`UPDATE`). The `WHERE` clause is the
   filter. Supported predicates: `col IN (…)` literal lists, `col = v`, `col != v`,
   and one `col IN (SELECT c FROM t WHERE …)` sub‑query, combined with `AND`.
3. **Resolves sub‑queries** by reading only the referenced table's region.
4. **Scans the target table's region** and emits matching rows, with a quote/
   escape/paren‑aware tuple parser that keeps JSON columns intact.
5. **(Optional) follows foreign keys** read from the dump's `CREATE TABLE`
   constraints, scanning each involved child table **once** (topological order),
   emitting their rows parents‑first.

## Install / build

Requires Go 1.22+.

```bash
go build -o rowback ./cmd/rowback
```

## Usage

### Web UI (recommended)

```bash
./rowback -serve
# → http://127.0.0.1:8765
```

Pick the dump, paste a query, choose the output format, optionally enable
**related entities**, run it, then curate the results in the preview grid and
**Export kept rows**.

### CLI

```bash
./rowback \
  -dump dump.sql \
  -query "DELETE FROM orders
          WHERE customer_id IN (SELECT id FROM customers WHERE region_id = 42)
            AND order_no IN ('A-1001','A-1002')" \
  -related \
  -format insert \
  -insert-mode insert \
  -out restore.sql
```

| Flag | Default | Description |
|------|---------|-------------|
| `-dump` | `dump.sql` | Path to the mysqldump `.sql` file |
| `-query` | *(example)* | `DELETE`/`SELECT`/`UPDATE` whose `WHERE` selects rows |
| `-out` | `restore_reservations.sql` | Output SQL file |
| `-format` | `insert` | `insert` (replayable) or `raw` (bare tuples) |
| `-insert-mode` | `insert` | `insert` · `ignore` · `replace` |
| `-batch` | `0` | Rows per `INSERT` statement (0/1 = one per row) |
| `-related` | `false` | Also extract child rows via FK constraints (transitive) |
| `-index-dir` | `~/.rowback/indexes` | Where the cached index lives |
| `-serve` | `false` | Start the web UI instead of a one‑shot run |
| `-progress` | `human` | `human` or `json` (one Progress object per line) |

### Desktop app (macOS)

```bash
cd desktop && ./build-app.sh
open "Rowback.app"
```

The desktop app wraps the same web UI. By default it builds a pure‑Go launcher
that opens the UI in your browser; with a working C++ toolchain it builds a true
native WebView window (`go build -tags webview`). See [desktop/README.md](desktop/README.md).

## Output

- **`insert`** → `INSERT INTO \`t\` VALUES (…);` (or `INSERT IGNORE` / `REPLACE`),
  wrapped in `SET FOREIGN_KEY_CHECKS=0/…` so it loads regardless of FK order.
- **`raw`** → bare `(…)` tuples, one per line, for inspection or piping.

Restore by replaying the file:

```bash
mysql -h <host> -u <user> -p <db> < restore.sql
```

## Coverage report

For each `IN (…)` list, `rowback` reports how many of the requested values were
actually found — e.g. `order_no 2/2`, or `1/2 — missing: A-1002` — so you know
whether the dump really contained everything you asked for.

## Safety notes

- `rowback` only **reads** the dump and **writes** an SQL file; it never touches a
  live database.
- A `WHERE` clause is required — it refuses to match every row.
- Related extraction follows the dump's declared FK constraints; references to a
  non‑primary‑key column are noted and skipped.

## License

[MIT](LICENSE) © Alberto Vincenzi
