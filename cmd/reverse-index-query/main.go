package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"project_cat_reverse/internal/generator"
	"project_cat_reverse/internal/index"
	"project_cat_reverse/internal/query"
	"project_cat_reverse/internal/reader"
	"project_cat_reverse/internal/result"
	"project_cat_reverse/internal/scan"
	"slices"
	"time"
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
	if len(args) == 0 {
		return errors.New("command is required: generate, run or compare")
	}
	switch args[0] {

	case "generate":
		generateCmd := flag.NewFlagSet("generate", flag.ContinueOnError)
		count := generateCmd.Int(
			"count",
			0,
			"number of events to generate",
		)
		out := generateCmd.String(
			"out",
			"",
			"path to output file",
		)
		seed := generateCmd.Int64(
			"seed",
			0,
			"random seed",
		)

		if err := generateCmd.Parse(args[1:]); err != nil {
			return err
		}
		if *count < 1 {
			return errors.New("count must be positive")
		}
		if *out == "" {
			return errors.New("output file is required")
		}
		if err := generator.GenerateToFile(*count, *out, *seed); err != nil {
			return err
		}

	case "run":
		runCmd := flag.NewFlagSet("run", flag.ContinueOnError)
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
		var durationMS float64
		var indexBuildDurationMS float64
		var indexMemoryEstimateBytes uint64

		if *method == "scan" {
			start := time.Now()

			ids, err = scan.Execute(eventsData, &q)

			durationMS = float64(time.Since(start)) / float64(time.Millisecond)

		} else {
			buildStart := time.Now()

			idx.Build(eventsData)
			idx.Sort()

			indexBuildDurationMS =
				float64(time.Since(buildStart)) / float64(time.Millisecond)

			indexMemoryEstimateBytes = idx.MemoryEstimateBytes()

			start := time.Now()

			ids, err = idx.Execute(&q)

			durationMS =
				float64(time.Since(start)) / float64(time.Millisecond)
		}

		if err != nil {
			return err
		}

		res := result.Result{
			Method:                   *method,
			MatchedCount:             len(ids),
			MatchedIDs:               ids,
			Truncated:                false,
			DurationMS:               durationMS,
			IndexBuildDurationMS:     indexBuildDurationMS,
			IndexMemoryEstimateBytes: indexMemoryEstimateBytes,
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
		compareCmd := flag.NewFlagSet("compare", flag.ContinueOnError)
		events := compareCmd.String(
			"events",
			"",
			"path to events file",
		)
		queryFile := compareCmd.String(
			"query",
			"",
			"path to query file",
		)
		out := compareCmd.String(
			"out",
			"",
			"path to output file",
		)
		if err := compareCmd.Parse(args[1:]); err != nil {
			return err
		}
		if *events == "" {
			return errors.New("compare: --events is required")
		}
		if *queryFile == "" {
			return errors.New("compare: --query is required")
		}
		if *out == "" {
			return errors.New("compare: --out is required")
		}

		eventsData, err := reader.ReadEvents(*events)
		if err != nil {
			return err
		}

		q, err := query.ReadQuery(*queryFile)
		if err != nil {
			return err
		}

		scanStart := time.Now()
		scanIDs, err := scan.Execute(eventsData, &q)
		scanDurationMS := float64(time.Since(scanStart)) / float64(time.Millisecond)
		if err != nil {
			return err
		}

		idx := index.NewIndex()
		indexBuildStart := time.Now()
		idx.Build(eventsData)
		idx.Sort()
		indexBuildDurationMS := float64(time.Since(indexBuildStart)) / float64(time.Millisecond)

		indexQueryStart := time.Now()
		indexIDs, err := idx.Execute(&q)
		indexQueryDurationMS := float64(time.Since(indexQueryStart)) / float64(time.Millisecond)
		if err != nil {
			return err
		}
		indexTotalDurationMS := indexBuildDurationMS + indexQueryDurationMS
		indexMemoryEstimateBytes := idx.MemoryEstimateBytes()
		indexMemoryEstimateMiB := float64(indexMemoryEstimateBytes) / (1024 * 1024)

		equal := slices.Equal(scanIDs, indexIDs)
		report := fmt.Sprintf(
			"# Compare report\n\n"+
				"- Events: %d\n"+
				"- Scan matched: %d\n"+
				"- Index matched: %d\n"+
				"- Results equal: %t\n"+
				"- Scan duration: %.4f ms\n"+
				"- Index build duration: %.4f ms\n"+
				"- Index query duration: %.4f ms\n"+
				"- Index total duration: %.4f ms\n"+
				"- Index memory estimate: %d bytes (%.2f MiB)\n",
			len(eventsData),
			len(scanIDs),
			len(indexIDs),
			equal,
			scanDurationMS,
			indexBuildDurationMS,
			indexQueryDurationMS,
			indexTotalDurationMS,
			indexMemoryEstimateBytes,
			indexMemoryEstimateMiB,
		)
		if err := os.WriteFile(*out, []byte(report), 0644); err != nil {
			return err
		}

		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])

	}
	return nil
}
