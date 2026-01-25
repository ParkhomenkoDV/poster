package config

import (
	"flag"
	"os"
	"runtime"
	"testing"
)

// TestParseFlags тестирует парсинг флагов с различными входными данными
func TestParseFlags(t *testing.T) {
	numCPU := runtime.NumCPU()

	tests := []struct {
		name        string
		args        []string
		wantURL     string
		wantReqDir  string
		wantResDir  string
		wantTimeout int
		wantWorkers int
		wantLog     string
		shouldFail  bool
	}{
		{
			name:        "все флаги заданы",
			args:        []string{"cmd", "--url", "https://test.com", "--requests", "req", "--responses", "res", "--timeout", "60", "--workers", "4", "--log", "debug"},
			wantURL:     "https://test.com",
			wantReqDir:  "req",
			wantResDir:  "res",
			wantTimeout: 60,
			wantWorkers: 4,
			wantLog:     "debug",
			shouldFail:  false,
		}, {
			name:        "только обязательные флаги",
			args:        []string{"cmd", "--requests", "req", "--responses", "res"},
			wantURL:     "http://localhost:8080/execute",
			wantReqDir:  "req",
			wantResDir:  "res",
			wantTimeout: 30,
			wantWorkers: numCPU,
			wantLog:     "",
			shouldFail:  false,
		}, {
			name:        "дефолтные значения",
			args:        []string{"cmd"},
			wantURL:     "http://localhost:8080/execute",
			wantReqDir:  "requests",
			wantResDir:  "responses",
			wantTimeout: 30,
			wantWorkers: numCPU,
			wantLog:     "",
			shouldFail:  false,
		}, {
			name:        "пустая директория запросов",
			args:        []string{"cmd", "--requests", ""},
			wantURL:     "",
			wantReqDir:  "",
			wantResDir:  "",
			wantTimeout: 0,
			wantWorkers: 0,
			wantLog:     "",
			shouldFail:  true,
		}, {
			name:        "пустая директория ответов",
			args:        []string{"cmd", "--responses", ""},
			wantURL:     "",
			wantReqDir:  "",
			wantResDir:  "",
			wantTimeout: 0,
			wantWorkers: 0,
			wantLog:     "",
			shouldFail:  true,
		}, {
			name:        "некорректный таймаут (0)",
			args:        []string{"cmd", "--timeout", "0"},
			wantURL:     "",
			wantReqDir:  "",
			wantResDir:  "",
			wantTimeout: 0,
			wantWorkers: 0,
			wantLog:     "",
			shouldFail:  true,
		}, {
			name:        "отрицательный таймаут",
			args:        []string{"cmd", "--timeout", "-1"},
			wantURL:     "",
			wantReqDir:  "",
			wantResDir:  "",
			wantTimeout: 0,
			wantWorkers: 0,
			wantLog:     "",
			shouldFail:  true,
		}, {
			name:        "workers больше чем CPU",
			args:        []string{"cmd", "--requests", "req", "--responses", "res", "--workers", "1000"},
			wantURL:     "",
			wantReqDir:  "",
			wantResDir:  "",
			wantTimeout: 0,
			wantWorkers: 0,
			wantLog:     "",
			shouldFail:  true,
		}, {
			name:        "workers меньше 1",
			args:        []string{"cmd", "--requests", "req", "--responses", "res", "--workers", "0"},
			wantURL:     "",
			wantReqDir:  "",
			wantResDir:  "",
			wantTimeout: 0,
			wantWorkers: 0,
			wantLog:     "",
			shouldFail:  true,
		}, {
			name:        "некорректный уровень логирования",
			args:        []string{"cmd", "--requests", "req", "--responses", "res", "--log", "invalid"},
			wantURL:     "",
			wantReqDir:  "",
			wantResDir:  "",
			wantTimeout: 0,
			wantWorkers: 0,
			wantLog:     "",
			shouldFail:  true,
		},
		{
			name:        "валидные уровни логирования - stdout",
			args:        []string{"cmd", "--requests", "req", "--responses", "res", "--log", "stdout"},
			wantURL:     "http://localhost:8080/execute",
			wantReqDir:  "req",
			wantResDir:  "res",
			wantTimeout: 30,
			wantWorkers: numCPU,
			wantLog:     "stdout",
			shouldFail:  false,
		}, {
			name:        "валидные уровни логирования - info",
			args:        []string{"cmd", "--requests", "req", "--responses", "res", "--log", "info"},
			wantURL:     "http://localhost:8080/execute",
			wantReqDir:  "req",
			wantResDir:  "res",
			wantTimeout: 30,
			wantWorkers: numCPU,
			wantLog:     "info",
			shouldFail:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = test.args
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			flags, err := parse()

			if test.shouldFail {
				if err == nil {
					t.Error("ожидалась ошибка, но не получена")
				}
				return
			}

			if err != nil {
				t.Fatalf("не ожидалась ошибка, но получена: %v", err)
			}

			if flags.URL != test.wantURL {
				t.Errorf("URL = %q, ожидалось %q", flags.URL, test.wantURL)
			}

			if flags.RequestsDir != test.wantReqDir {
				t.Errorf("RequestsDir = %q, ожидалось %q", flags.RequestsDir, test.wantReqDir)
			}

			if flags.ResponsesDir != test.wantResDir {
				t.Errorf("ResponsesDir = %q, ожидалось %q", flags.ResponsesDir, test.wantResDir)
			}

			if flags.Timeout != test.wantTimeout {
				t.Errorf("Timeout = %d, ожидалось %d", flags.Timeout, test.wantTimeout)
			}

			if flags.Workers != test.wantWorkers {
				t.Errorf("Workers = %d, ожидалось %d", flags.Workers, test.wantWorkers)
			}

			if flags.Log != test.wantLog {
				t.Errorf("Log = %q, ожидалось %q", flags.Log, test.wantLog)
			}
		})
	}
}

