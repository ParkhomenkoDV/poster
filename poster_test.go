package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSaveResponse_ValidJSON тестирует сохранение валидного JSON
func TestSaveResponse_ValidJSON(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()

	// Подготавливаем тестовый JSON
	testJSON := []byte(`{"name":"test","value":42}`)

	// Вызываем тестируемую функцию
	fileName := "test_response.json"
	err := saveResponse(fileName, testJSON, tempDir)
	if err != nil {
		t.Fatalf("saveResponse вернула ошибку: %v", err)
	}

	// Проверяем что файл создан
	filePath := filepath.Join(tempDir, fileName)
	_, err = os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Не удалось прочитать созданный файл: %v", err)
	}

	// Проверяем права доступа
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Не удалось получить информацию о файле: %v", err)
	}

	expectedPerm := os.FileMode(0644)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("Права доступа файла: %v, ожидалось: %v",
			info.Mode().Perm(), expectedPerm)
	}
}

// TestSaveResponse_InvalidJSON тестирует сохранение невалидного JSON
func TestSaveResponse_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()

	// Невалидный JSON
	invalidJSON := []byte(`{"name": "test", "value": 42,}`) // Лишняя запятая
	expectedContent := `{"name": "test", "value": 42,}`

	fileName := "invalid_response.json"
	err := saveResponse(fileName, invalidJSON, tempDir)
	if err != nil {
		t.Fatalf("saveResponse вернула ошибку для невалидного JSON: %v", err)
	}

	filePath := filepath.Join(tempDir, fileName)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Не удалось прочитать созданный файл: %v", err)
	}

	actualContent := string(content)
	if actualContent != expectedContent {
		t.Errorf("Содержимое файла не совпадает:\nОжидалось:\n%s\nПолучено:\n%s",
			expectedContent, actualContent)
	}
}

// TestSaveResponse_EmptyJSON тестирует сохранение пустого JSON
func TestSaveResponse_EmptyJSON(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name     string
		jsonData []byte
		expected string
	}{
		{
			name:     "пустой объект",
			jsonData: []byte(`{}`),
			expected: "{}",
		},
		{
			name:     "пустой массив",
			jsonData: []byte(`[]`),
			expected: "[]",
		},
		{
			name:     "пустая строка",
			jsonData: []byte(``),
			expected: "",
		},
		{
			name:     "null",
			jsonData: []byte(`null`),
			expected: "null",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileName := "empty_response.json"
			err := saveResponse(fileName, tc.jsonData, tempDir)
			if err != nil {
				t.Fatalf("saveResponse вернула ошибку: %v", err)
			}

			filePath := filepath.Join(tempDir, fileName)
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Не удалось прочитать созданный файл: %v", err)
			}

			// Форматируем ожидаемый результат для сравнения
			var expectedBuffer bytes.Buffer
			if len(tc.jsonData) > 0 {
				if err := json.Indent(&expectedBuffer, tc.jsonData, "", "  "); err != nil {
					expectedBuffer.Write(tc.jsonData)
				}
			}
			expected := expectedBuffer.String()

			actual := string(content)
			if actual != expected {
				t.Errorf("Содержимое файла не совпадает:\nОжидалось:\n%s\nПолучено:\n%s",
					expected, actual)
			}

			// Удаляем файл перед следующим тестом
			os.Remove(filePath)
		})
	}
}

// TestSaveResponse_LargeJSON тестирует сохранение большого JSON
func TestSaveResponse_LargeJSON(t *testing.T) {
	tempDir := t.TempDir()

	// Создаем большой JSON
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	jsonData, err := json.Marshal(largeData)
	if err != nil {
		t.Fatalf("Не удалось создать тестовый JSON: %v", err)
	}

	fileName := "large_response.json"
	err = saveResponse(fileName, jsonData, tempDir)
	if err != nil {
		t.Fatalf("saveResponse вернула ошибку: %v", err)
	}

	// Проверяем что файл создан и не пустой
	filePath := filepath.Join(tempDir, fileName)
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Не удалось получить информацию о файле: %v", err)
	}

	if info.Size() == 0 {
		t.Error("Созданный файл пустой")
	}

	// Проверяем что файл содержит валидный JSON
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Не удалось прочитать файл: %v", err)
	}

	if !json.Valid(content) {
		t.Error("Сохраненный файл не содержит валидный JSON")
	}
}

