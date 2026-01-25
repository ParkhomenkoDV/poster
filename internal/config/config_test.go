package config

import (
	"flag"
	"os"
	"runtime"
	"strconv"
	"testing"
)

// TestNew_DefaultValues тестирует создание конфигурации с значениями по умолчанию
func TestNew_DefaultValues(t *testing.T) {
	// Сохраняем оригинальные аргументы и восстанавливаем после теста
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Устанавливаем аргументы командной строки
	os.Args = []string{"cmd"}

	// Сбрасываем флаги для чистого состояния
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() вернул ошибку: %v", err)
	}

	expectedURL := "http://localhost:8080/execute"
	if cfg.URL != expectedURL {
		t.Errorf("URL = %q, ожидалось %q", cfg.URL, expectedURL)
	}

	expectedRequestsDir := "requests"
	if cfg.RequestsDir != expectedRequestsDir {
		t.Errorf("RequestsDir = %q, ожидалось %q", cfg.RequestsDir, expectedRequestsDir)
	}

	expectedResponsesDir := "responses"
	if cfg.ResponsesDir != expectedResponsesDir {
		t.Errorf("ResponsesDir = %q, ожидалось %q", cfg.ResponsesDir, expectedResponsesDir)
	}

	expectedTimeout := 30
	if cfg.Timeout != expectedTimeout {
		t.Errorf("Timeout = %d, ожидалось %d", cfg.Timeout, expectedTimeout)
	}

	expectedWorkers := runtime.NumCPU()
	if cfg.Workers != expectedWorkers {
		t.Errorf("Workers = %d, ожидалось %d", cfg.Workers, expectedWorkers)
	}

	expectedLog := ""
	if cfg.Log != expectedLog {
		t.Errorf("Log = %q, ожидалось %q", cfg.Log, expectedLog)
	}
}

// TestNew_CustomValues тестирует создание конфигурации с пользовательскими значениями
func TestNew_CustomValues(t *testing.T) {
	// Сохраняем оригинальные аргументы и восстанавливаем после теста
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Устанавливаем пользовательские аргументы
	os.Args = []string{
		"cmd",
		"--url", "https://api.example.com/v1",
		"--requests", "/path/to/requests",
		"--responses", "/path/to/responses",
		"--timeout", "10",
		"--workers", "4",
		"--log", "debug",
	}

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() вернул ошибку: %v", err)
	}

	expectedURL := "https://api.example.com/v1"
	if cfg.URL != expectedURL {
		t.Errorf("URL = %q, ожидалось %q", cfg.URL, expectedURL)
	}

	expectedRequestsDir := "/path/to/requests"
	if cfg.RequestsDir != expectedRequestsDir {
		t.Errorf("RequestsDir = %q, ожидалось %q", cfg.RequestsDir, expectedRequestsDir)
	}

	expectedResponsesDir := "/path/to/responses"
	if cfg.ResponsesDir != expectedResponsesDir {
		t.Errorf("ResponsesDir = %q, ожидалось %q", cfg.ResponsesDir, expectedResponsesDir)
	}

	expectedTimeout := 10
	if cfg.Timeout != expectedTimeout {
		t.Errorf("Timeout = %d, ожидалось %d", cfg.Timeout, expectedTimeout)
	}

	expectedWorkers := 4
	if cfg.Workers != expectedWorkers {
		t.Errorf("Workers = %d, ожидалось %d", cfg.Workers, expectedWorkers)
	}

	expectedLog := "debug"
	if cfg.Log != expectedLog {
		t.Errorf("Log = %q, ожидалось %q", cfg.Log, expectedLog)
	}
}

// TestNew_EmptyRequestsDir тестирует обработку пустой директории запросов
func TestNew_EmptyRequestsDir(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Пытаемся установить пустую директорию запросов
	os.Args = []string{"cmd", "--requests", ""}

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	cfg, err := New()
	if err == nil {
		t.Error("Ожидалась ошибка для пустой директории запросов, но её нет")
	}

	// Проверяем, что конфигурация не создана (должна вернуться пустая структура при ошибке)
	if cfg != nil && (cfg.RequestsDir != "" || cfg.ResponsesDir != "") {
		t.Errorf("Конфигурация не должна быть создана при ошибке, но получена: %+v", cfg)
	}
}

// TestNew_EmptyResponsesDir тестирует обработку пустой директории ответов
func TestNew_EmptyResponsesDir(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Пытаемся установить пустую директорию ответов
	os.Args = []string{"cmd", "--responses", ""}

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	cfg, err := New()
	if err == nil {
		t.Error("Ожидалась ошибка для пустой директории ответов, но её нет")
	}

	if cfg != nil && (cfg.RequestsDir != "" || cfg.ResponsesDir != "") {
		t.Errorf("Конфигурация не должна быть создана при ошибке, но получена: %+v", cfg)
	}
}

// TestNew_InvalidTimeout тестирует обработку некорректного таймаута
func TestNew_InvalidTimeout(t *testing.T) {
	tests := []struct {
		name      string
		timeout   string
		expectErr bool
	}{
		{"Нулевой таймаут", "0", true},
		{"Отрицательный таймаут", "-1", true},
		{"Положительный таймаут", "1", false},
		{"Большой таймаут", "3600", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = []string{"cmd", "--requests", "req", "--responses", "res", "--timeout", test.timeout}

			// Сбрасываем флаги
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			cfg, err := New()

			if test.expectErr {
				if err == nil {
					t.Error("Ожидалась ошибка для некорректного таймаута")
				}
			} else {
				if err != nil {
					t.Errorf("Не ожидалась ошибка, но получена: %v", err)
				}
				if cfg != nil {
					timeout, _ := strconv.Atoi(test.timeout)
					if cfg.Timeout != timeout {
						t.Errorf("Timeout = %d, ожидалось %d", cfg.Timeout, timeout)
					}
				}
			}
		})
	}
}

