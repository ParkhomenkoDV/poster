package config

import (
	"flag"
	"os"
	"testing"
)

// TestParseFlags тестирует парсинг флагов с различными входными данными
func TestParseFlags(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedURL     string
		expectedReqDir  string
		expectedResDir  string
		expectedTimeout int
		shouldFail      bool
	}{
		{
			name:            "все флаги заданы",
			args:            []string{"cmd", "--url", "https://test.com", "--requests", "req", "--responses", "res", "--timeout", "60"},
			expectedURL:     "https://test.com",
			expectedReqDir:  "req",
			expectedResDir:  "res",
			expectedTimeout: 60,
			shouldFail:      false,
		},
		{
			name:            "только обязательные флаги",
			args:            []string{"cmd", "--requests", "req", "--responses", "res"},
			expectedURL:     "http://localhost:8080/execute",
			expectedReqDir:  "req",
			expectedResDir:  "res",
			expectedTimeout: 30,
			shouldFail:      false,
		},
		{
			name:            "пустая директория запросов",
			args:            []string{"cmd", "--requests", ""},
			expectedURL:     "",
			expectedReqDir:  "",
			expectedResDir:  "",
			expectedTimeout: 0,
			shouldFail:      true,
		},
		{
			name:            "пустая директория ответов",
			args:            []string{"cmd", "--responses", ""},
			expectedURL:     "",
			expectedReqDir:  "",
			expectedResDir:  "",
			expectedTimeout: 0,
			shouldFail:      true,
		},
		{
			name:            "некорректный таймаут",
			args:            []string{"cmd", "--timeout", "0"},
			expectedURL:     "",
			expectedReqDir:  "",
			expectedResDir:  "",
			expectedTimeout: 0,
			shouldFail:      true,
		},
		{
			name:            "отрицательный таймаут",
			args:            []string{"cmd", "--timeout", "-1"},
			expectedURL:     "",
			expectedReqDir:  "",
			expectedResDir:  "",
			expectedTimeout: 0,
			shouldFail:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = tt.args
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			flags, err := parse()

			if tt.shouldFail {
				if err == nil {
					t.Error("ожидалась ошибка, но не получена")
				}
				return
			}

			if err != nil {
				t.Fatalf("не ожидалась ошибка, но получена: %v", err)
			}

			if flags.URL != tt.expectedURL {
				t.Errorf("URL = %q, ожидалось %q", flags.URL, tt.expectedURL)
			}

			if flags.RequestsDir != tt.expectedReqDir {
				t.Errorf("RequestsDir = %q, ожидалось %q", flags.RequestsDir, tt.expectedReqDir)
			}

			if flags.ResponsesDir != tt.expectedResDir {
				t.Errorf("ResponsesDir = %q, ожидалось %q", flags.ResponsesDir, tt.expectedResDir)
			}

			if flags.Timeout != tt.expectedTimeout {
				t.Errorf("Timeout = %d, ожидалось %d", flags.Timeout, tt.expectedTimeout)
			}
		})
	}
}

// TestParseFlagOrder тестирует порядок флагов
func TestParseFlagOrder(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Флаги в разном порядке
	os.Args = []string{"cmd", "--timeout", "5", "--responses", "resp", "--requests", "req", "--url", "http://test.com"}

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
}

// TestParseDuplicateFlags тестирует дублирование флагов
func TestParseDuplicateFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Дублирующийся флаг - последнее значение должно использоваться
	os.Args = []string{"cmd", "--url", "first.com", "--url", "second.com"}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags, err := parse()
	if err != nil {
		t.Fatalf("parse() вернул ошибку: %v", err)
	}

	// Должен использоваться последний флаг
	if flags.URL != "second.com" {
		t.Errorf("URL = %q, ожидалось последнее значение %q", flags.URL, "second.com")
	}
}
