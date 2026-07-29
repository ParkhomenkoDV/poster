package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// generateData создаёт срез байт указанного размера, заполненный повторяющимся паттерном
func generateData(size int64) []byte {
	data := make([]byte, size)
	// Заполняем повторяющимся паттерном для лучшей сжимаемости (не влияет на бенчмарк)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

// BenchmarkReadFile измеряет производительность чтения файлов
// с переиспользованием буфера для разных размеров.
func BenchmarkReadFile(b *testing.B) {
	// Создаём временную директорию для тестовых файлов
	tempDir := b.TempDir()

	// Подготавливаем файлы разных размеров
	sizes := []struct {
		name string
		size int64
	}{
		{"1KB", 1 << 10},     // 1024 байта
		{"100KB", 100 << 10}, // 102400 байт
		{"1MB", 1 << 20},     // 1048576 байт
		{"10MB", 10 << 20},   // 10485760 байт
	}

	files := make(map[string]string) // имя -> путь
	for _, s := range sizes {
		filePath := filepath.Join(tempDir, s.name+".txt")
		if err := os.WriteFile(filePath, generateData(s.size), 0644); err != nil {
			b.Fatalf("создание файла %s: %v", s.name, err)
		}
		files[s.name] = filePath
	}

	// Инициализация буфера (начальный размер 1 КБ, как в примере)
	initialBufCap := 1024

	b.Run("single", func(b *testing.B) {
		for _, s := range sizes {
			filePath := files[s.name]
			b.Run(s.name, func(b *testing.B) {
				// Создаём новый буфер для каждой итерации бенчмарка,
				// чтобы избежать влияния предыдущих вызовов.
				buf := make([]byte, 0, initialBufCap)

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					data, size, err := readFile(filePath, &buf)
					if err != nil {
						b.Fatalf("readFile error: %v", err)
					}
					if int64(len(data)) != size {
						b.Fatalf("размер данных %d не совпадает с размером файла %d", len(data), size)
					}
					// Чтобы не использовать данные, но и не оптимизировать компилятор
					_ = data
				}
			})
		}
	})

	// Дополнительно можно протестировать влияние начального размера буфера
	// (если начальный буфер меньше файла – будет расширение, что добавляет накладные расходы)
	b.Run("buffer-sizes", func(b *testing.B) {
		const fileSize = 1 << 20 // 1 MB
		filePath, ok := files["1MB"]
		if !ok {
			b.Skip("файл 1MB не создан")
		}

		bufferSizes := []int{512, 1024, 4096, 16 * 1024, 64 * 1024, 1024 * 1024}
		for _, capSize := range bufferSizes {
			b.Run(fmt.Sprintf("cap-%d", capSize), func(b *testing.B) {
				buf := make([]byte, 0, capSize)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _, err := readFile(filePath, &buf)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})

	// Параллельный бенчмарк для проверки contention при использовании разных буферов
	b.Run("parallel", func(b *testing.B) {
		const fileSize = 1 << 20 // 1 MB
		filePath, ok := files["1MB"]
		if !ok {
			b.Skip("файл 1MB не создан")
		}

		b.RunParallel(func(pb *testing.PB) {
			// Каждая горутина имеет свой собственный буфер
			buf := make([]byte, 0, 1024)
			for pb.Next() {
				_, _, err := readFile(filePath, &buf)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

// TestHandler возвращает фиксированный JSON-ответ со статусом 200
func testHandler(w http.ResponseWriter, r *http.Request) {
	// Читаем тело (можно игнорировать, но имитируем обработку)
	_, _ = io.ReadAll(r.Body)
	defer r.Body.Close()

	// Отправляем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","message":"benchmark"}`))
}

// generateJSON создаёт JSON-объект указанного размера (приблизительно)
func generateJSON(size int) []byte {
	// Создаём структуру с полем data, содержащим строку нужной длины
	data := make([]byte, size)
	for i := range data {
		data[i] = 'a' // Заполняем 'a'
	}
	obj := map[string]interface{}{
		"id":   12345,
		"data": string(data),
	}
	jsonBytes, _ := json.Marshal(obj)
	return jsonBytes
}

// BenchmarkSendRequest измеряет производительность отправки HTTP-запросов
// с разными размерами полезной нагрузки.
func BenchmarkSendRequest(b *testing.B) {
	// Запускаем тестовый HTTP-сервер
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()

	// Создаём HTTP-клиент с настройками, аналогичными вашему приложению
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     100,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  true,
		},
	}

	// Готовим тестовые данные разных размеров
	smallPayload := []byte(`{"id":1,"name":"test"}`)
	mediumPayload := generateJSON(1024) // ~1KB
	largePayload := generateJSON(10240) // ~10KB

	// Определяем под-бенчмарки
	testCases := []struct {
		name string
		data []byte
	}{
		{"small", smallPayload},
		{"medium", mediumPayload},
		{"large", largePayload},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Прогрев: делаем один запрос, чтобы установить соединение
			_, _, err := sendRequest(client, server.URL, tc.data)
			if err != nil {
				b.Fatalf("прогрев failed: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := sendRequest(client, server.URL, tc.data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark_saveResponse измеряет производительность сохранения ответов в двух режимах: с форматированием JSON и без.
func Benchmark_saveResponse(b *testing.B) {
	dir := b.TempDir() // Временная директория для файлов

	// Подготовка тестового JSON-ответа (структура с вложенностью)
	testData := map[string]interface{}{
		"status": "ok",
		"data": map[string]interface{}{
			"id":    12345,
			"name":  "benchmark-test",
			"items": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			"meta": map[string]string{
				"version": "1.0",
				"env":     "prod",
			},
		},
	}
	response, err := json.Marshal(testData)
	if err != nil {
		b.Fatal(err)
	}

	const fileName = "response.json" // Имя файла

	b.ResetTimer()
	// Бенчмарк с форматированием (indent = true)
	b.Run("indent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := saveResponse(fileName, response, dir, true); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.ResetTimer()
	// Бенчмарк без форматирования (indent = false)
	b.Run("no-indent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := saveResponse(fileName, response, dir, false); err != nil {
				b.Fatal(err)
			}
		}
	})
}
