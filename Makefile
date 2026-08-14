# Colors
RED    = \033[0;31m
GREEN  = \033[0;32m
YELLOW = \033[0;33m
BLUE   = \033[0;34m
RESET  = \033[0m

# Targets
.PHONY: help vendor test bench prof run build

help:
	@echo "Available commands:"
	@echo "  make vendor         - Make vendor"
	@echo "  make install        - Install dependences"
	@echo "  make format         - Format code (black, isort)"
	@echo "  make lint           - Run linters (flake8, pylint)"
	@echo "  make test           - Run tests"
	@echo "  make cover          - Run coverage"
	@echo "  make bench          - Run benchmarks"
	@echo "  make pprof          - Run profiling"
	@echo "  make doc            - Add documentation"
	@echo "  make run            - Run program"
	@echo "  make build          - Build program"
	@echo "  make clean          - Clean project"

vendor:
	go mod tidy
	go mod vendor

install:
	@echo "$(BLUE)Installing dependencies...$(RESET)"	
	go get -u ./...

format:
	@echo "$(BLUE)Formatting code...$(RESET)"	
	go fmt -s -w .

lint:
	@echo "$(BLUE)Running linters...$(RESET)"	
	go vet ./...

test:
	@echo "$(BLUE)Running tests...$(RESET)"
	go test ./... -cover

cover:
	@echo "$(BLUE)Running coverage...$(RESET)"
	go test ./... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out

bench:
	@echo "$(BLUE)Running benchmarks...$(RESET)"
	go test ./... -bench=. -benchmem -benchtime 1s -count=1

pprof:
	@echo "$(BLUE)Running profiling...$(RESET)"
	go test -benchmem -benchtime 5s -count=5 -cpuprofile=cpu.out -memprofile=mem.out -trace=trace.out

cpu: pprof
	go tool pprof -http=:8081 cpu.out

mem: pprof
	go tool pprof -http=:8082 mem.out

trace: pprof
	go tool trace -http=:8083 poster.go trace.out

race:
	go test -race

doc:
	go doc ./...

run:
	@echo "$(BLUE)Running project...$(RESET)"
	go run poster.go -log=info

build:
	@echo "$(BLUE)Building project...$(RESET)"
	go build -o poster poster.go 

clear:
	@echo "$(BLUE)Cleaning project...$(RESET)"
	find . -type d -name "__pycache__" -exec rm -r {} +
	find . -type d -name ".pytest_cache" -exec rm -r {} +
	find . -type f -name "*.pyc" -delete
	find . -type f -name "*.pyo" -delete
	find . -type f -name "*.out" -delete
	find . -type f -name "*.test" -delete
	rm -rf .coverage htmlcov
	go clean -testcache -modcache