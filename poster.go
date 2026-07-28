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
	"poster/internal/logger"
	"sync"
	"sync/atomic"
	"time"
)

// Result содержит результат обработки файла
type Result struct {
	Err          error         `doc:"Ошибка"`
	FileName     string        `doc:"Имя файла"`
	FileSize     int64         `doc:"Размер файла"`
	RequestSize  int           `doc:"Размер JSON запроса"`
	ResponseSize int           `doc:"Размер JSON ответа"`
	Duration     time.Duration `doc:"Время обработки"`
	StatusCode   int           `doc:"HTTP статус код"`
}

func main() {
	cfg, err := config.New()
	if err != nil {
		fmt.Printf("Ошибка конфигурации %+v: %v", cfg, err)
		return
	}

	// Создание логгера
	mainLogger, err := logger.New(cfg.Log, "log.json")
	if err != nil {
		fmt.Printf("Ошибка инициализации логгера: %v\n", err)
		return
	}
	defer mainLogger.Info("Приложение завершено")

	// Добавляем поля по умолчанию
	mainLogger = mainLogger.WithFields(map[string]interface{}{
		"app":       "poster",
		"pid":       os.Getpid(),
		"timestamp": time.Now().Format(time.RFC3339),
	})

	mainLogger.Info("Логгер инициализирован", map[string]interface{}{
		"level": cfg.Log,
		"file":  "log.json",
	})

	mainLogger.Info("Запуск приложения", map[string]interface{}{
		"config": map[string]interface{}{
			"url":           cfg.URL,
			"requests_dir":  cfg.RequestsDir,
			"responses_dir": cfg.ResponsesDir,
			"timeout":       cfg.Timeout,
			"workers":       cfg.Workers,
			"level":         cfg.Log,
			"file":          "log.json",
		},
	})

	// Проверка наличия директории с запросами
	if _, err := os.Stat(cfg.RequestsDir); os.IsNotExist(err) {
		mainLogger.Fatal("Директория с запросами не существует", map[string]interface{}{
			"directory": cfg.RequestsDir,
		})
	}

	// Создание директории для ответов, если её нет
	if err := os.MkdirAll(cfg.ResponsesDir, 0755); err != nil {
		mainLogger.Fatal("Ошибка создания директории для ответов", map[string]interface{}{
			"directory": cfg.ResponsesDir,
			"error":     err.Error(),
		})
	}

	// Получение списка файлов
	fileDirs, err := filepath.Glob(cfg.RequestsDir + "/*.json")
	if err != nil {
		mainLogger.Fatal("Ошибка чтения директории с запросами", map[string]interface{}{
			"directory": cfg.RequestsDir,
			"error":     err.Error(),
		})
	}

	if len(fileDirs) == 0 {
		mainLogger.Info("В папке requests не найдено JSON файлов")
		return
	}
	mainLogger.Info("Найдены файлы для отправки", map[string]interface{}{
		"count":   len(fileDirs),
		"workers": cfg.Workers,
	})

	// Создание HTTP клиента с пулом соединений
	client := &http.Client{
		Timeout: time.Duration(cfg.Timeout) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100, // Максимальное общее количество "бездействующих" (idle) соединений в пуле ко всем хостам.
			MaxIdleConnsPerHost: 100, // Максимальное количество idle-соединений к одному конкретному хосту.
			MaxConnsPerHost:     100, // Максимальное общее количество соединений к одному хосту (idle + active).

			IdleConnTimeout: 90 * time.Second, // Таймаут на неактивные соединения
			// DisableCompression: true, // Отключение сжатия
		},
	}

	// Запускаем обработку
	startTime := time.Now()
	results := post(cfg, fileDirs, client, mainLogger)
	totalDuration := time.Since(startTime)

	// Считаем статистику
	successCount, errorCount := 0, 0
	for _, result := range results {
		if result.Err != nil {
			errorCount++
			fmt.Printf("❌ %s: %v\n", result.FileName, result.Err)
		} else {
			successCount++
			fmt.Printf("✅ %s: %d [%dms]\n",
				result.FileName,
				result.StatusCode,
				result.Duration.Milliseconds())
		}
	}

	fmt.Println()
	fmt.Printf("📊 ИТОГО: Успешно %d | Ошибок %d | Всего %d\n", successCount, errorCount, len(fileDirs))
	fmt.Printf("⏱️  Общее время: %v\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("⚡ Скорость: %.2f файлов/сек\n", float64(len(fileDirs))/totalDuration.Seconds())
}

