package result

type Result struct {
	Method                   string   `json:"method"`
	MatchedCount             int      `json:"matched_count"`
	MatchedIDs               []uint64 `json:"matched_ids"`
	Truncated                bool     `json:"truncated"`
	DurationMS               float64  `json:"duration_ms"`
	IndexBuildDurationMS     float64  `json:"index_build_duration_ms,omitempty"`
	IndexMemoryEstimateBytes uint64   `json:"index_memory_estimate_bytes,omitempty"`
}
