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
	idx := index.NewIndex()

	idx.AddField("department", "sales", 10)
	idx.AddField("department", "sales", 3)
	idx.AddField("department", "sales", 7)
	idx.AddField("department", "sales", 1)

	fmt.Println("До сортировки:")
	fmt.Println(idx.Fields)

	idx.Sort()

	fmt.Println("После сортировки:")
	fmt.Println(idx.Fields)
}
