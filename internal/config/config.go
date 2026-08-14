package config

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
)

var logLevels = []string{"stdout", "debug", "info", "warn", "error", ""}

type Config struct {
	URL     string `doc:"Адрес сервера"`
	Req     string `doc:"Директория с запросами json"`
	Indent  bool   `doc:"Форматирование ответа"`
	Timeout int    `doc:"Max время для ответа"`
	Workers int    `doc:"Количество параллельных работников"`
	Log     string `doc:"Уровень логирования ('', 'stdout', 'debug', 'info', 'warn', 'error')"`
}

func New() (*Config, error) {
	flags, err := parse()
	if err != nil {
		return &Config{}, err
	}

	if flags.Req == "" {
		return &Config{}, fmt.Errorf("empty requests directory: %s", flags.Req)
	}
	if _, err := os.Stat(flags.Req); os.IsNotExist(err) {
		return &Config{}, fmt.Errorf("no found dir:%s", flags.Req)
	}
	if flags.Timeout <= 0 {
		return &Config{}, fmt.Errorf("timeout=%v must be > 0", flags.Timeout)
	}
	if flags.Workers < 1 {
		return &Config{}, fmt.Errorf("workers=%v must be >= 1", flags.Workers)
	}
	flags.Log = strings.ToLower(flags.Log)
	if !slices.Contains(logLevels, flags.Log) {
		return &Config{}, fmt.Errorf("log=%v must be one of %v", flags.Log, logLevels)
	}

	return flags, nil
}

func parse() (*Config, error) {
	url := flag.String(
		"url",
		"http://localhost:8080/execute",
		"Server address",
	)
	req := flag.String(
		"req",
		"requests",
		"Директория с запросами json",
	)
	indent := flag.Bool(
		"indent",
		false,
		"Format response",
	)
	timeout := flag.Int(
		"timeout",
		10,
		"Max response timeout in seconds",
	)
	workers := flag.Int(
		"workers",
		runtime.NumCPU(),
		"Number of parallel workers",
	)
	log := flag.String(
		"log",
		"",
		"Log level ('stdout', 'debug', 'info', 'warn', 'error', '')",
	)

	flag.Parse()

	return &Config{
		URL:     *url,
		Req:     *req,
		Indent:  *indent,
		Timeout: *timeout,
		Workers: *workers,
		Log:     *log,
	}, nil
}
