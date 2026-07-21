package query

import (
	"errors"
	"fmt"
	"strings"
)

// ParseString преобразует строковый запрос в дерево Query.
// Поддерживаются:
//
//	department=dev
//	department=dev AND file_ext=pdf
//	department=dev AND file_ext=pdf AND channel=email
func ParseString(input string) (Query, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Query{}, errors.New("query string is empty")
	}

	parts := splitByAnd(input)

	var q Query

	for i, part := range parts {
		term, err := parseTerm(part)
		if err != nil {
			return Query{}, err
		}

		if i == 0 {
			q = term
			continue
		}

		left := q
		right := term

		q = Query{
			Op:    And,
			Left:  &left,
			Right: &right,
		}
	}

	if err := q.Validate(); err != nil {
		return Query{}, err
	}

	return q, nil
}

func splitByAnd(input string) []string {
	normalized := strings.ReplaceAll(input, " and ", " AND ")
	normalized = strings.ReplaceAll(normalized, " And ", " AND ")
	normalized = strings.ReplaceAll(normalized, " aNd ", " AND ")
	normalized = strings.ReplaceAll(normalized, " anD ", " AND ")
	normalized = strings.ReplaceAll(normalized, " ANd ", " AND ")
	normalized = strings.ReplaceAll(normalized, " AnD ", " AND ")
	normalized = strings.ReplaceAll(normalized, " aND ", " AND ")

	return strings.Split(normalized, " AND ")
}

func parseTerm(input string) (Query, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Query{}, errors.New("empty query term")
	}

	parts := strings.SplitN(input, "=", 2)
	if len(parts) != 2 {
		return Query{}, fmt.Errorf(
			`query term %q must have format "field=value"`,
			input,
		)
	}

	field := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	if field == "" {
		return Query{}, errors.New("query field is empty")
	}

	if value == "" {
		return Query{}, errors.New("query value is empty")
	}

	if !IsValidField(field) {
		return Query{}, fmt.Errorf("unknown field: %s", field)
	}

	return Query{
		Op:    Term,
		Field: field,
		Value: value,
	}, nil
}
