package query

import (
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
	file, err := os.Open(fileName)
	if err != nil {
		return Query{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	q := Query{}
	err = decoder.Decode(&q)
	if err != nil {
		return q, err
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
