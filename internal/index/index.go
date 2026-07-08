package index

import (
	"project_cat_reverse/internal/event"
	"sort"
)

type Index struct {
	Fields map[string]map[string][]uint64
}

func NewIndex() Index {
	return Index{
		Fields: make(map[string]map[string][]uint64),
	}
}
func (idx *Index) AddField(field string, value string, id uint64) {
	if value == "" {
		return
	}
	fieldMap, ok := idx.Fields[field]
	if !ok {
		fieldMap = make(map[string][]uint64)
	}

	ids, ok := fieldMap[value]
	if !ok {
		ids = make([]uint64, 0)
	}
	ids = append(ids, id)
	fieldMap[value] = ids
	idx.Fields[field] = fieldMap
}
func (idx *Index) AddEvent(evt event.Event) {
	idx.AddField("department", evt.Department, evt.ID)
	idx.AddField("action", evt.Action, evt.ID)
	idx.AddField("channel", evt.Channel, evt.ID)
	idx.AddField("file_ext", evt.FileExt, evt.ID)
	idx.AddField("destination_type", evt.DestinationType, evt.ID)
	idx.AddField("severity", evt.Severity, evt.ID)
	idx.AddField("user_id", evt.UserID, evt.ID)
}

func (idx *Index) Build(events []event.Event) {
	for _, evt := range events {
		idx.AddEvent(evt)
	}

}
func (idx *Index) Sort() {
	for _, values := range idx.Fields {
		for value, ids := range values {
			sort.Slice(ids, func(i, j int) bool {
				return ids[i] < ids[j]
			})
			values[value] = ids
		}
	}
}

func (idx *Index) Find(field string, value string) []uint64 {
	fieldMap, ok := idx.Fields[field]
	if !ok {
		return []uint64{}
	}

	ids, ok := fieldMap[value]
	if !ok {
		return []uint64{}
	}
	return ids
}

func Intersect(left []uint64, right []uint64) []uint64 {
	var result []uint64
	i := 0
	j := 0
	for i < len(left) && j < len(right) {
		if left[i] == right[j] {
			result = append(result, left[i])
			i++
			j++
		} else if left[i] < right[j] {
			i++
		} else {
			j++
		}
	}
	return result
}