// TestNew_InvalidWorkers тестирует обработку некорректного количества workers
func TestNew_InvalidWorkers(t *testing.T) {
	tests := []struct {
		name      string
		workers   string
		expectErr bool
	}{
		{"Нулевое количество workers", "0", true},
		{"Отрицательное количество workers", "-1", true},
		{"Положительное количество workers", "1", false},
		{"Количество workers больше чем CPU", "1000", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = []string{"cmd", "--requests", "req", "--responses", "res", "--workers", test.workers}

			// Сбрасываем флаги
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			_, err := New()

			if test.expectErr {
				if err == nil {
					t.Error("Ожидалась ошибка для некорректного количества workers")
				}
			} else {
				if err != nil {
					t.Errorf("Не ожидалась ошибка, но получена: %v", err)
				}
			}
		})
	}
}

// TestNew_InvalidLogLevel тестирует обработку некорректного уровня логирования
func TestNew_InvalidLogLevel(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "--requests", "req", "--responses", "res", "--log", "invalid_level"}

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	_, err := New()
	if err == nil {
		t.Error("Ожидалась ошибка для некорректного уровня логирования, но её нет")
	}
}

// TestNew_ValidLogLevels тестирует обработку валидных уровней логирования
func TestNew_ValidLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
	}{
		{"Пустой уровень логирования", ""},
		{"Уровень stdout", "stdout"},
		{"Уровень debug", "debug"},
		{"Уровень info", "info"},
		{"Уровень warn", "warn"},
		{"Уровень error", "error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = []string{"cmd", "--requests", "req", "--responses", "res", "--log", test.logLevel}

			// Сбрасываем флаги
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			cfg, err := New()
			if err != nil {
				t.Fatalf("Не ожидалась ошибка для валидного уровня логирования %q: %v", test.logLevel, err)
			}

			if cfg.Log != test.logLevel {
				t.Errorf("Log = %q, ожидалось %q", cfg.Log, test.logLevel)
			}
		})
	}
}

// TestParse_ValidFlags тестирует функцию parse напрямую
func TestParse_ValidFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{
		"cmd",
		"--url", "http://test.com",
		"--requests", "test_requests",
		"--responses", "test_responses",
		"--timeout", "5",
		"--workers", "2",
		"--log", "info",
	}

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags, err := parse()
	if err != nil {
		t.Fatalf("parse() вернул ошибку: %v", err)
	}

	if flags.URL != "http://test.com" {
		t.Errorf("URL = %q, ожидалось %q", flags.URL, "http://test.com")
	}

	if flags.RequestsDir != "test_requests" {
		t.Errorf("RequestsDir = %q, ожидалось %q", flags.RequestsDir, "test_requests")
	}

	if flags.ResponsesDir != "test_responses" {
		t.Errorf("ResponsesDir = %q, ожидалось %q", flags.ResponsesDir, "test_responses")
	}

	if flags.Timeout != 5 {
		t.Errorf("Timeout = %d, ожидалось %d", flags.Timeout, 5)
	}

	if flags.Workers != 2 {
		t.Errorf("Workers = %d, ожидалось %d", flags.Workers, 2)
	}

	if flags.Log != "info" {
		t.Errorf("Log = %q, ожидалось %q", flags.Log, "info")
	}
}

// TestParse_RelativePaths тестирует обработку относительных путей
func TestParse_RelativePaths(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{
		"cmd",
		"--requests", "./my_requests",
		"--responses", "../responses",
	}

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags, err := parse()
	if err != nil {
		t.Fatalf("parse() вернул ошибку: %v", err)
	}

	// Проверяем что относительные пути приняты
	if flags.RequestsDir != "./my_requests" {
		t.Errorf("RequestsDir = %q, ожидалось %q", flags.RequestsDir, "./my_requests")
	}

	if flags.ResponsesDir != "../responses" {
		t.Errorf("ResponsesDir = %q, ожидалось %q", flags.ResponsesDir, "../responses")
	}
}

// TestParse_MissingFlags тестирует отсутствие флагов
func TestParse_MissingFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Только название программы без флагов
	os.Args = []string{"cmd"}

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags, err := parse()
	if err != nil {
		t.Fatalf("parse() не должен возвращать ошибку при отсутствии флагов: %v", err)
	}

	// Проверяем значения по умолчанию
	if flags.URL != "http://localhost:8080/execute" {
		t.Errorf("URL = %q, ожидалось значение по умолчанию", flags.URL)
	}
}

// TestNew_ErrorPropagation тестирует передачу ошибок из parse в New
func TestNew_ErrorPropagation(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Устанавливаем некорректные аргументы
	os.Args = []string{"cmd", "--requests", "", "--responses", ""}

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	cfg, err := New()
	if err == nil {
		t.Error("Ожидалась ошибка, но её нет")
	}

	// Проверяем что возвращается пустая конфигурация при ошибке
	if cfg == nil {
		t.Error("Ожидалась пустая структура Config при ошибке")
	} else if cfg.URL != "" || cfg.RequestsDir != "" || cfg.ResponsesDir != "" {
		t.Errorf("Ожидалась пустая структура Config, но получено: %+v", cfg)
	}
}

// TestNew_WorkersDefaultCPU тестирует дефолтное значение workers как количество CPU
func TestNew_WorkersDefaultCPU(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd"}

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() вернул ошибку: %v", err)
	}

	expectedWorkers := runtime.NumCPU()
	if cfg.Workers != expectedWorkers {
		t.Errorf("Workers = %d, ожидалось количество CPU: %d", cfg.Workers, expectedWorkers)
	}
}
