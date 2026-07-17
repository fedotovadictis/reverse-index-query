package generator

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"project_cat_reverse/internal/event"
	"time"
)

var departments = []string{"sales", "hr", "finance", "dev", "legal", "support"}
var actions = []string{"open_file", "copy_file", "email_send", "print_file", "copy_to_usb", "cloud_upload"}
var channels = []string{"local", "email", "usb", "printer", "cloud"}
var fileExtensions = []string{"docx", "xlsx", "pdf", "zip", "go", "sql"}
var destinationTypes = []string{"none", "internal", "external", "usb", "cloud", "printer"}
var severities = []string{"low", "medium", "high", "critical"}

func GenerateEvent(id uint64, rng *rand.Rand, baseTime time.Time) event.Event {
	department := departments[rng.Intn(len(departments))]
	action := actions[rng.Intn(len(actions))]
	channel := channels[rng.Intn(len(channels))]

	fileExt := fileExtensions[rng.Intn(len(fileExtensions))]
	destinationType := destinationTypes[rng.Intn(len(destinationTypes))]
	severity := severities[rng.Intn(len(severities))]

	userNumber := rng.Intn(5000) + 1
	userID := fmt.Sprintf("user_%04d", userNumber)

	randomHours := rng.Intn(721)
	randomTime := baseTime.Add(-time.Duration(randomHours) * time.Hour)
	timestamp := randomTime.Format(time.RFC3339)

	return event.Event{
		ID:              id,
		Department:      department,
		Action:          action,
		Channel:         channel,
		FileExt:         fileExt,
		DestinationType: destinationType,
		Severity:        severity,
		UserID:          userID,
		Timestamp:       timestamp,
	}
}
func GenerateToFile(count int, fileName string, seed int64) (err error) {
	if count < 1 {
		return errors.New("count must be positive")
	}
	if fileName == "" {
		return errors.New("output file is required")
	}
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}()
	rng := rand.New(rand.NewSource(seed))
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).
		Add(time.Duration(seed%8760) * time.Hour)

	for i := 1; i <= count; i++ {

		evt := GenerateEvent(uint64(i), rng, baseTime)

		data, err := json.Marshal(evt)
		if err != nil {
			return err
		}

		_, err = file.Write(data)
		if err != nil {
			return err
		}

		_, err = file.Write([]byte("\n"))
		if err != nil {
			return err
		}
	}
	return nil
}
