package app

import (
	"bufio"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// tableRange is the byte span of a table's section in the dump, from its
// "-- Table structure for table `X`" marker to the next table's marker (or EOF).
type tableRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// dumpIndex maps table name -> byte range and is persisted next to the dump so
// repeat queries can seek directly to the tables they need.
type dumpIndex struct {
	DumpSize int64                 `json:"dump_size"`
	DumpMod  int64                 `json:"dump_mod"`
	Tables   map[string]tableRange `json:"tables"`
}

const structMarker = "-- Table structure for table `"

// defaultIndexDir is a dedicated, writable folder for cached indexes, so they
// don't clutter (or fail to write next to) the dump itself.
func defaultIndexDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".rowback", "indexes")
	}
	return filepath.Join(os.TempDir(), "rowback-indexes")
}

// indexPath returns the index file path for a dump inside indexDir. The name is
// associated to the dump (basename + a hash of its absolute path), and the
// stored size/mtime guard against reuse for a different file.
func indexPath(dumpPath, indexDir string) string {
	if indexDir == "" {
		indexDir = defaultIndexDir()
	}
	abs, err := filepath.Abs(dumpPath)
	if err != nil {
		abs = dumpPath
	}
	h := sha1.Sum([]byte(abs))
	name := fmt.Sprintf("%s-%x.idx.json", filepath.Base(dumpPath), h[:6])
	return filepath.Join(indexDir, name)
}

// loadIndex returns a valid index for the dump, or nil if missing/stale. It
// checks the associated index folder first, then the legacy sidecar location
// (<dump>.idx.json) so existing indexes are reused without a rescan.
func loadIndex(dumpPath, indexDir string) *dumpIndex {
	fi, err := os.Stat(dumpPath)
	if err != nil {
		return nil
	}
	for _, p := range []string{indexPath(dumpPath, indexDir), dumpPath + ".idx.json"} {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var idx dumpIndex
		if json.Unmarshal(b, &idx) != nil {
			continue
		}
		if idx.DumpSize != fi.Size() || idx.DumpMod != fi.ModTime().UnixNano() {
			continue
		}
		return &idx
	}
	return nil
}

func saveIndex(dumpPath, indexDir string, idx *dumpIndex) {
	p := indexPath(dumpPath, indexDir)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	if b, err := json.Marshal(idx); err == nil {
		_ = os.WriteFile(p, b, 0o644)
	}
}

// evalCond is a compiled predicate against a positional row.
type evalCond struct {
	pos int
	op  string          // "in" | "=" | "!="
	set map[string]bool // for "in"
	val string          // for "=" / "!="

	// coverage tracking for literal IN-lists on the main table
	track  bool
	name   string          // column name (for reporting)
	listed []string        // the requested values, in order
	seen   map[string]bool // requested values actually found in matched rows
}

func compile(comps []Comparison, colpos map[string]int) ([]evalCond, *Comparison, error) {
	out := make([]evalCond, 0, len(comps))
	for i := range comps {
		c := comps[i]
		pos, ok := colpos[c.Col]
		if !ok {
			return nil, &comps[i], fmt.Errorf("column %q not found", c.Col)
		}
		ec := evalCond{pos: pos, op: c.op()}
		switch {
		case c.Sub != nil:
			ec.set = map[string]bool{} // filled after subquery resolution
		case c.Op == "in":
			ec.set = map[string]bool{}
			for _, v := range c.Values {
				ec.set[v] = true
			}
		default:
			ec.val = c.Values[0]
		}
		out = append(out, ec)
	}
	return out, nil, nil
}

func (c Comparison) op() string {
	if c.Op == "" {
		return "in"
	}
	return c.Op
}

func rowMatches(fields []string, conds []evalCond) bool {
	for _, ec := range conds {
		if ec.pos >= len(fields) {
			return false
		}
		v := unquote(fields[ec.pos])
		switch ec.op {
		case "in":
			if !ec.set[v] {
				return false
			}
		case "=":
			if v != ec.val {
				return false
			}
		case "!=":
			if v == ec.val {
				return false
			}
		}
	}
	return true
}

// colpos builds a name->position map from ordered column names.
func colposOf(cols []string) map[string]int {
	m := make(map[string]int, len(cols))
	for i, c := range cols {
		m[c] = i
	}
	return m
}

