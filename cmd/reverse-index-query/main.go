package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"project_cat_reverse/internal/index"
	"project_cat_reverse/internal/query"
	"project_cat_reverse/internal/reader"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "command is required: generate, run or compare")
		os.Exit(1)
	}

	// если дошли сюда, значит команда есть
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	switch args[0] {

	case "generate":
		fmt.Println("run")

	case "run":
		runCmd := flag.NewFlagSet("run", flag.ExitOnError)
		events := runCmd.String(
			"events",
			"",
			"path to events file",
		)
		queryFile := runCmd.String(
			"query",
			"",
			"path to query file",
		)
		method := runCmd.String(
			"method",
			"",
			"search method (scan or index)",
		)
		out := runCmd.String(
			"out",
			"",
			"path to output file",
		)
		if err := runCmd.Parse(args[1:]); err != nil {
			return err
		}
		if *events == "" {
			return errors.New("run: --events is required")
		}
		if *queryFile == "" {
			return errors.New("run: --query is required")
		}
		if *method == "" {
			return errors.New("run: --method is required")
		}
		if *method != "scan" && *method != "index" {
			return fmt.Errorf("run: unknown method %q", *method)
		}
		if *out == "" {
			return errors.New("run: --out is required")
		}
		if *method == "scan" {
			return errors.New("run: scan method is not implemented yet")
		}

		eventsData, err := reader.ReadEvents(*events)
		if err != nil {
			return err
		}

		idx := index.NewIndex()
		for _, evt := range eventsData {
			idx.AddEvent(evt)
		}

		q, err := query.ReadQuery(*queryFile)
		if err != nil {
			return err
		}

		ids, err := idx.Execute(&q)
		if err != nil {
			return err
		}
		fmt.Println(ids)
		return nil

	case "compare":
		fmt.Println("run")

	default:

	}
	return nil
}
