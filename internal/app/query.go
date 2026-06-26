package app

import (
	"fmt"
	"regexp"
	"strings"
)

// Comparison is a single WHERE predicate in the supported grammar.
//
//	col IN ('a','b')                       -> Op="in", Values set
//	col = 42                               -> Op="=",  Values=[42]
//	col != 'x'                             -> Op="!=", Values=[x]
//	col IN (SELECT sc FROM st WHERE ...)   -> Op="in", Sub set
type Comparison struct {
	Col    string
	Op     string // "in" | "=" | "!="
	Values []string
	Sub    *SubQuery
}

// SubQuery is a single, non-nested IN (SELECT col FROM table WHERE ...).
type SubQuery struct {
	Table string
	Col   string
	Where []Comparison // literal-only predicates (no further subqueries)
}

// Query is the parsed top-level statement.
type Query struct {
	Table string
	Where []Comparison
}

var (
	reMainTable = regexp.MustCompile(`(?is)\b(?:delete\s+from|update|select\b.*?\bfrom)\s+` + "`?" + `([a-z0-9_]+)` + "`?")
	reWhere     = regexp.MustCompile(`(?is)\bwhere\b`)
	reTail      = regexp.MustCompile(`(?is)\b(order\s+by|group\s+by|limit|returning)\b`)
	reSubSelect = regexp.MustCompile(`(?is)^\s*select\s+` + "`?" + `([a-z0-9_]+)` + "`?" + `\s+from\s+` + "`?" + `([a-z0-9_]+)` + "`?" + `(?:\s+where\s+(.*))?$`)
	reCmpIn     = regexp.MustCompile(`(?is)^\s*` + "`?" + `([a-z0-9_]+)` + "`?" + `\s+in\s*\((.*)\)\s*$`)
	reCmpEq     = regexp.MustCompile(`(?is)^\s*` + "`?" + `([a-z0-9_]+)` + "`?" + `\s*(=|!=|<>)\s*(.+?)\s*$`)
)

// ParseQuery parses a DELETE/SELECT/UPDATE statement in the supported shape.
func ParseQuery(sql string) (*Query, error) {
	sql = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sql), ";"))
	if sql == "" {
		return nil, fmt.Errorf("empty query")
	}

	m := reMainTable.FindStringSubmatch(sql)
	if m == nil {
		return nil, fmt.Errorf("could not find a target table (expected DELETE FROM / SELECT ... FROM / UPDATE <table>)")
	}
	q := &Query{Table: m[1]}

	loc := reWhere.FindStringIndex(sql)
	if loc == nil {
		return nil, fmt.Errorf("query has no WHERE clause — refusing to match every row")
	}
	whereStr := sql[loc[1]:]
	if t := reTail.FindStringIndex(whereStr); t != nil {
		whereStr = whereStr[:t[0]]
	}

	conds, err := parseConditions(whereStr, true)
	if err != nil {
		return nil, err
	}
	if len(conds) == 0 {
		return nil, fmt.Errorf("no conditions parsed from WHERE clause")
	}
	q.Where = conds
	return q, nil
}

// parseConditions splits on top-level AND and parses each predicate.
// allowSub controls whether IN (SELECT ...) subqueries are permitted.
func parseConditions(s string, allowSub bool) ([]Comparison, error) {
	parts := splitTopLevelAnd(s)
	var out []Comparison
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		c, err := parseComparison(p, allowSub)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func parseComparison(p string, allowSub bool) (Comparison, error) {
	if m := reCmpIn.FindStringSubmatch(p); m != nil {
		inner := strings.TrimSpace(m[2])
		if sm := reSubSelect.FindStringSubmatch(inner); sm != nil {
			if !allowSub {
				return Comparison{}, fmt.Errorf("nested subquery not supported in: %s", p)
			}
			sub := &SubQuery{Table: sm[2], Col: sm[1]}
			if strings.TrimSpace(sm[3]) != "" {
				w, err := parseConditions(sm[3], false)
				if err != nil {
					return Comparison{}, err
				}
				sub.Where = w
			}
			return Comparison{Col: m[1], Op: "in", Sub: sub}, nil
		}
		return Comparison{Col: m[1], Op: "in", Values: parseValueList(inner)}, nil
	}
	if m := reCmpEq.FindStringSubmatch(p); m != nil {
		op := m[2]
		if op == "<>" {
			op = "!="
		}
		return Comparison{Col: m[1], Op: op, Values: []string{unquote(strings.TrimSpace(m[3]))}}, nil
	}
	return Comparison{}, fmt.Errorf("unsupported condition: %q (supported: IN (...), IN (SELECT ...), =, !=)", p)
}

// splitTopLevelAnd splits on " AND " not enclosed in parentheses or quotes.
func splitTopLevelAnd(s string) []string {
	var parts []string
	depth, inStr := 0, false
	esc := false
	start := 0
	low := strings.ToLower(s)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			if esc {
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '\'' {
				inStr = false
			}
			continue
		}
		switch c {
		case '\'':
			inStr = true
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth == 0 && !inStr && i+5 <= len(s) {
			// match " and " with word boundaries
			if low[i] == ' ' && low[i+1:i+4] == "and" && (i+4 == len(s) || low[i+4] == ' ') {
				parts = append(parts, s[start:i])
				start = i + 4
				i += 3
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// parseValueList splits a comma list respecting single-quoted strings, then
// unquotes each value for comparison against dump field values.
func parseValueList(s string) []string {
	var vals []string
	inStr, esc := false, false
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			if esc {
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '\'' {
				inStr = false
			}
			continue
		}
		if c == '\'' {
			inStr = true
		} else if c == ',' {
			vals = append(vals, unquote(strings.TrimSpace(s[start:i])))
			start = i + 1
		}
	}
	if v := strings.TrimSpace(s[start:]); v != "" {
		vals = append(vals, unquote(v))
	}
	return vals
}

// referencedTables returns every table the query reads (main + subqueries).
func (q *Query) referencedTables() map[string]bool {
	t := map[string]bool{q.Table: true}
	for _, c := range q.Where {
		if c.Sub != nil {
			t[c.Sub.Table] = true
		}
	}
	return t
}