// processRegion drives the line state machine over a reader that covers exactly
// one table's section. It captures the column order from CREATE TABLE and calls
// onRow for each data tuple.
func processRegion(r *bufio.Reader, table string, onCols func([]string), onRow func(raw string), countBytes func(int)) error {
	inData := false
	insertPrefix := "INSERT INTO `" + table + "` VALUES"
	createPrefix := "CREATE TABLE `" + table + "` ("
	for {
		line, err := r.ReadString('\n')
		if countBytes != nil {
			countBytes(len(line))
		}
		t := strings.TrimRight(line, "\r\n")
		ls := strings.TrimLeft(t, " \t")
		switch {
		case strings.HasPrefix(ls, createPrefix):
			onCols(readCreateTableColumns(r))
		case strings.HasPrefix(ls, insertPrefix):
			inData = true
			forEachTuple(strings.TrimPrefix(ls, insertPrefix), onRow)
		case inData && strings.HasPrefix(ls, "("):
			forEachTuple(t, onRow)
		default:
			inData = false
		}
		if err != nil {
			return nil
		}
	}
}

// resolveSubquery scans a table region and returns the set of sub.Col values
// whose rows satisfy sub.Where.
func resolveSubquery(f *os.File, rng tableRange, sub *SubQuery, report func(int64)) (map[string]bool, error) {
	set := map[string]bool{}
	var cols []string
	var conds []evalCond
	var compileErr error
	if _, err := f.Seek(rng.Start, io.SeekStart); err != nil {
		return nil, err
	}
	lr := io.LimitReader(f, rng.End-rng.Start)
	r := bufio.NewReaderSize(lr, 1<<20)
	read := int64(0)
	err := processRegion(r, sub.Table,
		func(c []string) {
			cols = c
			conds, _, compileErr = compile(sub.Where, colposOf(c))
		},
		func(raw string) {
			if compileErr != nil {
				return
			}
			fields := splitFields(raw)
			if rowMatches(fields, conds) {
				if pos, ok := colposOf(cols)[sub.Col]; ok && pos < len(fields) {
					set[unquote(fields[pos])] = true
				}
			}
		},
		func(n int) {
			read += int64(n)
			if report != nil {
				report(read)
			}
		})
	if err != nil {
		return nil, err
	}
	if compileErr != nil {
		return nil, fmt.Errorf("subquery on %s: %w", sub.Table, compileErr)
	}
	return set, nil
}

// insertVerb maps an insert-mode to its SQL prefix.
func insertVerb(mode string) string {
	switch mode {
	case "ignore":
		return "INSERT IGNORE INTO"
	case "replace":
		return "REPLACE INTO"
	default:
		return "INSERT INTO"
	}
}

// emitter writes matched tuples per the configured format / insert-mode /
// batching. For "raw" it emits bare tuples; for "insert" it emits statements,
// optionally batching several rows into one multi-row VALUES list.
type emitter struct {
	w         *bufio.Writer
	table     string
	format    string // "insert" | "raw"
	verb      string // INSERT INTO | INSERT IGNORE INTO | REPLACE INTO
	batchSize int    // rows per statement; <=1 means one statement per row
	buf       []string
}

func newEmitter(w *bufio.Writer, table, format, insertMode string, batchSize int) *emitter {
	return &emitter{w: w, table: table, format: format, verb: insertVerb(insertMode), batchSize: batchSize}
}

func (e *emitter) add(raw string) {
	if e.format == "raw" {
		fmt.Fprintf(e.w, "(%s)\n", raw)
		return
	}
	if e.batchSize <= 1 {
		fmt.Fprintf(e.w, "%s `%s` VALUES (%s);\n", e.verb, e.table, raw)
		return
	}
	e.buf = append(e.buf, raw)
	if len(e.buf) >= e.batchSize {
		e.flush()
	}
}

func (e *emitter) flush() {
	if len(e.buf) == 0 {
		return
	}
	fmt.Fprintf(e.w, "%s `%s` VALUES\n", e.verb, e.table)
	for i, t := range e.buf {
		sep := ","
		if i == len(e.buf)-1 {
			sep = ";"
		}
		fmt.Fprintf(e.w, "(%s)%s\n", t, sep)
	}
	e.buf = e.buf[:0]
}
