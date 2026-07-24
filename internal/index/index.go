package index

import (
	"project_cat_reverse/internal/event"
	"project_cat_reverse/internal/idset"
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
	idx.Fields = make(map[string]map[string][]uint64)

	for _, evt := range events {
		idx.AddEvent(evt)
	}
	idx.Sort()
}

func (idx *Index) Sort() {
	for _, values := range idx.Fields {
		for value, ids := range values {
			sort.Slice(ids, func(i, j int) bool {
				return ids[i] < ids[j]
			})

			values[value] = idset.UniqueSorted(ids)
		}
	}
}

func (idx *Index) MemoryEstimateBytes() uint64 {
	var postingEntries uint64

	for _, values := range idx.Fields {
		for _, ids := range values {
			postingEntries += uint64(len(ids))
		}
	}

	return postingEntries * 8
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
func (idx *Index) And(
	leftField string,
	leftValue string,
	rightField string,
	rightValue string,
) []uint64 {
	leftIDs := idx.Find(leftField, leftValue)
	rightIDs := idx.Find(rightField, rightValue)

	// Intersect the shorter posting list first.
	if len(leftIDs) > len(rightIDs) {
		leftIDs, rightIDs = rightIDs, leftIDs
	}

	return Intersect(leftIDs, rightIDs)
}

func Union(left []uint64, right []uint64) []uint64 { // Union merges two sorted ID lists without duplicates.
	var result []uint64
	i := 0
	j := 0
	for i < len(left) && j < len(right) {
		if left[i] == right[j] {
			result = append(result, left[i])
			i++
			j++
		} else if left[i] < right[j] {
			result = append(result, left[i])
			i++
		} else {
			result = append(result, right[j])
			j++
		}
	}
	for i < len(left) {
		result = append(result, left[i])
		i++
	}

	for j < len(right) {
		result = append(result, right[j])
		j++
	}
	return result
}
func (idx *Index) Or(
	leftField string,
	leftValue string,
	rightField string,
	rightValue string,
) []uint64 {
	leftIDs := idx.Find(leftField, leftValue)
	rightIDs := idx.Find(rightField, rightValue)
	return Union(leftIDs, rightIDs)
}
