package app

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
)

// Config describes one extraction job.
type Config struct {
	DumpPath   string
	OutPath    string
	Query      string
	Format     string // "insert" | "raw"
	InsertMode string // "insert" | "ignore" | "replace" (only for Format=="insert")
	BatchSize  int    // rows per INSERT statement; <=1 = one statement per row
	Related    bool   // also extract child rows that reference the matched rows (transitive)
	IndexDir   string // folder for the cached table index (default ~/.rowback/indexes)
}

// CoverageStat reports, for one explicit IN-list, how many requested values were
// actually found among the matched rows.
type CoverageStat struct {
	Column  string   `json:"Column"`
	Found   int      `json:"Found"`
	Total   int      `json:"Total"`
	Missing []string `json:"Missing"`
}

// Progress is reported periodically and once at the end.
type Progress struct {
	BytesRead   int64          `json:"BytesRead"`
	TotalBytes  int64          `json:"TotalBytes"`
	Phase       string         `json:"Phase"`
	SubResolved int            `json:"SubResolved"`
	MainScanned int            `json:"MainScanned"`
	Matched     int            `json:"Matched"`
	Coverage    []CoverageStat `json:"Coverage"`
	RelatedRows int            `json:"RelatedRows"`
	RelatedTbls int            `json:"RelatedTbls"`
	Done        bool           `json:"Done"`
	Message     string         `json:"Message"`
}

