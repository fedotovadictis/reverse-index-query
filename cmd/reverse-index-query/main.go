package main

import (
	"fmt"
	"project_cat_reverse/internal/index"
)

func main() {
	//объединение
	/*left := []uint64{1, 3, 5}
	right := []uint64{3, 4, 6}
	fmt.Println(index.Union(left, right)) */ //result [1 3 4 5 6]

	//нет общих эл
	/* left := []uint64{1, 2, 3}
	right := []uint64{4, 5, 6}
	fmt.Println(index.Union(left, right))*/ // [1 2 3 4 5 6]

	//плное совпадение
	/* left := []uint64{1, 2, 3}
	right := []uint64{1, 2, 3}
	fmt.Println(index.Union(left, right)) */ //[1 2 3]

	//один список пустой
	left := []uint64{}
	right := []uint64{1, 2, 3}
	fmt.Println(index.Union(left, right)) //[1 2 3]

}
