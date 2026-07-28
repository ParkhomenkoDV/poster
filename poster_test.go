package main

import (
	"encoding/json"
	"testing"
)

// BenchmarkSaveResponse измеряет производительность сохранения ответов в двух режимах: с форматированием JSON и без.
func BenchmarkSaveResponse(b *testing.B) {
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
