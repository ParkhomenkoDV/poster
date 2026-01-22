package config

import (
	"flag"
	"fmt"
	"runtime"
	"slices"
)

const usage = "Использование: go run poster.go [--url <URL>] [--requests <имяДиректории>] [--responses <имяДиректории>] [--timeout N] [--workers N]"

type Flags struct {
	URL          string `doc:"Адрес сервера"`
	RequestsDir  string `doc:"Директория с запросами json"`
	ResponsesDir string `doc:"Директория с ответами json"`
	Timeout      int    `doc:"Max время для ответа"`
	Workers      int    `doc:"Количество параллельных работников"`
	Log          string `doc:"Уровень логирования"`
}

func parse() (*Flags, error) {
	numCPU := runtime.NumCPU()

	url := flag.String("url", "http://localhost:8080/execute", "Адрес сервера")
	requestsDir := flag.String("requests", "requests", "Директория с запросами json")
	responsesDir := flag.String("responses", "responses", "Директория с ответами json")
	timeout := flag.Int("timeout", 30, "Max время для ответа")
	workers := flag.Int("workers", numCPU, "Количество параллельных работников")
	log := flag.String("log", "", "Уровень логирования ('', 'stdout', 'debug', 'info', 'warn', 'error')")

	flag.Parse()

	if *requestsDir == "" {
		fmt.Println(usage)
		return &Flags{}, fmt.Errorf("empty requests dir %s", *requestsDir)
	}
	if *responsesDir == "" {
		fmt.Println(usage)
		return &Flags{}, fmt.Errorf("empty responses dir %s", *responsesDir)
	}
	if *timeout <= 0 {
		fmt.Println(usage)
		return &Flags{}, fmt.Errorf("timeout=%v <= 0", *timeout)
	}
	if *workers < 1 || numCPU < *workers {
		fmt.Println(usage)
		return &Flags{}, fmt.Errorf("workers=%v must be in [%v..%v]", *workers, 1, numCPU)
	}
	levels := []string{"", "stdout", "debug", "info", "warn", "error"}
	if !slices.Contains(levels, *log) {
		fmt.Println(usage)
		return &Flags{}, fmt.Errorf("log=%v must be in %v", *log, levels)
	}

	return &Flags{
		URL:          *url,
		RequestsDir:  *requestsDir,
		ResponsesDir: *responsesDir,
		Timeout:      *timeout,
		Workers:      *workers,
		Log:          *log,
	}, nil
}
