package reader

import (
	"bufio"
	"encoding/json"
	"os"
	"project_cat_reverse/internal/event"
)

func ReadEvents(fileName string) ([]event.Event, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	var events []event.Event

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		// здесь обрабатываем одну строку
		line := scanner.Text()

		var read event.Event

		err := json.Unmarshal([]byte(line), &read)
		if err != nil {
			return nil, err
		}
		events = append(events, read)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	defer func() {
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}()
	return events, nil
}
