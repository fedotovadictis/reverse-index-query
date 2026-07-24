package reader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"project_cat_reverse/internal/event"
	"strings"
)

func ReadEvents(fileName string) (events []event.Event, err error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("open events file: %w", err)
	}

	defer func() {
		closeErr := file.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close events file: %w", closeErr)
		}
	}()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++

		line := scanner.Text()
		line = strings.TrimPrefix(line, "\uFEFF")

		var read event.Event
		if unmarshalErr := json.Unmarshal([]byte(line), &read); unmarshalErr != nil {
			return nil, fmt.Errorf(
				"decode event on line %d: %w",
				lineNumber,
				unmarshalErr,
			)
		}

		events = append(events, read)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf(
			"read event on line %d: %w",
			lineNumber+1,
			scanErr,
		)
	}

	return events, nil
}
