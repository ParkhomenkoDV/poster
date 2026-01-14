package config

import (
	"flag"
	"fmt"
)

const usage = "Использование: go run poster.go [--url <URL>] [--requests <имяДиректории>] [--responses <имяДиректории>] [--timeout N]"

type Flags struct {
	URL          string `doc:"Адрес сервера"`
	RequestsDir  string `doc:"Директория с запросами json"`
	ResponsesDir string `doc:"Директория с ответами json"`
	Timeout      int    `doc:"Max время для ответа"`
}

func parse() (*Flags, error) {
	url := flag.String("url", "http://localhost:8080/execute", "Адрес сервера")
	requestsDir := flag.String("requests", "requests", "Директория с запросами json")
	responsesDir := flag.String("responses", "responses", "Директория с ответами json")
	timeout := flag.Int("timeout", 3, "Max время для ответа")
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
	}

	return &Flags{
		URL:          *url,
		RequestsDir:  *requestsDir,
		ResponsesDir: *responsesDir,
		Timeout:      *timeout,
	}, nil
}
