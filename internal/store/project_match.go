package store

import "strings"

// ProjectMatchKey folds a human-entered project label so spellings like
// "elementAdds", "element-adds", and "Element_Adds" resolve to the same key.
func ProjectMatchKey(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '-', r == '_', r == ' ', r == '\t', r == '\n', r == '\r':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// appendProjectFilter adds a WHERE clause fragment for fuzzy project matching.
// column must be a SQL column reference (e.g. "o.project", "project", "s.project").
// Empty project adds no filter.
func appendProjectFilter(q string, args []interface{}, column, project string) (string, []interface{}) {
	project = strings.TrimSpace(project)
	if project == "" {
		return q, args
	}
	key := ProjectMatchKey(project)
	cond := "lower(replace(replace(replace(trim(coalesce(" + column + ", '')), '-', ''), '_', ''), ' ', '')) = ?"
	return q + " AND " + cond, append(args, key)
}
