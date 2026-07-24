package idset

import "sort"

func Normalize(ids []uint64) []uint64 {
	if len(ids) == 0 {
		return []uint64{}
	}

	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	return UniqueSorted(ids)
}

func UniqueSorted(ids []uint64) []uint64 {
	if len(ids) == 0 {
		return []uint64{}
	}

	result := make([]uint64, 0, len(ids))
	result = append(result, ids[0])

	for _, id := range ids[1:] {
		if id != result[len(result)-1] {
			result = append(result, id)
		}
	}

	return result
}
