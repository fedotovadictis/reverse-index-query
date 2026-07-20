package query

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const (
	Term = "TERM"
	And  = "AND"
	Or   = "OR"
)

type Query struct {
	Op    string `json:"op"`
	Field string `json:"field"`
	Value string `json:"value"`
	Left  *Query `json:"left"`
	Right *Query `json:"right"`
}

func ReadQuery(fileName string) (Query, error) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return Query{}, err
	}

	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	var q Query
	if err := json.Unmarshal(data, &q); err != nil {
		return Query{}, err
	}

	return q, nil
}
func (q *Query) Validate() error {
	if q == nil {
		return errors.New("nil query")
	}
	switch q.Op {

	case Term:
		if q.Field == "" {
			return errors.New("term query: field is empty")
		}
		if q.Value == "" {
			return errors.New("term query: value is empty")
		}
		if q.Left != nil {
			return errors.New("term query: left operand is not allowed")
		}
		if q.Right != nil {
			return errors.New("term query: right operand is not allowed")
		}
		if !IsValidField(q.Field) {
			return fmt.Errorf("unknown field: %s", q.Field)
		}

	case And:
		if q.Left == nil {
			return errors.New("and query: missing left operand")
		}
		if q.Right == nil {
			return errors.New("and query: missing right operand")
		}
		if err := q.Left.Validate(); err != nil {
			return err
		}
		if err := q.Right.Validate(); err != nil {
			return err
		}

	case Or:
		if q.Left == nil {
			return errors.New("or query: missing left operand")
		}
		if q.Right == nil {
			return errors.New("or query: missing right operand")
		}
		if err := q.Left.Validate(); err != nil {
			return err
		}
		if err := q.Right.Validate(); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown operator: %s", q.Op)

	}
	return nil

}
func IsValidField(field string) bool {
	allowedFields := []string{
		"user_id",
		"department",
		"action",
		"channel",
		"file_ext",
		"destination_type",
		"severity",
	}
	for _, allowedField := range allowedFields {
		if allowedField == field {
			return true
		}
	}
	return false
}
