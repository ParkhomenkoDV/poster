package config

import (
	"flag"
	"fmt"
	"runtime"
	"slices"
)

type Flags struct {
	URL     string `doc:"Адрес сервера"`
	Req     string `doc:"Директория с запросами json"`
	Indent  bool   `doc:"Форматирование ответа"`
	Timeout int    `doc:"Max время для ответа"`
	Workers int    `doc:"Количество параллельных работников"`
	Log     string `doc:"Уровень логирования"`
}

var (
	logLevels = []string{"stdout", "debug", "info", "warn", "error", ""}
)

func parse() (*Flags, error) {
	url := flag.String("url", "http://localhost:8080/execute", "Server address")
	req := flag.String("req", "requests", "Директория с запросами json")
	indent := flag.Bool("indent", false, "Format response")
	timeout := flag.Int("timeout", 10, "Max response timeout in seconds")
	workers := flag.Int("workers", runtime.NumCPU(), "Number of parallel workers")
	log := flag.String("log", "", "Log level ('stdout', 'debug', 'info', 'warn', 'error', '')")

	flag.Parse()

	if *req == "" {
		return &Flags{}, fmt.Errorf("empty requests directory: %s", *req)
	}
	if *timeout <= 0 {
		return &Flags{}, fmt.Errorf("timeout=%v must be > 0", *timeout)
	}
	if *workers < 1 {
		return &Flags{}, fmt.Errorf("workers=%v must be >= 1", *workers)
	}
	if !slices.Contains(logLevels, *log) {
		return &Flags{}, fmt.Errorf("log=%v must be one of %v", *log, logLevels)
	}

	return &Flags{
		URL:     *url,
		Req:     *req,
		Indent:  *indent,
		Timeout: *timeout,
		Workers: *workers,
		Log:     *log,
	}, nil
}
