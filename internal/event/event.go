package event

type Event struct {
	ID              uint64 `json:"id"`
	Timestamp       string `json:"timestamp"`
	UserID          string `json:"user_id"`
	Department      string `json:"department"`
	Action          string `json:"action"`
	Channel         string `json:"channel"`
	FileExt         string `json:"file_ext"`
	DestinationType string `json:"destination_type"`
	Severity        string `json:"severity"`
}
