package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"project_cat_reverse/internal/index"
	"project_cat_reverse/internal/query"
	"project_cat_reverse/internal/reader"
	"project_cat_reverse/internal/result"
	"project_cat_reverse/internal/scan"
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
		fmt.Println("generate")

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

		eventsData, err := reader.ReadEvents(*events)
		if err != nil {
			return err
		}

		idx := index.NewIndex()

		q, err := query.ReadQuery(*queryFile)
		if err != nil {
			return err
		}

		var ids []uint64

		if *method == "scan" {
			ids, err = scan.Execute(eventsData, &q)
		} else {
			ids, err = idx.Execute(&q)
			idx.Build(eventsData)
			idx.Sort()

			ids, err = idx.Execute(&q)
		}

		if err != nil {
			return err
		}

		res := result.Result{
			Method:       *method,
			MatchedCount: len(ids),
			MatchedIDs:   ids,
			Truncated:    false,
			DurationMS:   0,
		}

		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return err
		}

		if err := os.WriteFile(*out, data, 0644); err != nil {
			return err
		}
		return nil

	case "compare":
		fmt.Println("compare")

	default:
		return fmt.Errorf("unknown command: %s", args[0])

	}
	return nil
}
