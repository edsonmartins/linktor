package service

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// positionalPlaceholder matches Meta's positional variable syntax ({{1}}, {{2}}, ...).
// Named placeholders ({{first_name}}) use a separate matcher because they
// follow different validation rules (no count check, names must match examples).
var (
	positionalPlaceholder = regexp.MustCompile(`\{\{\s*(\d+)\s*\}\}`)
	namedPlaceholder      = regexp.MustCompile(`\{\{\s*([a-z][a-z0-9_]*)\s*\}\}`)
)

// validateTemplateComponents rejects payloads Meta would reject anyway:
// templates that declare variables in their text but don't attach an
// `example` with enough sample values. Catching it here saves a Graph API
// round-trip and gives the admin a readable error instead of Meta's 400.
//
// Rules enforced:
//   - BODY text with N positional variables ({{1}}..{{N}}) must have a
//     non-empty Example.BodyText row with at least N entries.
//   - HEADER text (format=TEXT) with N positional variables must have
//     Example.HeaderText with at least N entries.
//   - HEADER with media format (IMAGE/VIDEO/DOCUMENT) must have at least
//     one Example.HeaderHandle entry (Meta's spec).
//   - Named placeholders require matching keys in the example — but because
//     our entity only carries positional examples today, we just require
//     *some* example values rather than name-matching. This prevents the
//     obvious silent pass-through.
func validateTemplateComponents(components []entity.TemplateComponent) error {
	for i, c := range components {
		if err := validateComponent(i, c); err != nil {
			return err
		}
	}
	return nil
}

func validateComponent(index int, c entity.TemplateComponent) error {
	switch c.Type {
	case "BODY":
		count := maxPositionalIndex(c.Text)
		named := namedPlaceholderCount(c.Text)

		if count == 0 && named == 0 {
			return nil // no variables, no example needed
		}

		if c.Example == nil || len(c.Example.BodyText) == 0 {
			return fmt.Errorf("component[%d] BODY has variables but no example.body_text", index)
		}
		// Meta accepts multiple example rows; require at least one to hit the
		// expected arity so we don't send a degenerate payload.
		found := false
		for _, row := range c.Example.BodyText {
			if count > 0 && len(row) >= count {
				found = true
				break
			}
			if named > 0 && len(row) >= named {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf(
				"component[%d] BODY declares %d variable(s) but example.body_text rows are shorter",
				index, maxInt(count, named),
			)
		}

	case "HEADER":
		format := c.Format
		if format == "" {
			format = "TEXT" // Meta's implicit default
		}
		switch format {
		case "TEXT":
			count := maxPositionalIndex(c.Text)
			named := namedPlaceholderCount(c.Text)
			if count == 0 && named == 0 {
				return nil
			}
			if c.Example == nil || len(c.Example.HeaderText) < maxInt(count, named) {
				return fmt.Errorf(
					"component[%d] HEADER text declares %d variable(s) but example.header_text has %d",
					index, maxInt(count, named), headerTextLen(c.Example),
				)
			}
		case "IMAGE", "VIDEO", "DOCUMENT":
			if c.Example == nil || len(c.Example.HeaderHandle) == 0 {
				return fmt.Errorf(
					"component[%d] HEADER format=%s requires example.header_handle",
					index, format,
				)
			}
		}

	case "FOOTER":
		// Footers are static text; Meta does not allow variables there.
		if maxPositionalIndex(c.Text) > 0 || namedPlaceholderCount(c.Text) > 0 {
			return fmt.Errorf("component[%d] FOOTER must not contain variables", index)
		}

	case "BUTTONS":
		// URL buttons may embed a single {{1}} in the href; this is validated
		// at button creation on Meta's side. We only reject the trivially
		// wrong case where a button text carries variables (Meta disallows it).
		for j, b := range c.Buttons {
			if maxPositionalIndex(b.Text) > 0 || namedPlaceholderCount(b.Text) > 0 {
				return fmt.Errorf("component[%d].buttons[%d] text must not contain variables", index, j)
			}
		}
	}
	return nil
}

// maxPositionalIndex returns the highest N in {{N}} placeholders found in s.
// This is the count Meta expects in the example when using positional format.
func maxPositionalIndex(s string) int {
	matches := positionalPlaceholder.FindAllStringSubmatch(s, -1)
	max := 0
	for _, m := range matches {
		n, err := strconv.Atoi(m[1])
		if err == nil && n > max {
			max = n
		}
	}
	return max
}

// namedPlaceholderCount returns the number of distinct {{name}} placeholders.
func namedPlaceholderCount(s string) int {
	matches := namedPlaceholder.FindAllStringSubmatch(s, -1)
	seen := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		seen[m[1]] = struct{}{}
	}
	return len(seen)
}

func headerTextLen(e *entity.TemplateExample) int {
	if e == nil {
		return 0
	}
	return len(e.HeaderText)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// validateParameterFormat enforces mutual consistency between the declared
// parameter_format and the placeholders actually used in the components.
// Meta rejects templates that mix named and positional placeholders in the
// same component, and it requires named placeholders to be lowercase /
// underscore-safe identifiers.
func validateParameterFormat(format entity.TemplateParameterFormat, components []entity.TemplateComponent) error {
	for i, c := range components {
		texts := collectVariableTexts(c)
		for _, txt := range texts {
			hasPos := positionalPlaceholder.MatchString(txt)
			hasNamed := namedPlaceholder.MatchString(txt)

			if hasPos && hasNamed {
				return fmt.Errorf("component[%d] mixes positional and named placeholders; pick one", i)
			}

			switch format {
			case entity.TemplateParameterFormatNamed:
				if hasPos {
					return fmt.Errorf("component[%d] uses positional placeholders but parameter_format=NAMED", i)
				}
			case entity.TemplateParameterFormatPositional, "":
				if hasNamed {
					return fmt.Errorf("component[%d] uses named placeholders but parameter_format=POSITIONAL (the default); set parameter_format=NAMED", i)
				}
			}
		}
	}
	return nil
}

// collectVariableTexts returns every text-bearing string in a component
// that Meta will scan for placeholders. Buttons/footers are not included
// here because they cannot contain variables (enforced in validateComponent).
func collectVariableTexts(c entity.TemplateComponent) []string {
	switch c.Type {
	case "BODY", "HEADER":
		return []string{c.Text}
	}
	return nil
}