// TestSaveResponse_PathOperations тестирует различные пути сохранения
func TestSaveResponse_PathOperations(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name       string
		fileName   string
		path       string
		shouldFail bool
	}{
		{
			name:       "обычный путь",
			fileName:   "response.json",
			path:       tempDir,
			shouldFail: false,
		},
		{
			name:       "путь с поддиректорией",
			fileName:   "response.json",
			path:       filepath.Join(tempDir, "subdir"),
			shouldFail: true, // Директория не существует
		},
		{
			name:       "имя файла с пробелами",
			fileName:   "my response.json",
			path:       tempDir,
			shouldFail: false,
		},
		{
			name:       "имя файла с кириллицей",
			fileName:   "ответ.json",
			path:       tempDir,
			shouldFail: false,
		},
		{
			name:       "относительный путь",
			fileName:   "response.json",
			path:       ".",
			shouldFail: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonData := []byte(`{"test": "data"}`)

			err := saveResponse(tc.fileName, jsonData, tc.path)

			if tc.shouldFail {
				if err == nil {
					t.Error("Ожидалась ошибка, но её нет")
				}
				return
			}

			if err != nil {
				t.Fatalf("Неожиданная ошибка: %v", err)
			}

			// Проверяем что файл создан
			filePath := filepath.Join(tc.path, tc.fileName)
			if _, err := os.Stat(filePath); err != nil {
				t.Errorf("Файл не создан: %v", err)
			}

			// Убираем тестовые файлы
			if tc.path == "." {
				os.Remove(tc.fileName)
			}
		})
	}
}

// TestSaveResponse_PermissionDenied тестирует сохранение в защищенную директорию
func TestSaveResponse_PermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Тест пропущен: запущено от root, нет смысла проверять права")
	}

	// Пытаемся сохранить в системную директорию
	systemDir := "/root"
	fileName := "test.json"
	jsonData := []byte(`{"test": "data"}`)

	err := saveResponse(fileName, jsonData, systemDir)
	if err == nil {
		// Если тест проходит под root, это нормально
		if os.Geteuid() == 0 {
			t.Log("Тест выполнен под root, ошибка прав доступа не ожидается")
		} else {
			t.Error("Ожидалась ошибка прав доступа, но её нет")
		}
	}
}

// TestSaveResponse_FileAlreadyExists тестирует перезапись существующего файла
func TestSaveResponse_FileAlreadyExists(t *testing.T) {
	tempDir := t.TempDir()

	// Создаем файл заранее
	fileName := "existing.json"
	existingPath := filepath.Join(tempDir, fileName)
	existingContent := []byte("existing content")

	if err := os.WriteFile(existingPath, existingContent, 0644); err != nil {
		t.Fatalf("Не удалось создать тестовый файл: %v", err)
	}

	// Теперь сохраняем новый JSON поверх существующего файла
	newJSON := []byte(`{"new": "data"}`)
	err := saveResponse(fileName, newJSON, tempDir)
	if err != nil {
		t.Fatalf("saveResponse вернула ошибку: %v", err)
	}

	// Проверяем что файл перезаписан
	content, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("Не удалось прочитать файл: %v", err)
	}

	// Форматируем ожидаемый результат
	var expectedBuffer bytes.Buffer
	if err := json.Indent(&expectedBuffer, newJSON, "", "  "); err != nil {
		expectedBuffer.Write(newJSON)
	}
	expected := expectedBuffer.String()

	if string(content) != expected {
		t.Errorf("Файл не был перезаписан:\nОжидалось:\n%s\nПолучено:\n%s",
			expected, string(content))
	}
}

