package main

import (
	"fmt"
	"project_cat_reverse/internal/generator"
)

func main() {
	err := generator.GenerateToFile(5, "events.jsonl")
	if err != nil {
		fmt.Println(err)
		return
	}
}
