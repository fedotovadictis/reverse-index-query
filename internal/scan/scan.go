package scan

import (
	"fmt"
	"project_cat_reverse/internal/event"
	"project_cat_reverse/internal/query"
	"sort"
)

func Execute(events []event.Event, q *query.Query) ([]uint64, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}

	var ids []uint64
	for _, evt := range events {
		ok, err := match(evt, q)
		if err != nil {
			return nil, err
		}
		if ok {
			ids = append(ids, evt.ID)
		}

	}
	return normalizeIDs(ids), nil
}

func match(evt event.Event, q *query.Query) (bool, error) {
	switch q.Op {
	case query.Term:
		switch q.Field {
		case "department":
			return evt.Department == q.Value, nil
		case "action":
			return evt.Action == q.Value, nil
		case "channel":
			return evt.Channel == q.Value, nil
		case "file_ext":
			return evt.FileExt == q.Value, nil
		case "destination_type":
			return evt.DestinationType == q.Value, nil
		case "severity":
			return evt.Severity == q.Value, nil
		case "user_id":
			return evt.UserID == q.Value, nil
		default:
			return false, fmt.Errorf("unknown field: %s", q.Field)
		}

	case query.And:
		leftMatch, err := match(evt, q.Left)
		if err != nil {
			return false, err
		}

		rightMatch, err := match(evt, q.Right)
		if err != nil {
			return false, err
		}

		return leftMatch && rightMatch, nil

	case query.Or:
		leftMatch, err := match(evt, q.Left)
		if err != nil {
			return false, err
		}

		rightMatch, err := match(evt, q.Right)
		if err != nil {
			return false, err
		}

		return leftMatch || rightMatch, nil
	}

	return false, fmt.Errorf("unknown operator: %s", q.Op)
}
func normalizeIDs(ids []uint64) []uint64 {
	if len(ids) == 0 {
		return []uint64{}
	}

	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	result := make([]uint64, 0, len(ids))
	result = append(result, ids[0])

	for _, id := range ids[1:] {
		if id != result[len(result)-1] {
			result = append(result, id)
		}
	}

	return result
}
