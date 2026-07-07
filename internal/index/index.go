package index

import "project_cat_reverse/internal/event"

type Index struct {
	Fields map[string]map[string][]uint64
}

func NewIndex() Index {
	return Index{
		Fields: make(map[string]map[string][]uint64),
	}
}

func (idx *Index) AddDepartment(evt event.Event) {
	departmentMap, ok := idx.Fields["department"]
	if !ok {
		departmentMap = make(map[string][]uint64)
	}
	idx.Fields["department"] = departmentMap

	ids, ok := departmentMap[evt.Department]
	if !ok {
		ids = make([]uint64, 0)
	}
	ids = append(ids, evt.ID)
	departmentMap[evt.Department] = ids
}
