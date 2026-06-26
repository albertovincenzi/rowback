package app

// maxPreviewRows caps how many rows per table are retained for the interactive
// preview. The output FILE always contains every matched row; this cap only
// bounds the in-memory/JSON payload sent to the UI.
const maxPreviewRows = 20000

// PreviewRow is one row: the verbatim dump tuple (for faithful re-export) plus
// per-column display cells.
type PreviewRow struct {
	Raw   string   `json:"raw"`
	Cells []string `json:"cells"`
}

// TablePreview is the collected rows of one table (main or related).
type TablePreview struct {
	Name     string       `json:"name"`
	Columns  []string     `json:"columns"`
	Rows     []PreviewRow `json:"rows"`
	Total    int          `json:"total"`    // total matched (may exceed len(Rows) if capped)
	Relation string       `json:"relation"` // "" for the main table; FK path for related
	Capped   bool         `json:"capped"`
}

// Result is the structured preview of an extraction.
type Result struct {
	Table  string          `json:"table"`
	Tables []*TablePreview `json:"tables"`
	byName map[string]*TablePreview
}

func newResult() *Result { return &Result{byName: map[string]*TablePreview{}} }

// add appends a row to the named table's preview, creating it if needed.
func (r *Result) add(table string, cols []string, raw string, fields []string, relation string) {
	tp := r.byName[table]
	if tp == nil {
		tp = &TablePreview{Name: table, Columns: cols, Relation: relation}
		r.byName[table] = tp
		r.Tables = append(r.Tables, tp)
	}
	tp.Total++
	if len(tp.Rows) >= maxPreviewRows {
		tp.Capped = true
		return
	}
	tp.Rows = append(tp.Rows, PreviewRow{Raw: raw, Cells: cellsOf(fields)})
}

// cellsOf turns raw field tokens into display values (quotes/escapes decoded).
func cellsOf(fields []string) []string {
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = unquote(f)
	}
	return out
}
