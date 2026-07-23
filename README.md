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

## Run query from command line

A query can also be passed directly using `--query-string`.

### Scan

```powershell
.\reverse-index-query.exe run `
  --events events.jsonl `
  --query-string "department=sales AND (file_ext=pdf OR channel=email)" `
  --method scan `
  --out scan.json
```

### Index

```powershell
.\reverse-index-query.exe run `
  --events events.jsonl `
  --query-string "department=sales AND (file_ext=pdf OR channel=email)" `
  --method index `
  --out index.json
```

The string query language supports:

- `AND`
- `OR`
- parentheses
- `AND` precedence over `OR`
- case-insensitive logical operators

Exactly one of `--query` and `--query-string` must be provided.

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
   "duration_ms": 3.1,
   "index_build_duration_ms": 12.8,
   "index_memory_estimate_bytes": 56000000
}
```

Where:

- `duration_ms` – query execution time.
- `index_build_duration_ms` – time required to build and sort the inverted index (only for the `index` method).
- `index_memory_estimate_bytes` – estimated memory occupied by posting-list IDs in the inverted index.

### Result limit

The result contains at most 1000 IDs in `matched_ids`.

- `matched_count` contains the full number of matching events.
- `matched_ids` contains no more than the first 1000 IDs.
- `truncated` is `true` when `matched_count` is greater than 1000.

## Algorithm

1. Read events from a JSONL file.
2. Build an inverted index for indexed fields.
3. Parse the query into a tree.
4. Execute the query using either:
   - full scan;
   - inverted index.
5. Return matching event IDs and execution statistics.

## Scan and index trade-offs

The `scan` and `index` methods produce the same query results, but they have different performance characteristics.

### Scan

The scan method checks every event against the query.

Advantages:

- does not require additional index construction;
- uses less additional memory;
- is suitable for small datasets;
- is useful when only one query needs to be executed.

Disadvantages:

- query execution time grows with the number of events;
- repeated queries require scanning the full dataset again.

### Index

The index method builds posting lists that map field values to event IDs.

Advantages:

- repeated queries are usually faster after the index has been built;
- TERM queries can directly access matching posting lists;
- AND and OR queries operate on sorted event ID lists instead of scanning every event.

Disadvantages:

- building and sorting the index takes additional time;
- the index requires additional memory;
- rebuilding the index may be necessary when the dataset changes.

For a single query over a small dataset, `scan` may be simpler and faster because it avoids index construction. For many queries over the same large dataset, `index` is generally more efficient because the build cost is reused.

### Intersection order

For `AND` queries, nested AND expressions are flattened into a collection of terms.

The index executor obtains the posting list for every term, sorts the posting lists by length, and starts intersection with the shortest posting lists.

Each intersection of two sorted posting lists takes `O(n + m)` time. Starting with the shortest posting lists reduces the size of intermediate results and minimizes unnecessary comparisons.

### Index memory estimate

Each posting list stores event IDs as `uint64` values.

Each `uint64` occupies 8 bytes.

One event may contribute up to seven posting entries because the index contains seven searchable fields.

Therefore, the lower-bound memory estimate for posting-list IDs is:

```text
number of posting entries × 8 bytes
```

For one million fully populated events:

```text
7 × 1,000,000 × 8 bytes ≈ 56 MB
```

This estimate includes only posting-list event IDs.

It does not include Go map overhead, slice headers, string storage, unused slice capacity, or runtime metadata, so the actual memory usage is higher.

## Verification

Format the source code:

```powershell
gofmt -w .
```

Run all tests:

```powershell
go test ./...
```

Run static analysis:

```powershell
go vet ./...
```

Build the CLI:

```powershell
go build -o reverse-index-query.exe ./cmd/reverse-index-query
```

After committing all changes, the repository should be in the following state:

```text
nothing to commit, working tree clean
```

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

## Large dataset scenario

The project includes a reproducible performance scenario for a dataset of **1,000,000 events**.

Run:

```powershell
make large-demo
```

This command performs the following steps:

1. Generates 1,000,000 deterministic events using a fixed seed.
2. Executes the query using the `scan` method.
3. Executes the same query using the `index` method.
4. Compares the results of both methods.

Generated files are stored in:

```text
testdata/large/
```
Expected generated artifacts:

```text
events.jsonl
scan.json
index.json
compare_report.md
```

The comparison report confirms that both search methods produce identical results.

Example compare report:

```text
# Compare report

- Events: 1000000
- Scan matched: 27765
- Index matched: 27765
- Results equal: true
- Scan duration: 20.8141 ms
- Index build duration: 393.6062 ms
- Index query duration: 1.0001 ms
- Index total duration: 394.6063 ms
- Index memory estimate: 56000000 bytes (53.41 MiB)
```

The exact execution times depend on the hardware and Go runtime version, while the result counts and memory estimate remain deterministic for the provided dataset and seed.

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

- Queries can be provided as JSON trees or through `--query-string`.
- Supported operators are TERM, AND and OR.
- NOT queries are not implemented.
- Only exact value matching is supported.
- The inverted index is built entirely in memory.
- Range and wildcard queries are not supported.