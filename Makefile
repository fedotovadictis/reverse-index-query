APP := reverse-index-query
CMD := ./cmd/reverse-index-query

CONTROL_DIR := testdata/control
EVENTS := $(CONTROL_DIR)/events.jsonl
QUERY := $(CONTROL_DIR)/query.json

SCAN_RESULT := $(CONTROL_DIR)/scan_result.json
INDEX_RESULT := $(CONTROL_DIR)/index_result.json
COMPARE_REPORT := $(CONTROL_DIR)/compare_report.md
LARGE_DIR := testdata/large
LARGE_EVENTS := $(LARGE_DIR)/events_1000000.jsonl
LARGE_SCAN_RESULT := $(LARGE_DIR)/scan_result.json
LARGE_INDEX_RESULT := $(LARGE_DIR)/index_result.json
LARGE_COMPARE_REPORT := $(LARGE_DIR)/compare_report.md

.PHONY: build test bench demo large-demo clean

build:
	go build -o $(APP) $(CMD)

test:
	go test ./...

bench:
	go test -bench=. -benchmem ./...

demo:
	go run $(CMD) run \
		--events $(EVENTS) \
		--query $(QUERY) \
		--method scan \
		--out $(SCAN_RESULT)
	go run $(CMD) run \
		--events $(EVENTS) \
		--query $(QUERY) \
		--method index \
		--out $(INDEX_RESULT)
	go run $(CMD) compare \
		--events $(EVENTS) \
		--query $(QUERY) \
		--out $(COMPARE_REPORT)

large-demo:
	go run $(CMD) generate \
		--count 1000000 \
		--out $(LARGE_EVENTS) \
		--seed 42
	go run $(CMD) run \
		--events $(LARGE_EVENTS) \
		--query $(QUERY) \
		--method scan \
		--out $(LARGE_SCAN_RESULT)
	go run $(CMD) run \
		--events $(LARGE_EVENTS) \
		--query $(QUERY) \
		--method index \
		--out $(LARGE_INDEX_RESULT)
	go run $(CMD) compare \
		--events $(LARGE_EVENTS) \
		--query $(QUERY) \
		--out $(LARGE_COMPARE_REPORT)

clean:
	go clean
	rm -f $(APP) $(APP).exe
	rm -f $(SCAN_RESULT) $(INDEX_RESULT) $(COMPARE_REPORT)
	rm -rf $(LARGE_DIR)
