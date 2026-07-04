package generator

import "project_cat_reverse/internal/event"

func GenerateEvent(id uint64) event.Event {
	return event.Event{
		ID: id,
	}
}
