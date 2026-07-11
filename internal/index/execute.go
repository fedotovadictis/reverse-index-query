package index

import (
	"fmt"
	"project_cat_reverse/internal/query"
)

func (idx *Index) Execute(q *query.Query) ([]uint64, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}

	switch q.Op {
	case query.Term:
		return idx.Find(q.Field, q.Value), nil

	case query.And:
		leftIDs, err := idx.Execute(q.Left)
		if err != nil {
			return nil, err
		}
		rightIDs, err := idx.Execute(q.Right)
		if err != nil {
			return nil, err
		}
		return Intersect(leftIDs, rightIDs), nil

	case query.Or:
		leftIDs, err := idx.Execute(q.Left)
		if err != nil {
			return nil, err
		}
		rightIDs, err := idx.Execute(q.Right)
		if err != nil {
			return nil, err
		}
		return Union(leftIDs, rightIDs), nil

	default:
		return nil, fmt.Errorf("unknown operator: %s", q.Op)
	}

}
