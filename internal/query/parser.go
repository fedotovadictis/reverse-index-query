package query

import (
	"errors"
	"fmt"
	"strings"
)

// ParseString преобразует строковый запрос в дерево Query.

//Поддерживаются:
//   - простые условия: department=dev
//   - оператор AND
//   - оператор OR
//   - приоритет AND выше OR
//   - группировка с помощью скобок

// Примеры:
//	department=dev
//	department=dev AND file_ext=pdf
//	department=dev OR file_ext=pdf AND channel=email
//	(department=dev OR department=finance) AND file_ext=pdf

func ParseString(input string) (Query, error) {
	input = strings.TrimSpace(input)
	if err := validateParentheses(input); err != nil {
		return Query{}, err
	}
	if input == "" {
		return Query{}, errors.New("query string is empty")
	}
	q, err := parseOr(input)
	if err != nil {
		return Query{}, err
	}
	if err := q.Validate(); err != nil {
		return Query{}, err
	}
	return q, nil
}
func splitTopLevel(input, operator string) []string {
	var parts []string

	depth := 0
	start := 0
	separator := " " + operator + " "

	for i := 0; i < len(input); i++ {
		switch input[i] {
		case '(':
			depth++
		case ')':
			depth--
		}

		if depth != 0 {
			continue
		}

		end := i + len(separator)

		if end <= len(input) &&
			strings.EqualFold(input[i:end], separator) {
			parts = append(
				parts,
				strings.TrimSpace(input[start:i]),
			)

			i += len(separator) - 1
			start = i + 1
		}
	}

	parts = append(
		parts,
		strings.TrimSpace(input[start:]),
	)

	return parts
}

func trimOuterParentheses(input string) string {
	input = strings.TrimSpace(input)

	for len(input) >= 2 && input[0] == '(' && input[len(input)-1] == ')' {
		depth := 0
		wrapsWholeExpression := true

		for i := 0; i < len(input); i++ {
			switch input[i] {
			case '(':
				depth++
			case ')':
				depth--
			}

			if depth == 0 && i < len(input)-1 {
				wrapsWholeExpression = false
				break
			}
		}

		if !wrapsWholeExpression {
			break
		}

		input = strings.TrimSpace(input[1 : len(input)-1])
	}

	return input
}

func parseAnd(input string) (Query, error) {
	parts := splitByAnd(input)

	if len(parts) == 1 {
		part := strings.TrimSpace(parts[0])

		if strings.HasPrefix(part, "(") && strings.HasSuffix(part, ")") {
			return parseOr(part)
		}

		return parseTerm(part)
	}

	var q Query

	for i, part := range parts {
		term, err := parseOr(part)
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

	return q, nil
}

func splitByAnd(input string) []string {
	return splitTopLevel(input, "AND")
}

func parseOr(input string) (Query, error) {
	input = trimOuterParentheses(input)
	parts := splitByOr(input)

	var q Query

	for i, part := range parts {
		expr, err := parseAnd(part)
		if err != nil {
			return Query{}, err
		}

		if i == 0 {
			q = expr
			continue
		}

		left := q
		right := expr

		q = Query{
			Op:    Or,
			Left:  &left,
			Right: &right,
		}
	}

	return q, nil
}

func splitByOr(input string) []string {
	return splitTopLevel(input, "OR")
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
func validateParentheses(input string) error {
	depth := 0

	for _, ch := range input {
		switch ch {
		case '(':
			depth++

		case ')':
			depth--

			if depth < 0 {
				return fmt.Errorf("unbalanced parentheses")
			}
		}
	}

	if depth != 0 {
		return fmt.Errorf("unbalanced parentheses")
	}

	return nil
}