// Extract parses cfg.Query, ensures a table index exists (building it once),
// then reads only the referenced tables to emit matching rows. If collect is
// non-nil it is filled with the structured matched rows for an interactive
// preview (the output file always contains every row).
func Extract(cfg Config, progress func(Progress), collect *Result) (Progress, error) {
	report := func(p Progress) {
		if progress != nil {
			progress(p)
		}
	}
	if cfg.Format == "" {
		cfg.Format = "insert"
	}

	q, err := ParseQuery(cfg.Query)
	if err != nil {
		return Progress{}, err
	}

	idx := loadIndex(cfg.DumpPath, cfg.IndexDir)
	if idx == nil {
		report(Progress{Phase: "indexing", Message: "building one-time table index…"})
		idx, err = buildIndex(cfg.DumpPath, func(off, total int64) {
			report(Progress{Phase: "indexing", BytesRead: off, TotalBytes: total})
		})
		if err != nil {
			return Progress{}, fmt.Errorf("build index: %w", err)
		}
		saveIndex(cfg.DumpPath, cfg.IndexDir, idx)
	}

	// Verify all referenced tables are present.
	var totalBytes int64
	for tbl := range q.referencedTables() {
		rng, ok := idx.Tables[tbl]
		if !ok {
			return Progress{}, fmt.Errorf("table %q not found in dump", tbl)
		}
		totalBytes += rng.End - rng.Start
	}

	// #nosec G304 -- cfg.DumpPath is the dump the local user chose to read.
	f, err := os.Open(cfg.DumpPath)
	if err != nil {
		return Progress{}, err
	}
	defer f.Close()

	// #nosec G304 -- cfg.OutPath is the restore file the local user chose to write.
	out, err := os.Create(cfg.OutPath)
	if err != nil {
		return Progress{}, err
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	defer w.Flush()

	fmt.Fprintf(w, "-- Restore extracted from %s\n-- query: %s\n", cfg.DumpPath, oneLine(cfg.Query))
	if cfg.Format == "insert" {
		fmt.Fprint(w, "SET @OLD_FK=@@FOREIGN_KEY_CHECKS; SET FOREIGN_KEY_CHECKS=0;\n\n")
	}

	var bytesRead int64
	bump := func(n int) { atomic.AddInt64(&bytesRead, int64(n)) }

	// Resolve subqueries first (each reads only its table's region).
	subSets := map[int]map[string]bool{}
	subResolved := 0
	for i := range q.Where {
		c := q.Where[i]
		if c.Sub == nil {
			continue
		}
		report(Progress{Phase: "subquery", BytesRead: atomic.LoadInt64(&bytesRead), TotalBytes: totalBytes})
		set, err := resolveSubquery(f, idx.Tables[c.Sub.Table], c.Sub, func(int64) {})
		if err != nil {
			return Progress{}, err
		}
		// account region bytes for progress
		bump(int(idx.Tables[c.Sub.Table].End - idx.Tables[c.Sub.Table].Start))
		subSets[i] = set
		subResolved += len(set)
	}

	// When following related entities, parse the schema (PKs + FKs) up front.
	var (
		schema     map[string]tableDef
		childIndex map[string][]fkRef
		mainPKPos  = -1
		matchedIDs = map[string]bool{}
	)
	if cfg.Related {
		report(Progress{Phase: "schema", BytesRead: atomic.LoadInt64(&bytesRead), TotalBytes: totalBytes})
		var serr error
		schema, childIndex, serr = parseSchemas(f, idx)
		if serr != nil {
			return Progress{}, fmt.Errorf("parse schema: %w", serr)
		}
		if def, ok := schema[q.Table]; ok {
			mainPKPos = def.PKPos
		}
	}

	// Scan the main table region and emit matches.
	rng := idx.Tables[q.Table]
	if _, err := f.Seek(rng.Start, io.SeekStart); err != nil {
		return Progress{}, err
	}
	r := bufio.NewReaderSize(io.LimitReader(f, rng.End-rng.Start), 1<<20)

	em := newEmitter(w, q.Table, cfg.Format, cfg.InsertMode, cfg.BatchSize)
	var (
		conds      []evalCond
		compileErr error
		mainCols   []string
		mainRows   int
		matched    int
		lastReport int64
	)
	scanErr := processRegion(r, q.Table,
		func(cols []string) {
			mainCols = cols
			conds, compileErr = compileMain(q.Where, colposOf(cols), subSets)
		},
		func(raw string) {
			if compileErr != nil {
				return
			}
			mainRows++
			fields := splitFields(raw)
			if rowMatches(fields, conds) {
				em.add(raw)
				matched++
				if collect != nil {
					collect.add(q.Table, mainCols, raw, fields, "")
				}
				for i := range conds {
					if conds[i].track && conds[i].pos < len(fields) {
						v := unquote(fields[conds[i].pos])
						if conds[i].set[v] {
							conds[i].seen[v] = true
						}
					}
				}
				if cfg.Related && mainPKPos >= 0 && mainPKPos < len(fields) {
					matchedIDs[unquote(fields[mainPKPos])] = true
				}
			}
		},
		func(n int) {
			bump(n)
			if c := atomic.LoadInt64(&bytesRead); c-lastReport >= 8<<20 {
				lastReport = c
				report(Progress{Phase: "scanning", BytesRead: c, TotalBytes: totalBytes,
					SubResolved: subResolved, MainScanned: mainRows, Matched: matched})
			}
		})
	if scanErr != nil {
		return Progress{}, scanErr
	}
	if compileErr != nil {
		return Progress{}, fmt.Errorf("main table %s: %w", q.Table, compileErr)
	}
	em.flush()

	// Coverage: for each explicit IN-list, how many requested values were found.
	var coverage []CoverageStat
	for i := range conds {
		if !conds[i].track {
			continue
		}
		var missing []string
		for _, v := range conds[i].listed {
			if !conds[i].seen[v] {
				missing = append(missing, v)
			}
		}
		coverage = append(coverage, CoverageStat{
			Column: conds[i].name, Found: len(conds[i].seen), Total: len(conds[i].listed), Missing: missing,
		})
	}
	for _, cs := range coverage {
		fmt.Fprintf(w, "-- coverage: %s %d/%d found", cs.Column, cs.Found, cs.Total)
		if len(cs.Missing) > 0 {
			fmt.Fprintf(w, "; missing: %s", strings.Join(cs.Missing, ", "))
		}
		fmt.Fprintln(w)
	}

	// Related entities: walk children (transitive) and emit them parents-first.
	var related relatedResult
	if cfg.Related {
		switch {
		case mainPKPos < 0:
			fmt.Fprintf(w, "\n-- related: skipped — no primary key found for %s\n", q.Table)
		case len(matchedIDs) == 0:
			fmt.Fprint(w, "\n-- related: skipped — no rows matched\n")
		default:
			fmt.Fprintf(w, "\n-- ===== related entities: children of %s (transitive) =====\n", q.Table)
			related = extractRelated(f, idx, schema, childIndex, q.Table, matchedIDs, w, cfg.InsertMode, collect,
				func(table string, scanned, total int64, rows int) {
					// Cap below 100% so the bar only completes on the final Done event.
					br, tb := scanned, total
					if tb <= 0 || br > tb {
						br = tb * 99 / 100
					}
					report(Progress{Phase: "related", BytesRead: br, TotalBytes: tb,
						SubResolved: subResolved, MainScanned: mainRows, Matched: matched,
						RelatedRows: rows, Message: "scanning related: " + table})
				})
			for _, u := range related.Unsupported {
				fmt.Fprintf(w, "-- note: skipped non-PK reference %s\n", u)
			}
		}
	}

	if cfg.Format == "insert" {
		fmt.Fprint(w, "\nSET FOREIGN_KEY_CHECKS=@OLD_FK;\n")
	}
	if err := w.Flush(); err != nil {
		return Progress{}, err
	}

	relMsg := ""
	if cfg.Related {
		relMsg = fmt.Sprintf(" + %d related row(s) across %d table(s)", related.Rows, related.Tables)
	}
	final := Progress{
		BytesRead: totalBytes, TotalBytes: totalBytes, Phase: "done",
		SubResolved: subResolved, MainScanned: mainRows, Matched: matched, Coverage: coverage,
		RelatedRows: related.Rows, RelatedTbls: related.Tables, Done: true,
		Message: fmt.Sprintf("matched %d row(s) from %s%s%s → %s", matched, q.Table, coverageSummary(coverage), relMsg, cfg.OutPath),
	}
	report(final)
	return final, nil
}

// coverageSummary renders a compact " (res_id 3/3)" style suffix.
func coverageSummary(cov []CoverageStat) string {
	if len(cov) == 0 {
		return ""
	}
	parts := make([]string, 0, len(cov))
	for _, c := range cov {
		parts = append(parts, fmt.Sprintf("%s %d/%d", c.Column, c.Found, c.Total))
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// compileMain builds evalConds, wiring resolved subquery sets in by index.
func compileMain(comps []Comparison, colpos map[string]int, subSets map[int]map[string]bool) ([]evalCond, error) {
	out := make([]evalCond, 0, len(comps))
	for i := range comps {
		c := comps[i]
		pos, ok := colpos[c.Col]
		if !ok {
			return nil, fmt.Errorf("column %q not found", c.Col)
		}
		ec := evalCond{pos: pos, op: c.op()}
		switch {
		case c.Sub != nil:
			ec.op = "in"
			ec.set = subSets[i]
		case c.Op == "in":
			ec.set = map[string]bool{}
			for _, v := range c.Values {
				ec.set[v] = true
			}
			// track coverage for explicit IN-lists
			ec.track = true
			ec.name = c.Col
			ec.listed = c.Values
			ec.seen = map[string]bool{}
		default:
			ec.val = c.Values[0]
		}
		out = append(out, ec)
	}
	return out, nil
}

// buildIndex scans the dump once with block reads (no per-line allocation),
// recording each table's byte range from its structure marker to the next.
func buildIndex(dumpPath string, report func(off, total int64)) (*dumpIndex, error) {
	// #nosec G304 -- dumpPath is the file the local user explicitly asked to scan.
	f, err := os.Open(dumpPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	total := fi.Size()

	const block = 4 << 20
	const tail = 256 // enough to hold a full marker + table name
	marker := []byte("\n" + structMarker)

	idx := &dumpIndex{
		DumpSize: total,
		DumpMod:  fi.ModTime().UnixNano(),
		Tables:   map[string]tableRange{},
	}
	type hit struct {
		name string
		off  int64
	}
	var hits []hit

	buf := make([]byte, block+tail)
	var fileOff int64
	var carry int

	// Handle a marker at the very start of file (no preceding newline).
	startMarker := []byte(structMarker)

	for {
		n, rerr := io.ReadFull(f, buf[carry:carry+block])
		valid := carry + n
		if fileOff == 0 && bytes.HasPrefix(buf, startMarker) {
			if name, ok := tableName(buf[len(startMarker):]); ok {
				hits = append(hits, hit{name, 0})
			}
		}
		searchLimit := valid
		if rerr == nil {
			searchLimit = valid - tail // keep tail so names aren't cut off
		}
		base := 0
		for {
			rel := bytes.Index(buf[base:searchLimit], marker)
			if rel < 0 {
				break
			}
			pos := base + rel
			lineStart := fileOff + int64(pos) + 1 // +1: marker begins with '\n'
			if name, ok := tableName(buf[pos+len(marker):]); ok {
				hits = append(hits, hit{name, lineStart})
			}
			base = pos + len(marker)
		}
		if report != nil {
			report(fileOff, total)
		}
		if rerr != nil {
			break
		}
		// carry the tail into the next block to catch boundary-straddling markers
		copy(buf, buf[valid-tail:valid])
		carry = tail
		fileOff += int64(valid - tail)
	}

	for i, h := range hits {
		end := total
		if i+1 < len(hits) {
			end = hits[i+1].off
		}
		idx.Tables[h.name] = tableRange{Start: h.off, End: end}
	}
	if len(idx.Tables) == 0 {
		return nil, fmt.Errorf("no table markers found — is this a mysqldump file?")
	}
	return idx, nil
}

// tableName reads a backtick-terminated identifier from the start of b.
func tableName(b []byte) (string, bool) {
	end := bytes.IndexByte(b, '`')
	if end < 0 {
		return "", false
	}
	return string(b[:end]), true
}

func oneLine(s string) string {
	s = bytes.NewBufferString(s).String()
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			r = ' '
		}
		out = append(out, r)
	}
	return string(out)
}
