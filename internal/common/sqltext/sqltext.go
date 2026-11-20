// Package sqltext reads SQL far enough to say what a statement is and which tables it names.
// It is not a parser and it never executes anything: it exists so a read route cannot be talked
// into a write, and so a later rehearsal can name its scope from the same reading.
package sqltext

import (
	"errors"
	"fmt"
	"strings"
)

// Kind is what a statement does, at the granularity this build cares about.
type Kind string

// Statement kinds. Everything a read route refuses lands in one of the write kinds or Other.
const (
	Select  Kind = "select"
	Explain Kind = "explain"
	Show    Kind = "show"
	Insert  Kind = "insert"
	Update  Kind = "update"
	Delete  Kind = "delete"
	DDL     Kind = "ddl"
	Other   Kind = "other"
)

// Errors a caller maps onto a 400.
var (
	ErrEmpty    = errors.New("sqltext: the statement is empty")
	ErrMultiple = errors.New("sqltext: one statement per request")
)

// ddlKeywords begin a statement that changes a schema.
var ddlKeywords = map[string]bool{
	"ALTER": true, "CREATE": true, "DROP": true, "RENAME": true, "TRUNCATE": true,
	"ANALYZE": true, "OPTIMIZE": true, "REPAIR": true, "GRANT": true, "REVOKE": true,
	"COMMENT": true,
}

// Statement is the classification plus whatever could be read out of it.
type Statement struct {
	Kind Kind
	Raw  string
	// Tables are the qualified names the statement reads or writes, best effort, in the case
	// the statement wrote them in.
	Tables []string
	// Locking marks a read that takes locks or writes a file: `SELECT ... FOR UPDATE`,
	// `LOCK IN SHARE MODE`, `INTO OUTFILE`, `INTO DUMPFILE`.
	Locking bool
	// Reason says why a read route must refuse, empty when it may not.
	Reason string
}

// Read reports whether a read route may run this statement at all.
func (s Statement) Read() bool {
	return s.Kind == Select || s.Kind == Explain || s.Kind == Show
}

// Classify reads one statement. Comments and string literals are blanked before anything is
// matched, so a keyword inside a literal can never change what the statement is judged to be.
func Classify(sql string) (Statement, error) {
	raw := strings.TrimSpace(sql)
	if raw == "" {
		return Statement{}, ErrEmpty
	}
	scrubbed, upper, err := scrub(raw)
	if err != nil {
		return Statement{}, err
	}
	if err := single(scrubbed); err != nil {
		return Statement{}, err
	}
	if len(tokenize(upper)) == 0 {
		// A statement can be nothing but a comment once the literals and comments are blanked.
		return Statement{}, ErrEmpty
	}

	statement := Statement{Kind: classifyKind(upper), Raw: raw}
	statement.Tables = tablesOf(scrubbed, upper)
	statement.Locking = locksOrWrites(upper)
	if statement.Locking {
		statement.Reason = "a locking read is not a read: it takes rows out of the server's control"
	}
	return statement, nil
}

// scrub blanks comments and the contents of literals, and returns the same text uppercased.
// Quotes and comment markers are replaced by spaces of the same length, so positions still line
// up with the original for anything that needs them later.
func scrub(sql string) (scrubbed, upper string, err error) {
	out := []byte(sql)
	for i := 0; i < len(sql); i++ {
		switch {
		case sql[i] == '-' && i+1 < len(sql) && sql[i+1] == '-':
			i = blankUntilLineEnd(out, i)
		case sql[i] == '#':
			i = blankUntilLineEnd(out, i)
		case sql[i] == '/' && i+1 < len(sql) && sql[i+1] == '*':
			end := strings.Index(sql[i+2:], "*/")
			if end < 0 {
				return "", "", fmt.Errorf("sqltext: an unterminated comment")
			}
			for j := i; j < i+2+end+2; j++ {
				blank(out, j)
			}
			i += 2 + end + 1
		case sql[i] == '\'' || sql[i] == '"' || sql[i] == '`':
			quote := sql[i]
			j := i + 1
			closed := false
			for ; j < len(sql); j++ {
				if sql[j] == '\\' {
					j++
					continue
				}
				if sql[j] == quote {
					// A doubled quote is an escape, not the end of the literal.
					if j+1 < len(sql) && sql[j+1] == quote {
						j++
						continue
					}
					closed = true
					break
				}
			}
			if !closed {
				return "", "", fmt.Errorf("sqltext: an unterminated %c literal", quote)
			}
			// Keep a quoted identifier readable; blank a string literal, which can hold anything.
			if quote != '`' {
				for k := i; k <= j; k++ {
					blank(out, k)
				}
			}
			i = j
		}
	}
	scrubbed = string(out)
	return scrubbed, strings.ToUpper(scrubbed), nil
}

