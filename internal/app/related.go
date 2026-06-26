package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// fkRef is one foreign key: Table.Col references RefTable.RefCol.
type fkRef struct {
	Table, Col, RefTable, RefCol string
}

// tableDef is the parsed schema of one table.
type tableDef struct {
	Cols  []string
	Pos   map[string]int
	PK    string
	PKPos int
}

var (
	reFK = regexp.MustCompile("(?i)FOREIGN KEY\\s*\\(`([a-z0-9_]+)`\\)\\s*REFERENCES\\s*`([a-z0-9_]+)`\\s*\\(`([a-z0-9_]+)`\\)")
	rePK = regexp.MustCompile("(?i)PRIMARY KEY\\s*\\(`([a-z0-9_]+)`")
)

// parseSchemas reads every table's CREATE TABLE (seeking to each region start)
// and returns the schema map plus a child index keyed by referenced table.
func parseSchemas(f *os.File, idx *dumpIndex) (map[string]tableDef, map[string][]fkRef, error) {
	schema := make(map[string]tableDef, len(idx.Tables))
	childIndex := map[string][]fkRef{}

	for name, rng := range idx.Tables {
		if _, err := f.Seek(rng.Start, io.SeekStart); err != nil {
			return nil, nil, err
		}
		r := bufio.NewReaderSize(io.LimitReader(f, rng.End-rng.Start), 1<<16)
		def, fks := parseTableDef(r, name)
		schema[name] = def
		for _, fk := range fks {
			childIndex[fk.RefTable] = append(childIndex[fk.RefTable], fk)
		}
	}
	return schema, childIndex, nil
}

// parseTableDef parses the CREATE TABLE body for columns, primary key, and FKs.
func parseTableDef(r *bufio.Reader, table string) (tableDef, []fkRef) {
	def := tableDef{Pos: map[string]int{}, PKPos: -1}
	var fks []fkRef
	createPrefix := "CREATE TABLE `" + table + "` ("

	// advance to the CREATE TABLE line
	for {
		line, err := r.ReadString('\n')
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), createPrefix) {
			break
		}
		if err != nil {
			return def, fks
		}
	}
	// parse body until the closing ")"
	for {
		line, err := r.ReadString('\n')
		t := strings.TrimLeft(strings.TrimRight(line, "\r\n"), " \t")
		switch {
		case strings.HasPrefix(t, ")"):
			finalizeDef(&def)
			return def, fks
		case strings.HasPrefix(t, "`"):
			if end := strings.IndexByte(t[1:], '`'); end >= 0 {
				def.Cols = append(def.Cols, t[1:1+end])
			}
		case strings.HasPrefix(strings.ToUpper(t), "PRIMARY KEY"):
			if m := rePK.FindStringSubmatch(t); m != nil {
				def.PK = m[1]
			}
		case strings.Contains(strings.ToUpper(t), "FOREIGN KEY"):
			if m := reFK.FindStringSubmatch(t); m != nil {
				fks = append(fks, fkRef{Table: table, Col: m[1], RefTable: m[2], RefCol: m[3]})
			}
		}
		if err != nil {
			finalizeDef(&def)
			return def, fks
		}
	}
}

func finalizeDef(def *tableDef) {
	for i, c := range def.Cols {
		def.Pos[c] = i
	}
	if p, ok := def.Pos[def.PK]; ok {
		def.PKPos = p
	}
}

// relatedResult summarizes a related-entities traversal.
type relatedResult struct {
	Rows        int
	Tables      int
	PerTable    map[string]int
	Unsupported []string
}

