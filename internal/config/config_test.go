package config

import (
	"flag"
	"os"
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

	expectedTimeout := 3
	if cfg.Timeout != expectedTimeout {
		t.Errorf("Timeout = %d, ожидалось %d", cfg.Timeout, expectedTimeout)
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

	// Проверяем, что конфигурация всё равно создана с дефолтными значениями
	// (в зависимости от реализации вашей функции parse)
	if cfg != nil {
		t.Logf("Конфигурация создана, но ожидалась ошибка: %+v", cfg)
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

	if cfg != nil {
		t.Logf("Конфигурация создана, но ожидалась ошибка: %+v", cfg)
	}
}

// TestNew_InvalidTimeout тестирует обработку некорректного таймаута
func TestNew_InvalidTimeout(t *testing.T) {
	testCases := []struct {
		name      string
		timeout   string
		expectErr bool
	}{
		{"Нулевой таймаут", "0", true},
		{"Отрицательный таймаут", "-1", true},
		{"Положительный таймаут", "1", false},
		{"Большой таймаут", "3600", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = []string{"cmd", "--timeout", tc.timeout}

			// Сбрасываем флаги
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			cfg, err := New()

			if tc.expectErr {
				if err == nil {
					t.Error("Ожидалась ошибка для некорректного таймаута")
				}
			} else {
				if err != nil {
					t.Errorf("Не ожидалась ошибка, но получена: %v", err)
				}
				timeout, err := strconv.Atoi(tc.timeout)
				if cfg != nil && cfg.Timeout != timeout || err != nil {
					t.Errorf("Timeout = %d, ожидалось %s", cfg.Timeout, tc.timeout)
				}
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
