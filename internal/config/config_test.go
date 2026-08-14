package config

import (
	"flag"
	"os"
	"runtime"
	"testing"
)

// TestParseFlags проверяет корректность парсинга флагов (без валидации)
func TestParseFlags(t *testing.T) {
	numCPU := runtime.NumCPU()

	tests := []struct {
		name        string
		args        []string
		wantURL     string
		wantReq     string
		wantIndent  bool
		wantTimeout int
		wantWorkers int
		wantLog     string
	}{
		{
			name: "все флаги заданы",
			args: []string{"cmd",
				"-url", "https://test.com",
				"-req", "req",
				"-timeout", "60",
				"-workers", "4",
				"-log", "debug",
				"-indent",
			},
			wantURL:     "https://test.com",
			wantReq:     "req",
			wantIndent:  true,
			wantTimeout: 60,
			wantWorkers: 4,
			wantLog:     "debug",
		},
		{
			name:        "только обязательный -req",
			args:        []string{"cmd", "-req", "my_requests"},
			wantURL:     "http://localhost:8080/execute",
			wantReq:     "my_requests",
			wantIndent:  false,
			wantTimeout: 10,
			wantWorkers: numCPU,
			wantLog:     "",
		},
		{
			name:        "дефолтные значения (без флагов)",
			args:        []string{"cmd"},
			wantURL:     "http://localhost:8080/execute",
			wantReq:     "requests",
			wantIndent:  false,
			wantTimeout: 10,
			wantWorkers: numCPU,
			wantLog:     "",
		},
		{
			name:        "относительный путь для req",
			args:        []string{"cmd", "-req", "./subdir"},
			wantURL:     "http://localhost:8080/execute",
			wantReq:     "./subdir",
			wantIndent:  false,
			wantTimeout: 10,
			wantWorkers: numCPU,
			wantLog:     "",
		},
		{
			name:        "уровень лога stdout",
			args:        []string{"cmd", "-req", "req", "-log", "stdout"},
			wantURL:     "http://localhost:8080/execute",
			wantReq:     "req",
			wantIndent:  false,
			wantTimeout: 10,
			wantWorkers: numCPU,
			wantLog:     "stdout",
		},
		{
			name:        "уровень лога info",
			args:        []string{"cmd", "-req", "req", "-log", "info"},
			wantURL:     "http://localhost:8080/execute",
			wantReq:     "req",
			wantIndent:  false,
			wantTimeout: 10,
			wantWorkers: numCPU,
			wantLog:     "info",
		},
		{
			name:        "indent=true",
			args:        []string{"cmd", "-req", "req", "-indent"},
			wantURL:     "http://localhost:8080/execute",
			wantReq:     "req",
			wantIndent:  true,
			wantTimeout: 10,
			wantWorkers: numCPU,
			wantLog:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = tt.args
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			flags, err := parse()
			if err != nil {
				t.Fatalf("parse() вернул ошибку: %v", err)
			}

			if flags.URL != tt.wantURL {
				t.Errorf("URL = %q, ожидалось %q", flags.URL, tt.wantURL)
			}
			if flags.Req != tt.wantReq {
				t.Errorf("Req = %q, ожидалось %q", flags.Req, tt.wantReq)
			}
			if flags.Indent != tt.wantIndent {
				t.Errorf("Indent = %v, ожидалось %v", flags.Indent, tt.wantIndent)
			}
			if flags.Timeout != tt.wantTimeout {
				t.Errorf("Timeout = %d, ожидалось %d", flags.Timeout, tt.wantTimeout)
			}
			if flags.Workers != tt.wantWorkers {
				t.Errorf("Workers = %d, ожидалось %d", flags.Workers, tt.wantWorkers)
			}
			if flags.Log != tt.wantLog {
				t.Errorf("Log = %q, ожидалось %q", flags.Log, tt.wantLog)
			}
		})
	}
}

// TestParseFlagOrder проверяет, что порядок флагов не влияет на результат
func TestParseFlagOrder(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "-log", "error", "-timeout", "5", "-workers", "2", "-req", "req", "-url", "http://test.com", "-indent"}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags, err := parse()
	if err != nil {
		t.Fatalf("parse() вернул ошибку: %v", err)
	}

	if flags.URL != "http://test.com" {
		t.Errorf("URL = %q, ожидалось %q", flags.URL, "http://test.com")
	}
	if flags.Req != "req" {
		t.Errorf("Req = %q, ожидалось %q", flags.Req, "req")
	}
	if !flags.Indent {
		t.Error("Indent должен быть true")
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

// TestParseDuplicateFlags проверяет, что последнее значение флага имеет приоритет
func TestParseDuplicateFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "-req", "req", "-url", "first.com", "-timeout", "10", "-workers", "1", "-log", "debug", "-url", "second.com", "-timeout", "20", "-workers", "2", "-log", "info"}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags, err := parse()
	if err != nil {
		t.Fatalf("parse() вернул ошибку: %v", err)
	}

	if flags.URL != "second.com" {
		t.Errorf("URL = %q, ожидалось %q", flags.URL, "second.com")
	}
	if flags.Timeout != 20 {
		t.Errorf("Timeout = %d, ожидалось %d", flags.Timeout, 20)
	}
	if flags.Workers != 2 {
		t.Errorf("Workers = %d, ожидалось %d", flags.Workers, 2)
	}
	if flags.Log != "info" {
		t.Errorf("Log = %q, ожидалось %q", flags.Log, "info")
	}
}

