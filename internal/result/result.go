package result

type Result struct {
	Method       string   `json:"method"`
	MatchedCount int      `json:"matched_count"`
	MatchedIDs   []uint64 `json:"matched_ids"`
	Truncated    bool     `json:"truncated"`
	DurationMS   float64  `json:"duration_ms"`
}
