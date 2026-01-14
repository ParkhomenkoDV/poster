package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"poster/internal/config"
	"time"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	// Проверка наличия директории с запросами
	if _, err := os.Stat(cfg.RequestsDir); os.IsNotExist(err) {
		fmt.Printf("Директория '%s' не существует\n", cfg.RequestsDir)
		return
	}

	// Создание директории для ответов, если её нет
	if err := os.MkdirAll(cfg.ResponsesDir, 0755); err != nil {
		fmt.Printf("Ошибка создания директории '%v': %v\n", cfg.ResponsesDir, err)
		return
	}

	// Чтение всех запросов
	filePaths, err := filepath.Glob(cfg.RequestsDir + "/*.json")
	if err != nil {
		fmt.Printf("Ошибка чтения директории %s: %v\n", cfg.RequestsDir, err)
		return
	}

	if len(filePaths) == 0 {
		fmt.Println("В папке requests не найдено JSON файлов")
		return
	}
	fmt.Printf("Найдено %d JSON файлов для отправки\n", len(filePaths))

	// Создание HTTP клиента с таймаутом
	client := &http.Client{
		Timeout: time.Duration(cfg.Timeout) * time.Second,
	}

	// Обработка каждого файла
	for _, filePath := range filePaths {
		fileName := filepath.Base(filePath)

		// Чтение JSON файла
		jsonData, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Ошибка чтения файла %s: %v\n", fileName, err)
			continue
		}

		// Проверка валидности JSON
		if !json.Valid(jsonData) {
			fmt.Printf("Файл %s содержит невалидный JSON\n", fileName)
			continue
		}

		// Отправка запроса на сервер
		response, err := sendRequest(client, cfg.URL, jsonData)
		if err != nil {
			fmt.Printf("Ошибка отправки запроса: %v\n", err)
			continue
		}

		// Сохранение ответа
		if err := saveResponse(fileName, response, cfg.ResponsesDir); err != nil {
			fmt.Printf("Ошибка сохранения ответа: %v\n", err)
			continue
		}
	}

	fmt.Println("Обработка завершена!")
}

// sendRequest отправляет JSON на сервер
func sendRequest(client *http.Client, url string, jsonData []byte) ([]byte, error) {
	// Создание POST запроса
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// Установка заголовков
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Выполнение запроса
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Чтение ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Проверка статуса ответа
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, fmt.Errorf("сервер вернул статус: %d", resp.StatusCode)
	}

	return body, nil
}

// saveResponse сохраняет ответ директорию
func saveResponse(fileName string, response []byte, path string) error {
	// Форматирование JSON для красивого вывода
	var formattedJSON bytes.Buffer
	if err := json.Indent(&formattedJSON, response, "", "  "); err != nil {
		formattedJSON.Write(response) // Если JSON невалидный, сохраняем как есть
	}

	// Сохранение в файл
	return os.WriteFile(path+"/"+fileName, formattedJSON.Bytes(), 0644)
}