// helper: создаёт временную директорию и возвращает её путь
func createTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

// TestNew_ValidationErrors тестирует ошибки валидации (пустая/несуществующая директория, невалидные значения)
func TestNew_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "пустая директория",
			args:    []string{"cmd", "-req", ""},
			wantErr: true,
		},
		{
			name:    "несуществующая директория",
			args:    []string{"cmd", "-req", "/non/existent/path"},
			wantErr: true,
		},
		{
			name:    "таймаут 0",
			args:    []string{"cmd", "-req", "/tmp", "-timeout", "0"},
			wantErr: true,
		},
		{
			name:    "отрицательный таймаут",
			args:    []string{"cmd", "-req", "/tmp", "-timeout", "-1"},
			wantErr: true,
		},
		{
			name:    "workers 0",
			args:    []string{"cmd", "-req", "/tmp", "-workers", "0"},
			wantErr: true,
		},
		{
			name:    "отрицательные workers",
			args:    []string{"cmd", "-req", "/tmp", "-workers", "-1"},
			wantErr: true,
		},
		{
			name:    "некорректный уровень лога",
			args:    []string{"cmd", "-req", "/tmp", "-log", "invalid"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = test.args
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			cfg, err := New()
			if test.wantErr {
				if err == nil {
					t.Error("ожидалась ошибка, но не получена")
				}
				// при ошибке возвращается пустая структура
				if cfg == nil || cfg.Req != "" || cfg.URL != "" {
					t.Errorf("при ошибке ожидалась пустая структура, получено %+v", cfg)
				}
			} else {
				if err != nil {
					t.Fatalf("не ожидалась ошибка, получена: %v", err)
				}
				// дополнительные проверки при успехе
			}
		})
	}
}

// TestNew_ValidConfig тестирует успешное создание конфигурации с корректными значениями
func TestNew_ValidConfig(t *testing.T) {
	dir := createTempDir(t)

	tests := []struct {
		name        string
		args        []string
		wantURL     string
		wantReq     string
		wantTimeout int
		wantWorkers int
		wantLog     string
	}{
		{
			name:        "все флаги заданы",
			args:        []string{"cmd", "-url", "https://test.com", "-req", dir, "-timeout", "30", "-workers", "2", "-log", "debug"},
			wantURL:     "https://test.com",
			wantReq:     dir,
			wantTimeout: 30,
			wantWorkers: 2,
			wantLog:     "debug",
		},
		{
			name:        "минимальные флаги (только req)",
			args:        []string{"cmd", "-req", dir},
			wantURL:     "http://localhost:8080/execute",
			wantReq:     dir,
			wantTimeout: 10,
			wantWorkers: runtime.NumCPU(),
			wantLog:     "",
		},
		{
			name:        "логирование stdout",
			args:        []string{"cmd", "-req", dir, "-log", "stdout"},
			wantURL:     "http://localhost:8080/execute",
			wantReq:     dir,
			wantTimeout: 10,
			wantWorkers: runtime.NumCPU(),
			wantLog:     "stdout",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = test.args
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			cfg, err := New()
			if err != nil {
				t.Fatalf("не ожидалась ошибка, получена: %v", err)
			}

			if cfg.URL != test.wantURL {
				t.Errorf("URL = %q, ожидалось %q", cfg.URL, test.wantURL)
			}
			if cfg.Req != test.wantReq {
				t.Errorf("Req = %q, ожидалось %q", cfg.Req, test.wantReq)
			}
			if cfg.Timeout != test.wantTimeout {
				t.Errorf("Timeout = %d, ожидалось %d", cfg.Timeout, test.wantTimeout)
			}
			if cfg.Workers != test.wantWorkers {
				t.Errorf("Workers = %d, ожидалось %d", cfg.Workers, test.wantWorkers)
			}
			if cfg.Log != test.wantLog {
				t.Errorf("Log = %q, ожидалось %q", cfg.Log, test.wantLog)
			}
		})
	}
}

// TestNew_WorkersDefaultCPU проверяет, что workers по умолчанию равен runtime.NumCPU()
func TestNew_WorkersDefaultCPU(t *testing.T) {
	dir := createTempDir(t)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "-req", dir}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	cfg, err := New()
	if err != nil {
		t.Fatalf("не ожидалась ошибка: %v", err)
	}

	expected := runtime.NumCPU()
	if cfg.Workers != expected {
		t.Errorf("Workers = %d, ожидалось %d", cfg.Workers, expected)
	}
}
