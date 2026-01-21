# Colors
RED    = \033[0;31m
GREEN  = \033[0;32m
YELLOW = \033[0;33m
BLUE   = \033[0;34m
RESET  = \033[0m

# Targets
.PHONY: help vendor test bench prof run build

vendor:
	go mod vendor

test:
	@echo "$(BLUE)Running tests...$(RESET)"
	go test ./...

bench:
	@echo "$(BLUE)Running benchmarks...$(RESET)"
	go test -benchmem -benchtime 5s -count=5

prof:
	@echo "$(BLUE)Running profiling...$(RESET)"
	go test -benchmem -benchtime 5s -count=5 -cpuprofile=cpu.out -memprofile=mem.out

run:
	@echo "$(BLUE)Running project...$(RESET)"
	go run poster.go

build:
	@echo "$(BLUE)Building project...$(RESET)"
	go build