func blank(out []byte, i int) {
	if i >= 0 && i < len(out) && out[i] != '\n' {
		out[i] = ' '
	}
}

func blankUntilLineEnd(out []byte, start int) int {
	i := start
	for ; i < len(out) && out[i] != '\n'; i++ {
		blank(out, i)
	}
	return i
}

// classifyKind names the statement from its first keyword, after any opening parentheses and
// after a WITH clause, whose body is the statement that matters.
func classifyKind(upper string) Kind {
	tokens := tokenize(upper)
	if len(tokens) == 0 {
		return Other
	}
	start := 0
	for start < len(tokens) && tokens[start] == "(" {
		start++
	}
	if start >= len(tokens) {
		return Other
	}
	if tokens[start] == "WITH" {
		// The first verb at depth zero after a CTE is the statement.
		depth := 0
		for i := start + 1; i < len(tokens); i++ {
			switch tokens[i] {
			case "(":
				depth++
			case ")":
				depth--
			case "SELECT", "INSERT", "UPDATE", "DELETE":
				if depth == 0 {
					return kindOfVerb(tokens[i])
				}
			}
		}
		return Other
	}
	return kindOfVerb(tokens[start])
}

func kindOfVerb(verb string) Kind {
	switch verb {
	case "SELECT":
		return Select
	case "EXPLAIN", "DESCRIBE", "DESC":
		return Explain
	case "SHOW":
		return Show
	case "INSERT", "REPLACE":
		return Insert
	case "UPDATE":
		return Update
	case "DELETE":
		return Delete
	}
	if ddlKeywords[verb] {
		return DDL
	}
	return Other
}

// locksOrWrites reports the reads that are not reads: a lock, or a write to the filesystem.
// `SELECT ... INTO @var` is not one of them: it writes a session variable, not a table.
func locksOrWrites(upper string) bool {
	for _, phrase := range []string{"FOR UPDATE", "LOCK IN SHARE MODE", "INTO OUTFILE", "INTO DUMPFILE"} {
		if strings.Contains(upper, phrase) {
			return true
		}
	}
	return false
}

// single refuses two statements in one request: a guard that reads only the first would happily
// pass the second, and the driver would happily run it.
func single(scrubbed string) error {
	trimmed := strings.TrimRight(scrubbed, " \t\r\n")
	if at := strings.IndexByte(trimmed, ';'); at >= 0 && strings.TrimSpace(trimmed[at+1:]) != "" {
		return ErrMultiple
	}
	return nil
}

// tablesOf reads the names after FROM, JOIN, UPDATE and INTO. It is best effort by design: a
// name it misses costs a rehearsal its scope later, and a name it invents would cost a reahearsal
// its correctness, so it only reports what plainly follows those keywords.
func tablesOf(scrubbed, upper string) []string {
	tokens := tokenize(scrubbed)
	uppers := tokenize(upper)
	if len(tokens) != len(uppers) {
		return nil
	}
	var tables []string
	for i := 0; i < len(uppers)-1; i++ {
		switch uppers[i] {
		case "FROM", "JOIN", "UPDATE", "INTO":
		default:
			continue
		}
		name := strings.Trim(tokens[i+1], "`")
		switch name {
		case "", "(", "SELECT":
			continue
		}
		tables = append(tables, name)
	}
	return tables
}

// tokenize splits on whitespace, keeping parentheses and commas as their own tokens so depth and
// list boundaries stay recoverable.
func tokenize(text string) []string {
	var (
		tokens []string
		start  = -1
	)
	flush := func(end int) {
		if start >= 0 {
			tokens = append(tokens, text[start:end])
			start = -1
		}
	}
	for i, r := range text {
		switch r {
		case ' ', '\t', '\n', '\r', ';', ',':
			flush(i)
		case '(', ')':
			flush(i)
			tokens = append(tokens, string(r))
		default:
			if start < 0 {
				start = i
			}
		}
	}
	flush(len(text))
	return tokens
}
