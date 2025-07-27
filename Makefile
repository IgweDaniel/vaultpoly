SHELL := /bin/bash

.PHONY: all test

all: test

# Run all tests with verbose output
test:
	@if [ "$(VERBOSE)" = "1" ]; then \
		go test ./... -v; \
	else \
		go test ./...; \
	fi

# Run a specific test case by name, e.g.:
#   make test-case name="TestFeeCalculationCases/Basic_P2PKH_to_P2PKH_with_change"
test-case:
	go test ./... -v -run "${name}"

# Docker Compose targets
.PHONY: compose-up compose-down compose-logs compose-build

compose-up:
	@if [ "$(BUILD)" = "1" ]; then \
		docker compose -f docker/compose.yml up --build;  \
	else \
		docker compose -f docker/compose.yml up; \
	fi


compose-down:
	docker compose -f docker/compose.yml down

compose-logs:
	docker compose -f docker/compose.yml logs -f

compose-build:
	docker compose -f docker/compose.yml build