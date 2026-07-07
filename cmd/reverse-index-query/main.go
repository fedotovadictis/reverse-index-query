package main

import (
	"fmt"
	"project_cat_reverse/internal/generator"
	"project_cat_reverse/internal/reader"
)

func main() {
	err := generator.GenerateToFile(5, "events.jsonl")
	if err != nil {
		fmt.Println(err)
		return
	}
	events, err := reader.ReadEvents("events.jsonl")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(len(events))
	fmt.Println(len(events))

	if len(events) > 0 {
		fmt.Println(events[0])
	}
}
