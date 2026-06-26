package app

import (
	"bufio"
	"io"
	"strings"
)

// readLine reads one logical line (until '\n'), returning it without the
// trailing newline. mysqldump escapes real newlines inside string data as "\n",
// so a physical line never splits a value. Lines can be large; ReadString grows
// as needed.
func readLine(r *bufio.Reader) (string, error) {
	s, err := r.ReadString('\n')
	s = strings.TrimRight(s, "\r\n")
	return s, err
}

// readCreateTableColumns consumes the body of a CREATE TABLE statement (the
// CREATE line was already read) and returns the ordered column names. Column
// definition lines start with a backtick; the body ends at a line beginning
// with ')'. Index/constraint lines (KEY, PRIMARY KEY, CONSTRAINT, UNIQUE) do not
// start with a backtick column name, so they are naturally skipped.
func readCreateTableColumns(r *bufio.Reader) []string {
	var cols []string
	for {
		line, err := readLine(r)
		t := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(t, ")") {
			break
		}
		if strings.HasPrefix(t, "`") {
			// `name` <type> ...  -> extract name between first pair of backticks
			if end := strings.IndexByte(t[1:], '`'); end >= 0 {
				cols = append(cols, t[1:1+end])
			}
		}
		if err != nil {
			break
		}
	}
	return cols
}

// forEachTuple scans a segment of a VALUES list and calls fn with the raw inner
// text of each top-level (...) tuple (parentheses stripped). It is quote- and
// escape-aware so parens/commas inside string or JSON values are ignored.
func forEachTuple(segment string, fn func(raw string)) {
	inStr := false
	esc := false
	depth := 0
	start := -1
	for i := 0; i < len(segment); i++ {
		c := segment[i]
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
			if depth == 0 {
				start = i + 1
			}
			depth++
		case ')':
			depth--
			if depth == 0 && start >= 0 {
				fn(segment[start:i])
				start = -1
			}
		}
	}
}

// splitFields splits one tuple's inner text into its top-level comma-separated
// fields, preserving each field's original quoting/escaping verbatim. Commas
// inside strings are ignored.
func splitFields(raw string) []string {
	var fields []string
	inStr := false
	esc := false
	depth := 0
	start := 0
	for i := 0; i < len(raw); i++ {
		c := raw[i]
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
		case ',':
			if depth == 0 {
				fields = append(fields, strings.TrimSpace(raw[start:i]))
				start = i + 1
			}
		}
	}
	fields = append(fields, strings.TrimSpace(raw[start:]))
	return fields
}

// unquote strips surrounding single quotes from a field value and decodes the
// common mysqldump escapes needed for equality comparison. Numeric fields
// (no quotes) are returned as-is.
func unquote(field string) string {
	if len(field) >= 2 && field[0] == '\'' && field[len(field)-1] == '\'' {
		inner := field[1 : len(field)-1]
		if !strings.ContainsRune(inner, '\\') {
			return inner
		}
		var b strings.Builder
		for i := 0; i < len(inner); i++ {
			if inner[i] == '\\' && i+1 < len(inner) {
				i++
				switch inner[i] {
				case 'n':
					b.WriteByte('\n')
				case 't':
					b.WriteByte('\t')
				case 'r':
					b.WriteByte('\r')
				case '0':
					b.WriteByte(0)
				default:
					b.WriteByte(inner[i])
				}
				continue
			}
			b.WriteByte(inner[i])
		}
		return b.String()
	}
	return field
}

var _ = io.EOF