// post запускает конвейерную обработку
func post(
	cfg *config.Config,
	fileDirs []string,
	client *http.Client,
	log *logger.Logger,
) []Result {
	taskChan := make(chan string, len(fileDirs))   // канал задач
	resultChan := make(chan Result, len(fileDirs)) // Канал результатов

	// Заполняем очередь задач
	for _, dir := range fileDirs {
		taskChan <- dir
	}
	close(taskChan) // Закрываем смену

	var wg sync.WaitGroup // Счётчик рабочих

	// Атомарная статистика
	var (
		totalRequests   int64
		totalSuccess    int64
		totalErrors     int64
		totalBytesSent  int64
		totalBytesRecv  int64
		totalDurationNs int64
	)

	// Запускаем рабочих
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go work(cfg, i, client, cfg.URL, cfg.ResponsesDir,
			taskChan, resultChan, &wg, log,
			&totalRequests, &totalSuccess, &totalErrors,
			&totalBytesSent, &totalBytesRecv,
			&totalDurationNs)
	}

	// Горутина прогресса
	progressDone := make(chan struct{})
	go showProgress(&totalRequests, &totalSuccess, &totalErrors, len(fileDirs), progressDone)

	// Ждём окончания смены
	go func() {
		wg.Wait()
		close(resultChan)
		close(progressDone)
	}()

	// Собираем результаты
	results := make([]Result, 0, len(fileDirs))
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// work - конвейерный обработчик
func work(
	cfg *config.Config,
	id int,
	client *http.Client,
	url string,
	responsesDir string,
	taskChan <-chan string,
	resultChan chan<- Result,
	wg *sync.WaitGroup,
	log *logger.Logger,
	totalRequests, totalSuccess, totalErrors *int64,
	totalBytesSent, totalBytesRecv, totalDurationNs *int64,
) {
	defer wg.Done()

	// Локальный буфер для чтения файлов
	buf := make([]byte, 0, 1024*1024) // 1MB начальный размер

	for fileDir := range taskChan {
		fileName := filepath.Base(fileDir)
		startTime := time.Now()

		// Читаем файл
		jsonData, fileSize, err := readFile(fileDir, &buf)
		if err != nil {
			resultChan <- Result{
				FileName: fileName,
				Duration: time.Since(startTime),
				Err:      err,
			}
			atomic.AddInt64(totalErrors, 1)
			atomic.AddInt64(totalRequests, 1)
			continue
		}

		// Валидируем JSON
		if !json.Valid(jsonData) {
			resultChan <- Result{
				FileName:    fileName,
				FileSize:    fileSize,
				RequestSize: len(jsonData),
				Duration:    time.Since(startTime),
				Err:         fmt.Errorf("невалидный JSON"),
			}
			atomic.AddInt64(totalErrors, 1)
			atomic.AddInt64(totalRequests, 1)
			continue
		}

		// Отправляем HTTP запрос
		response, statusCode, err := sendRequest(client, url, jsonData)
		if err != nil {
			resultChan <- Result{
				FileName:    fileName,
				FileSize:    fileSize,
				RequestSize: len(jsonData),
				Duration:    time.Since(startTime),
				StatusCode:  statusCode,
				Err:         fmt.Errorf("отправка: %v", err),
			}
			atomic.AddInt64(totalErrors, 1)
			atomic.AddInt64(totalRequests, 1)
			continue
		}

		// Сохраняем ответ
		err = saveResponse(fileName, response, responsesDir, cfg.Indent)
		totalDuration := time.Since(startTime)

		if err != nil {
			resultChan <- Result{
				FileName:     fileName,
				FileSize:     fileSize,
				RequestSize:  len(jsonData),
				ResponseSize: len(response),
				Duration:     totalDuration,
				StatusCode:   statusCode,
				Err:          fmt.Errorf("сохранение: %v", err),
			}
			atomic.AddInt64(totalErrors, 1)
		} else {
			resultChan <- Result{
				FileName:     fileName,
				FileSize:     fileSize,
				RequestSize:  len(jsonData),
				ResponseSize: len(response),
				Duration:     totalDuration,
				StatusCode:   statusCode,
				Err:          nil,
			}
			atomic.AddInt64(totalSuccess, 1)
		}

		// Обновляем статистику
		atomic.AddInt64(totalRequests, 1)
		atomic.AddInt64(totalBytesSent, int64(len(jsonData)))
		atomic.AddInt64(totalBytesRecv, int64(len(response)))
		atomic.AddInt64(totalDurationNs, int64(totalDuration))
	}
}

// readFile читает файл, переиспользуя буфер
func readFile(dir string, buf *[]byte) ([]byte, int64, error) {
	file, err := os.Open(dir)
	if err != nil {
		return nil, 0, fmt.Errorf("открытие файла: %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("получение размера: %v", err)
	}

	fileSize := stat.Size()

	// Расширяем буфер если нужно
	if int64(cap(*buf)) < fileSize {
		*buf = make([]byte, fileSize*2) // Увеличиваем с двойным запасом
	}

	// Читаем в пред-выделенный буфер
	data := (*buf)[:fileSize]
	n, err := io.ReadFull(file, data)
	if err != nil {
		return nil, 0, fmt.Errorf("чтение: %v", err)
	}

	return data[:n], fileSize, nil
}

// sendRequest отправляет HTTP запрос и возвращает ответ
func sendRequest(client *http.Client, url string, jsonData []byte) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Connection", "keep-alive") // Переиспользуем соединения
	req.Close = false                          // Не закрываем соединение после запроса

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	// Проверяем статус
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, resp.StatusCode, fmt.Errorf("статус: %d", resp.StatusCode)
	}

	return body, resp.StatusCode, nil
}

// saveResponse сохраняет ответ в файл.
func saveResponse(fileName string, response []byte, dir string, format bool) error {
	var data []byte
	if format {
		var formatted bytes.Buffer
		if err := json.Indent(&formatted, response, "", "  "); err != nil {
			data = response // Если форматирование не удалось (возможно, response не JSON), сохраняем как есть
		} else {
			data = formatted.Bytes()
		}
	} else {
		data = response
	}

	filePath := filepath.Join(dir, fileName)
	return os.WriteFile(filePath, data, 0644)
}

// showProgress показывает прогресс в реальном времени
func showProgress(
	totalRequests, totalSuccess, totalErrors *int64,
	total int,
	done chan struct{},
) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			req := atomic.LoadInt64(totalRequests)
			succ := atomic.LoadInt64(totalSuccess)
			errs := atomic.LoadInt64(totalErrors)

			if req > 0 {
				percent := float64(req) * 100 / float64(total)
				fmt.Printf("\r⏳ Прогресс: %d/%d (%.1f%%) | ✅ %d | ❌ %d",
					req, total, percent, succ, errs)
			}
		}
	}
}
