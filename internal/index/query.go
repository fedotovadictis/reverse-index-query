package index

import (
	"encoding/json"
	"os"
)

type Query struct {
	Op    string `json:"op"`
	Field string `json:"field"`
	Value string `json:"value"`
	Left  *Query `json:"left"`
	Right *Query `json:"right"`
}

func ReadQuery(fileName string) (Query, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return Query{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	q := Query{}
	err = decoder.Decode(&q)
	if err != nil {
		return q, err
	}
	return q, nil
}
