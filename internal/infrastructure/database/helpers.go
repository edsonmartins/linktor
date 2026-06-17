package database

import "strconv"

// derefString returns the pointed-to string, or "" when the pointer is nil.
// Mirror of nullString (which maps "" -> nil) for scanning nullable columns.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// itoa is a short alias for strconv.Itoa used when building positional SQL
// placeholders ($1, $2, ...).
func itoa(i int) string {
	return strconv.Itoa(i)
}
