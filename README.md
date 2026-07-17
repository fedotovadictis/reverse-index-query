# Reverse Index Query

## Description

This project implements a simplified inverted index for events. It generates event data, builds an inverted index for selected fields, executes boolean queries (TERM, AND, OR), and compares the results with a full scan.

## Requirements

- Go 1.26 or later

## Build

```powershell
go build -o reverse-index-query.exe ./cmd/reverse-index-query
```

## Generate events

Generate a JSONL file with test events.

```powershell
.\reverse-index-query.exe generate --count 1000 --out events.jsonl --seed 42
```

### Parameters

| Flag | Description |
|------|-------------|
| `--count` | Number of events to generate |
| `--out` | Output JSONL file |
| `--seed` | Random seed for deterministic generation |

## Run query (scan)

Execute a query using a full scan.

```powershell
.\reverse-index-query.exe run `
  --events events.jsonl `
  --query query.json `
  --method scan `
  --out scan.json
```

## Run query (index)

Execute a query using the inverted index.

```powershell
.\reverse-index-query.exe run `
  --events events.jsonl `
  --query query.json `
  --method index `
  --out index.json
```

## Compare scan and index

Compare the results of the scan-based and index-based execution.

```powershell
.\reverse-index-query.exe compare `
  --events events.jsonl `
  --query query.json `
  --out compare_report.md
```

## Input format

### Event

Events are stored in JSONL format (one JSON object per line).

Example:

```json
{
  "id": 1,
  "timestamp": "2026-06-16T10:00:00Z",
  "user_id": "user_017",
  "department": "sales",
  "action": "email_send",
  "channel": "email",
  "file_ext": "xlsx",
  "destination_type": "external",
  "severity": "high"
}
```

Indexed fields:

- user_id
- department
- action
- channel
- file_ext
- destination_type
- severity

### Query

Queries are provided as a JSON tree.

Example:

```json
{
  "op": "AND",
  "left": {
    "op": "TERM",
    "field": "channel",
    "value": "email"
  },
  "right": {
    "op": "TERM",
    "field": "destination_type",
    "value": "external"
  }
}
```

Supported operators:

- TERM
- AND
- OR

## Output format

Example:

```json
{
  "method": "index",
  "matched_count": 1422,
  "matched_ids": [1, 7, 29, 44],
  "truncated": true,
  "duration_ms": 3.1
}
```

## Algorithm

1. Read events from a JSONL file.
2. Build an inverted index for indexed fields.
3. Parse the query into a tree.
4. Execute the query using either:
   - full scan;
   - inverted index.
5. Return matching event IDs and execution statistics.

## Benchmark results

Benchmarks were executed using:

```powershell
go test ./internal/index -bench="."
```

Results:

| Benchmark | Result |
|-----------|--------:|
| BenchmarkIntersect | 233323 ns/op |
| BenchmarkUnion | 868554 ns/op |

Test environment:

- OS: Windows
- Architecture: amd64
- CPU: 13th Gen Intel(R) Core(TM) i5-13400F

## Control dataset

A deterministic control dataset is provided in `testdata/control`.

Query:

```text
department = dev AND file_ext = pdf
```

Expected result:

- matched_count = 2
- matched_ids = [9, 20]

Comparison report:

- Scan matched: 2
- Index matched: 2
- Results equal: true

## Current limitations

- Only JSON query trees are supported.
- Supported operators are TERM, AND and OR.
- NOT queries are not implemented.
- Only exact value matching is supported.
- The inverted index is built entirely in memory.
- Range and wildcard queries are not supported.