// extractRelated walks children (transitively) of the start rows and writes
// their INSERTs parents-first. It scans each involved table AT MOST ONCE:
// tables are processed in topological order of the FK child-graph so that, by
// the time a table is read, every in-scope parent's id set is already known.
// (A fixpoint pass handles rare FK cycles.) startIDs are PK values of the
// already-emitted start-table rows.
func extractRelated(f *os.File, idx *dumpIndex, schema map[string]tableDef, childIndex map[string][]fkRef,
	startTable string, startIDs map[string]bool, w *bufio.Writer, insertMode string, collect *Result,
	report func(table string, scanned, total int64, rows int)) relatedResult {

	res := relatedResult{PerTable: map[string]int{}}
	unsupported := map[string]bool{}

	// reach = transitive set of child tables; total = sum of their region sizes
	// (each scanned once, so this is the true denominator for progress).
	var total int64
	reach := map[string]bool{}
	rstack := []string{startTable}
	for len(rstack) > 0 {
		t := rstack[len(rstack)-1]
		rstack = rstack[:len(rstack)-1]
		for _, fk := range childIndex[t] {
			if !reach[fk.Table] {
				reach[fk.Table] = true
				rstack = append(rstack, fk.Table)
				if rng, ok := idx.Tables[fk.Table]; ok {
					total += rng.End - rng.Start
				}
			}
		}
	}

	inScope := func(t string) bool { return t == startTable || reach[t] }

	// For each reachable table, the FKs that point at an in-scope parent on its
	// primary key (the only supported reference). Non-PK refs are noted/skipped.
	relFKs := map[string][]fkRef{}
	for _, fks := range childIndex {
		for _, fk := range fks {
			if !reach[fk.Table] || !inScope(fk.RefTable) {
				continue
			}
			if pdef, ok := schema[fk.RefTable]; ok && pdef.PK != "" && pdef.PK == fk.RefCol {
				relFKs[fk.Table] = append(relFKs[fk.Table], fk)
			} else {
				unsupported[fk.Table+"."+fk.Col+"→"+fk.RefTable+"."+fk.RefCol] = true
			}
		}
	}

	// Topological sort over reach: edge parent→child for each in-reach parent.
	indeg := map[string]int{}
	adj := map[string][]string{}
	for c := range reach {
		parents := map[string]bool{}
		for _, fk := range relFKs[c] {
			parents[fk.RefTable] = true
		}
		for p := range parents {
			if reach[p] { // edges only among reachable tables; startTable is always ready
				adj[p] = append(adj[p], c)
				indeg[c]++
			}
		}
	}

	pkScope := map[string]map[string]bool{startTable: startIDs}
	emitted := map[string]map[string]bool{}
	printed := map[string]bool{}
	var scanned, lastReport int64

	// scanTable reads one table's region once, emits newly-matched child rows,
	// and grows pkScope[table]. Returns whether it emitted anything new.
	scanTable := func(c string) bool {
		cdef, ok := schema[c]
		if !ok || len(relFKs[c]) == 0 {
			return false
		}
		rng, ok := idx.Tables[c]
		if !ok {
			return false
		}
		// Build (column position, allowed id set) constraints from current scope.
		type constraint struct {
			pos int
			set map[string]bool
		}
		var cons []constraint
		var rel fkRef
		for _, fk := range relFKs[c] {
			pos, ok := cdef.Pos[fk.Col]
			if !ok {
				continue
			}
			if set := pkScope[fk.RefTable]; len(set) > 0 {
				cons = append(cons, constraint{pos, set})
				rel = fk
			}
		}
		if len(cons) == 0 {
			return false
		}
		if emitted[c] == nil {
			emitted[c] = map[string]bool{}
		}
		if pkScope[c] == nil {
			pkScope[c] = map[string]bool{}
		}
		added := false

		if _, err := f.Seek(rng.Start, io.SeekStart); err != nil {
			return false
		}
		if report != nil {
			report(c, scanned, total, res.Rows)
		}
		r := bufio.NewReaderSize(io.LimitReader(f, rng.End-rng.Start), 1<<20)
		_ = processRegion(r, c,
			func([]string) {},
			func(raw string) {
				fields := splitFields(raw)
				match := false
				for _, con := range cons {
					if con.pos < len(fields) && con.set[unquote(fields[con.pos])] {
						match = true
						break
					}
				}
				if !match {
					return
				}
				pk := ""
				if cdef.PKPos >= 0 && cdef.PKPos < len(fields) {
					pk = unquote(fields[cdef.PKPos])
				}
				key := pk
				if key == "" {
					key = raw
				}
				if emitted[c][key] {
					return
				}
				emitted[c][key] = true
				if !printed[c] {
					fmt.Fprintf(w, "\n-- related: %s (via %s.%s → %s.%s)\n", c, c, rel.Col, rel.RefTable, rel.RefCol)
					printed[c] = true
				}
				fmt.Fprintf(w, "%s `%s` VALUES (%s);\n", insertVerb(insertMode), c, raw)
				if collect != nil {
					collect.add(c, cdef.Cols, raw, fields,
						fmt.Sprintf("via %s.%s → %s.%s", c, rel.Col, rel.RefTable, rel.RefCol))
				}
				res.Rows++
				res.PerTable[c]++
				if pk != "" {
					pkScope[c][pk] = true
				}
				added = true
			},
			func(n int) {
				scanned += int64(n)
				if scanned-lastReport >= 8<<20 {
					lastReport = scanned
					if report != nil {
						report(c, scanned, total, res.Rows)
					}
				}
			})
		return added
	}

	// Kahn's algorithm: process a table once all its in-reach parents are done.
	queue := []string{}
	for c := range reach {
		if indeg[c] == 0 {
			queue = append(queue, c)
		}
	}
	processed := map[string]bool{}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if processed[c] {
			continue
		}
		processed[c] = true
		scanTable(c)
		for _, ch := range adj[c] {
			indeg[ch]--
			if indeg[ch] == 0 {
				queue = append(queue, ch)
			}
		}
	}

	// Cycle fallback: any table left unprocessed is part of an FK cycle. Re-scan
	// the leftovers to a fixpoint (dedup keeps it correct; bounded by row count).
	var leftovers []string
	for c := range reach {
		if !processed[c] {
			leftovers = append(leftovers, c)
		}
	}
	for changed := len(leftovers) > 0; changed; {
		changed = false
		for _, c := range leftovers {
			if scanTable(c) {
				changed = true
			}
		}
	}

	res.Tables = len(res.PerTable)
	for k := range unsupported {
		res.Unsupported = append(res.Unsupported, k)
	}
	return res
}
