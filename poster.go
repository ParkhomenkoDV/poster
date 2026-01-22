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
	"sync"
	"time"
)

// Result содержит результат обработки файла
type Result struct {
	FileName string
	Err      error
}

func main() {
	cfg, err := config.New()
	if err != nil {
		fmt.Printf("Ошибка конфигурации %+v: %v", cfg, err)
		return
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

	// Ограничиваем количество одновременных горутин
	if len(filePaths) < cfg.Workers {
		cfg.Workers = len(filePaths)
	}

	// Каналы для работы
	filesChan := make(chan string, len(filePaths))
	resultsChan := make(chan Result, len(filePaths))

	// Создание HTTP клиента с таймаутом
	client := &http.Client{
		Timeout: time.Duration(cfg.Timeout) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.Workers * 10,
			MaxIdleConnsPerHost: cfg.Workers * 10,
			MaxConnsPerHost:     cfg.Workers * 20,
			IdleConnTimeout:     time.Duration(cfg.Timeout*3) * time.Second, // Таймаут на неактивные соединения
		},
	}

	// Запускаем воркеров
	var wg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go work(client, cfg.URL, cfg.ResponsesDir, filesChan, resultsChan, &wg)
	}

	// Отправляем задачи в канал
	for _, filePath := range filePaths {
		filesChan <- filePath
	}
	close(filesChan)

	// Ждем завершения воркеров
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Собираем результаты
	successCount, errorCount := 0, 0
	for result := range resultsChan {
		if result.Err != nil {
			errorCount++
			fmt.Printf("Ошибка обработки файла %s: %v\n", result.FileName, result.Err)
		} else {
			successCount++
		}
	}
	fmt.Printf("\nОбработка завершена! Успешно: %d, Ошибок: %d\n", successCount, errorCount)
}

// work обрабатывает файлы из канала
func work(client *http.Client, url, responsesDir string,
	filesChan <-chan string, resultsChan chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for filePath := range filesChan {
		fileName := filepath.Base(filePath)

		// Чтение JSON файла
		jsonData, err := os.ReadFile(filePath)
		if err != nil {
			resultsChan <- Result{FileName: fileName, Err: fmt.Errorf("чтение файла: %v", err)}
			continue
		}

		// Проверка валидности JSON
		if !json.Valid(jsonData) {
			resultsChan <- Result{FileName: fileName, Err: fmt.Errorf("невалидный JSON")}
			continue
		}

		// Отправка запроса на сервер
		response, err := sendRequest(client, url, jsonData)
		if err != nil {
			resultsChan <- Result{FileName: fileName, Err: fmt.Errorf("отправка запроса: %v", err)}
			continue
		}

		// Сохранение ответа
		if err := saveResponse(fileName, response, responsesDir); err != nil {
			resultsChan <- Result{FileName: fileName, Err: fmt.Errorf("сохранение ответа: %v", err)}
			continue
		}

		resultsChan <- Result{FileName: fileName, Err: nil}
	}
}

// sendRequest отправляет JSON на сервер (без изменений)
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

	return os.WriteFile(path+"/"+fileName, formattedJSON.Bytes(), 0644)
}
