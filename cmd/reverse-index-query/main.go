package main

import (
	"fmt"
	"project_cat_reverse/internal/index"
	"project_cat_reverse/internal/reader"
)

func main() {
	/*
		err := generator.GenerateToFile(5, "events.jsonl")
		if err != nil {
			fmt.Println(err)
			return
		}
	*/

	events, err := reader.ReadEvents("events.jsonl")
	if err != nil {
		fmt.Println(err)
		return
	}

	idx := index.NewIndex()

	for _, evt := range events {
		idx.AddDepartment(evt)
	}

	fmt.Println(idx.Fields)
}
