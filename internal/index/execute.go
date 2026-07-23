package index

import (
	"fmt"
	"project_cat_reverse/internal/query"
	"sort"
)

func (idx *Index) Execute(q *query.Query) ([]uint64, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}

	switch q.Op {
	case query.Term:
		return idx.Find(q.Field, q.Value), nil

	case query.And:
		return idx.executeAnd(q)

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

// executeAnd собирает все связанные условия AND,
// сортирует их результаты по размеру и пересекает,
// начиная с самого маленького набора.
func (idx *Index) executeAnd(q *query.Query) ([]uint64, error) {
	var operands []*query.Query
	collectAndOperands(q, &operands)

	results := make([][]uint64, 0, len(operands))

	for _, operand := range operands {
		ids, err := idx.Execute(operand)
		if err != nil {
			return nil, err
		}

		// Если хотя бы одно условие ничего не нашло,
		// все выражение AND также пустое.
		if len(ids) == 0 {
			return []uint64{}, nil
		}

		results = append(results, ids)
	}

	sortPostingListsByLength(results)

	matchedIDs := append([]uint64(nil), results[0]...)

	for i := 1; i < len(results); i++ {
		matchedIDs = Intersect(matchedIDs, results[i])

		if len(matchedIDs) == 0 {
			break
		}
	}

	return matchedIDs, nil
}

// collectAndOperands превращает дерево:
//
//	A AND (B AND C)
//
// в список:
//
//	A, B, C
//
// Выражения OR и TERM остаются отдельными операндами.
func collectAndOperands(q *query.Query, operands *[]*query.Query) {
	if q.Op == query.And {
		collectAndOperands(q.Left, operands)
		collectAndOperands(q.Right, operands)
		return
	}

	*operands = append(*operands, q)
}
func sortPostingListsByLength(postingLists [][]uint64) {
	sort.SliceStable(postingLists, func(i, j int) bool {
		return len(postingLists[i]) < len(postingLists[j])
	})
}