// TestParseFlagOrder тестирует порядок флагов
func TestParseFlagOrder(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Флаги в разном порядке
	os.Args = []string{"cmd", "--log", "error", "--timeout", "5", "--responses", "resp", "--workers", "2", "--requests", "req", "--url", "http://test.com"}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags, err := parse()
	if err != nil {
		t.Fatalf("parse() вернул ошибку: %v", err)
	}

	if flags.URL != "http://test.com" {
		t.Errorf("URL = %q, ожидалось %q", flags.URL, "http://test.com")
	}

	if flags.RequestsDir != "req" {
		t.Errorf("RequestsDir = %q, ожидалось %q", flags.RequestsDir, "req")
	}

	if flags.ResponsesDir != "resp" {
		t.Errorf("ResponsesDir = %q, ожидалось %q", flags.ResponsesDir, "resp")
	}

	if flags.Timeout != 5 {
		t.Errorf("Timeout = %d, ожидалось %d", flags.Timeout, 5)
	}

	if flags.Workers != 2 {
		t.Errorf("Workers = %d, ожидалось %d", flags.Workers, 2)
	}

	if flags.Log != "error" {
		t.Errorf("Log = %q, ожидалось %q", flags.Log, "error")
	}
}

// TestParseDuplicateFlags тестирует дублирование флагов
func TestParseDuplicateFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Дублирующийся флаг - последнее значение должно использоваться
	os.Args = []string{"cmd", "--requests", "req", "--responses", "res", "--url", "first.com", "--timeout", "10", "--workers", "1", "--log", "debug", "--url", "second.com", "--timeout", "20", "--workers", "2", "--log", "info"}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags, err := parse()
	if err != nil {
		t.Fatalf("parse() вернул ошибку: %v", err)
	}

	// Должны использоваться последние значения
	if flags.URL != "second.com" {
		t.Errorf("URL = %q, ожидалось последнее значение %q", flags.URL, "second.com")
	}

	if flags.Timeout != 20 {
		t.Errorf("Timeout = %d, ожидалось последнее значение %d", flags.Timeout, 20)
	}

	if flags.Workers != 2 {
		t.Errorf("Workers = %d, ожидалось последнее значение %d", flags.Workers, 2)
	}

	if flags.Log != "info" {
		t.Errorf("Log = %q, ожидалось последнее значение %q", flags.Log, "info")
	}
}

// TestParseWorkersRange тестирует граничные значения workers
func TestParseWorkersRange(t *testing.T) {
	numCPU := runtime.NumCPU()

	tests := []struct {
		name       string
		args       []string
		want       int
		shouldFail bool
	}{
		{
			name:       "workers = 1 (минимум)",
			args:       []string{"cmd", "--requests", "req", "--responses", "res", "--workers", "1"},
			want:       1,
			shouldFail: false,
		}, {
			name:       "workers = numCPU (максимум)",
			args:       []string{"cmd", "--requests", "req", "--responses", "res", "--workers", string(rune(numCPU))},
			want:       numCPU,
			shouldFail: false,
		}, {
			name:       "workers = numCPU + 1 (больше максимума)",
			args:       []string{"cmd", "--requests", "req", "--responses", "res", "--workers", string(rune(numCPU + 1))},
			want:       0,
			shouldFail: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = test.args
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			flags, err := parse()

			if test.shouldFail {
				if err == nil {
					t.Error("ожидалась ошибка, но не получена")
				}
				return
			}

			if err != nil {
				t.Fatalf("не ожидалась ошибка, но получена: %v", err)
			}

			if flags.Workers != test.want {
				t.Errorf("Workers = %d, ожидалось %d", flags.Workers, test.want)
			}
		})
	}
}
