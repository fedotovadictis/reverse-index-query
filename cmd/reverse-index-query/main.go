package main

import (
	"fmt"
	"project_cat_reverse/internal/index"
)

func main() {
	/*
		err := generator.GenerateToFile(5, "events.jsonl")
		if err != nil {
			fmt.Println(err)
			return
		}
	*/
	/*
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
	*/
	left := []uint64{1, 2, 3}
	right := []uint64{1, 2, 3}

	fmt.Println(index.Intersect(left, right))
}
