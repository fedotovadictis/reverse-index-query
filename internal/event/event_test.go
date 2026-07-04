package event

import (
	"encoding/json"
	"testing"
)

func TestEventMarshalUnmarshal(t *testing.T) {
	original := Event{
		ID:              1,
		Timestamp:       "2026-06-16T10:00:00Z",
		UserID:          "user_017",
		Department:      "sales",
		Action:          "email_send",
		Channel:         "email",
		FileExt:         "xlsx",
		DestinationType: "external",
		Severity:        "high",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored Event
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatal(err)
	}
	if original != restored {
		t.Fatal("restored event is not the same as original", original, restored)
	}
}
