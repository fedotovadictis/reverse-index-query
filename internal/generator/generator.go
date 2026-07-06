package generator

import (
	"fmt"
	"math/rand"
	"project_cat_reverse/internal/event"
	"time"
)

var departments = []string{"sales", "hr", "finance", "dev", "legal", "support"}
var actions = []string{"open_file", "copy_file", "email_send", "print_file", "copy_to_usb", "cloud_upload"}
var channels = []string{"local", "email", "usb", "printer", "cloud"}
var fileExtensions = []string{"docx", "xlsx", "pdf", "zip", "go", "sql"}
var destinationTypes = []string{"none", "internal", "external", "usb", "cloud", "printer"}
var severities = []string{"low", "medium", "high", "critical"}

func GenerateEvent(id uint64) event.Event {
	department := departments[rand.Intn(len(departments))]
	action := actions[rand.Intn(len(actions))]
	channel := channels[rand.Intn(len(channels))]

	fileExt := fileExtensions[rand.Intn(len(fileExtensions))]
	destinationType := destinationTypes[rand.Intn(len(destinationTypes))]
	severity := severities[rand.Intn(len(severities))]

	userNumber := rand.Intn(5000) + 1
	userID := fmt.Sprintf("user_%04d", userNumber)

	currentTime := time.Now().UTC()
	randomHours := rand.Intn(721)
	randomTime := currentTime.Add(-time.Duration(randomHours) * time.Hour)
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
