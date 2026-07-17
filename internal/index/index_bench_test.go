package index

import "testing"

func makeIDs(size int, step uint64) []uint64 {
	ids := make([]uint64, size)

	for i := range ids {
		ids[i] = uint64(i) * step
	}

	return ids
}
func BenchmarkIntersect(b *testing.B) {
	left := makeIDs(100_000, 2)
	right := makeIDs(100_000, 3)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Intersect(left, right)
	}
}
func BenchmarkUnion(b *testing.B) {

	left := makeIDs(100_000, 2)
	right := makeIDs(100_000, 3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Union(left, right)
	}
}
