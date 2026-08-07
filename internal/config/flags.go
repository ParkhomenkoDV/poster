package config

import (
	"flag"
	"fmt"
	"runtime"
	"slices"
)

const usage = "Usage: go run poster.go [-url=<URL>] [-requests=<dir>] [-responses=<dir>] [-indent] [-timeout=N] [-workers=N] [-log=<level>]"

type Flags struct {
	URL          string `doc:"Адрес сервера"`
	RequestsDir  string `doc:"Директория с запросами json"`
	ResponsesDir string `doc:"Директория с ответами json"`
	Indent       bool   `doc:"Форматирование ответа"`
	Timeout      int    `doc:"Max время для ответа"`
	Workers      int    `doc:"Количество параллельных работников"`
	Log          string `doc:"Уровень логирования"`
}

func parse() (*Flags, error) {
	numCPU := runtime.NumCPU()

	url := flag.String("url", "http://localhost:8080/execute", "Server address")
	requestsDir := flag.String("requests", "requests", "Директория с запросами json")
	responsesDir := flag.String("responses", "responses", "Директория с ответами json")
	indent := flag.Bool("indent", false, "Format response")
	timeout := flag.Int("timeout", 10, "Max response timeout in seconds")
	workers := flag.Int("workers", numCPU, "Number of parallel workers")
	log := flag.String("log", "", "Log level ('', 'stdout', 'debug', 'info', 'warn', 'error')")

	flag.Parse()

	if *requestsDir == "" {
		fmt.Println(usage)
		return &Flags{}, fmt.Errorf("empty requests directory: %s", *requestsDir)
	}
	if *responsesDir == "" {
		fmt.Println(usage)
		return &Flags{}, fmt.Errorf("empty responses directory: %s", *responsesDir)
	}
	if *timeout <= 0 {
		fmt.Println(usage)
		return &Flags{}, fmt.Errorf("timeout=%v must be > 0", *timeout)
	}
	if *workers < 1 || numCPU < *workers {
		fmt.Println(usage)
		return &Flags{}, fmt.Errorf("workers=%v must be in range [1..%v]", *workers, numCPU)
	}
	levels := []string{"", "stdout", "debug", "info", "warn", "error"}
	if !slices.Contains(levels, *log) {
		fmt.Println(usage)
		return &Flags{}, fmt.Errorf("log=%v must be one of %v", *log, levels)
	}

	return &Flags{
		URL:          *url,
		RequestsDir:  *requestsDir,
		ResponsesDir: *responsesDir,
		Indent:       *indent,
		Timeout:      *timeout,
		Workers:      *workers,
		Log:          *log,
	}, nil
}