// TestSaveResponse_SpecialCharacters тестирует специальные символы в JSON
func TestSaveResponse_SpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name     string
		jsonData []byte
		desc     string
	}{
		{
			name:     "unicode символы",
			jsonData: []byte(`{"message": "Привет мир! 🚀"}`),
			desc:     "кириллица и эмодзи",
		},
		{
			name:     "escape последовательности",
			jsonData: []byte(`{"text": "Line1\nLine2\tTab\"Quote\\Backslash"}`),
			desc:     "специальные символы",
		},
		{
			name:     "HTML символы",
			jsonData: []byte(`{"html": "<div>Test &amp; Check</div>"}`),
			desc:     "HTML entities",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileName := "special_chars.json"
			err := saveResponse(fileName, tc.jsonData, tempDir)
			if err != nil {
				t.Fatalf("saveResponse вернула ошибку: %v", err)
			}

			filePath := filepath.Join(tempDir, fileName)
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Не удалось прочитать файл: %v", err)
			}

			// Проверяем что JSON валидный
			if !json.Valid(content) {
				t.Error("Сохраненный JSON невалидный")
			}

			// Проверяем что содержимое корректно
			var decoded map[string]interface{}
			if err := json.Unmarshal(content, &decoded); err != nil {
				t.Errorf("Не удалось распарсить сохраненный JSON: %v", err)
			}

			// Удаляем тестовый файл
			os.Remove(filePath)
		})
	}
}

// TestSaveResponse_NestedJSON тестирует сохранение вложенных структур JSON
func TestSaveResponse_NestedJSON(t *testing.T) {
	tempDir := t.TempDir()

	complexJSON := []byte(`{
		"users": [
			{"id": 1, "name": "Alice", "tags": ["admin", "user"]},
			{"id": 2, "name": "Bob", "tags": ["user"]}
		],
		"metadata": {
			"count": 2,
			"timestamp": "2024-01-01T00:00:00Z"
		}
	}`)

	fileName := "nested.json"
	err := saveResponse(fileName, complexJSON, tempDir)
	if err != nil {
		t.Fatalf("saveResponse вернула ошибку: %v", err)
	}

	filePath := filepath.Join(tempDir, fileName)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Не удалось прочитать файл: %v", err)
	}

	// Проверяем форматирование
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 10 {
		t.Errorf("Ожидалось многострочное форматирование, получено %d строк", len(lines))
	}

	// Проверяем отступы
	for i, line := range lines {
		if i > 0 && i < len(lines)-1 {
			// Проверяем что строки имеют отступы
			if !strings.HasPrefix(line, "  ") && line != "{" && line != "}" && !strings.HasPrefix(line, "    ") {
				t.Errorf("Строка %d не имеет правильных отступов: %s", i, line)
			}
		}
	}
}

// BenchmarkSaveResponse бенчмарк функции saveResponse
func BenchmarkSaveResponse(b *testing.B) {
	tempDir := b.TempDir()

	// Подготавливаем тестовые данные
	jsonData, _ := json.Marshal(map[string]interface{}{
		"field1": "value1",
		"field2": 123,
		"field3": []string{"a", "b", "c"},
		"field4": map[string]interface{}{
			"nested": true,
			"count":  42,
		},
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		fileName := fmt.Sprintf("benchmark_%d.json", i)
		err := saveResponse(fileName, jsonData, tempDir)
		if err != nil {
			b.Fatalf("saveResponse вернула ошибку: %v", err)
		}
	}
}

// TestSaveResponse_Concurrent тестирует конкурентное сохранение
func TestSaveResponse_Concurrent(t *testing.T) {
	tempDir := t.TempDir()

	jsonData := []byte(`{"test": "data"}`)

	// Запускаем несколько горутин
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			fileName := fmt.Sprintf("concurrent_%d.json", index)
			err := saveResponse(fileName, jsonData, tempDir)
			errors <- err
		}(i)
	}

	// Собираем ошибки
	for i := 0; i < 10; i++ {
		err := <-errors
		if err != nil {
			t.Errorf("Ошибка в горутине %d: %v", i, err)
		}
	}

	// Проверяем что все файлы созданы
	for i := 0; i < 10; i++ {
		fileName := fmt.Sprintf("concurrent_%d.json", i)
		filePath := filepath.Join(tempDir, fileName)
		if _, err := os.Stat(filePath); err != nil {
			t.Errorf("Файл %s не создан: %v", fileName, err)
		}
	}
}
