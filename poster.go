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
	"time"
)

// Result содержит результат обработки файла
type Result struct {
	FileName     string
	FileSize     int64         // Размер файла запроса
	RequestSize  int           // Размер JSON данных
	ResponseSize int           // Размер ответа
	Duration     time.Duration // Время обработки
	StatusCode   int           // HTTP статус код
	Err          error
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

	// Чтение всех запросов
	filePaths, err := filepath.Glob(cfg.RequestsDir + "/*.json")
	if err != nil {
		mainLogger.Fatal("Ошибка чтения директории с запросами", map[string]interface{}{
			"directory": cfg.RequestsDir,
			"error":     err.Error(),
		})
	}

	if len(filePaths) == 0 {
		mainLogger.Info("В папке requests не найдено JSON файлов")
		return
	}
	mainLogger.Info("Найдены файлы для отправки", map[string]interface{}{
		"count": len(filePaths),
	})

	// Ограничиваем количество одновременных горутин
	if len(filePaths) < cfg.Workers {
		cfg.Workers = len(filePaths)
	}

	mainLogger.Debug("Настройка воркеров", map[string]interface{}{
		"workers": cfg.Workers,
		"files":   len(filePaths),
	})

	// Каналы для работы
	filesChan := make(chan string, len(filePaths))
	resultsChan := make(chan Result, len(filePaths))

	// Создание HTTP клиента с таймаутом
	client := &http.Client{
		Timeout: time.Duration(cfg.Timeout) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.Workers * 10, // Максимальное общее количество "бездействующих" (idle) соединений в пуле ко всем хостам.
			MaxIdleConnsPerHost: cfg.Workers * 10, // Максимальное количество idle-соединений к одному конкретному хосту.
			MaxConnsPerHost:     cfg.Workers * 20, // Максимальное общее количество соединений к одному хосту (idle + active).

			IdleConnTimeout: time.Duration(cfg.Timeout*3) * time.Second, // Таймаут на неактивные соединения
		},
	}

	// Запускаем воркеров
	var wg sync.WaitGroup
	workerLogger := mainLogger.WithFields(map[string]interface{}{
		"component": "worker",
	})
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go work(i, client, cfg.URL, cfg.ResponsesDir, filesChan, resultsChan, &wg, workerLogger)
	}

	// Отправляем задачи в канал
	for _, filePath := range filePaths {
		filesChan <- filePath
	}
	close(filesChan)
	mainLogger.Debug("Все задачи отправлены в канал")

	// Ждем завершения воркеров
	go func() {
		wg.Wait()
		close(resultsChan)
		mainLogger.Debug("Все воркеры завершили работу")
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
func work(id int, client *http.Client, url, responsesDir string,
	filesChan <-chan string, resultsChan chan<- Result, wg *sync.WaitGroup,
	log *logger.Logger) {
	defer wg.Done()

	workerLogger := log.WithFields(map[string]interface{}{
		"worker_id": id,
	})

	workerLogger.Debug("Воркер запущен")

	done := 0
	for filePath := range filesChan {
		done++
		fileName := filepath.Base(filePath)

		startTime := time.Now()
		workerLogger.Debug("Начало обработки файла", map[string]interface{}{
			"file":  fileName,
			"done":  done,
			"start": startTime.Format(time.RFC3339),
		})

		// Чтение JSON файла
		jsonData, err := os.ReadFile(filePath)
		if err != nil {
			workerLogger.Error("Ошибка чтения файла", map[string]interface{}{
				"file":  fileName,
				"error": err.Error(),
			})
			resultsChan <- Result{
				FileName: fileName,
				Duration: time.Since(startTime),
				Err:      fmt.Errorf("чтение файла: %v", err),
			}
			continue
		}

		// Получаем размер файла
		fileInfo, _ := os.Stat(filePath)
		fileSize := int64(0)
		if fileInfo != nil {
			fileSize = fileInfo.Size()
		}

		// Проверка валидности JSON
		if !json.Valid(jsonData) {
			workerLogger.Error("Невалидный JSON", map[string]interface{}{
				"file":      fileName,
				"file_size": fileSize,
			})
			resultsChan <- Result{
				FileName:    fileName,
				FileSize:    fileSize,
				RequestSize: len(jsonData),
				Duration:    time.Since(startTime),
				Err:         fmt.Errorf("невалидный JSON"),
			}
			continue
		}

		workerLogger.Debug("JSON файл прочитан", map[string]interface{}{
			"file":      fileName,
			"file_size": fileSize,
			"json_size": len(jsonData),
		})

		// Отправка запроса на сервер
		response, statusCode, err := sendRequest(client, url, jsonData, workerLogger)
		requestDuration := time.Since(startTime)
		if err != nil {
			workerLogger.Error("Ошибка отправки запроса", map[string]interface{}{
				"file":      fileName,
				"duration":  requestDuration.String(),
				"error":     err.Error(),
				"file_size": fileSize,
			})
			resultsChan <- Result{
				FileName:    fileName,
				FileSize:    fileSize,
				RequestSize: len(jsonData),
				Duration:    requestDuration,
				StatusCode:  statusCode,
				Err:         fmt.Errorf("отправка запроса: %v", err),
			}
			continue
		}

		workerLogger.Info("Запрос успешно отправлен", map[string]interface{}{
			"file":        fileName,
			"duration":    requestDuration.String(),
			"status_code": statusCode,
			"file_size":   fileSize,
			"resp_size":   len(response),
		})

		// Сохранение ответа
		err = saveResponse(fileName, response, responsesDir, workerLogger)
		totalDuration := time.Since(startTime)

		// Сохранение ответа
		if err != nil {
			workerLogger.Error("Ошибка сохранения ответа", map[string]interface{}{
				"file":      fileName,
				"duration":  totalDuration.String(),
				"error":     err.Error(),
				"resp_size": len(response),
			})
			resultsChan <- Result{
				FileName:     fileName,
				FileSize:     fileSize,
				RequestSize:  len(jsonData),
				ResponseSize: len(response),
				Duration:     totalDuration,
				StatusCode:   statusCode,
				Err:          fmt.Errorf("сохранение ответа: %v", err),
			}
			continue
		}

		workerLogger.Info("Ответ успешно сохранен", map[string]interface{}{
			"file":         fileName,
			"total_time":   totalDuration.String(),
			"request_time": requestDuration.String(),
			"save_time":    (totalDuration - requestDuration).String(),
			"status_code":  statusCode,
			"file_size":    fileSize,
			"req_size":     len(jsonData),
			"resp_size":    len(response),
		})

		resultsChan <- Result{
			FileName:     fileName,
			FileSize:     fileSize,
			RequestSize:  len(jsonData),
			ResponseSize: len(response),
			Duration:     totalDuration,
			StatusCode:   statusCode,
			Err:          nil,
		}
	}

	workerLogger.Debug("Воркер завершен", map[string]interface{}{
		"done": done,
	})
}

// sendRequest отправляет JSON на сервер (без изменений)
func sendRequest(client *http.Client, url string, jsonData []byte, log *logger.Logger) ([]byte, int, error) {
	// Создание POST запроса
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, err
	}

	// Установка заголовков
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	log.Debug("Отправка HTTP запроса", map[string]interface{}{
		"url":          url,
		"method":       req.Method,
		"content_type": req.Header.Get("Content-Type"),
		"data_size":    len(jsonData),
		"timestamp":    time.Now().Format(time.RFC3339Nano),
	})

	start := time.Now()
	resp, err := client.Do(req) // Выполнение запроса
	duration := time.Since(start)

	if err != nil {
		log.Error("HTTP запрос не удался", map[string]interface{}{
			"duration":    duration.String(),
			"duration_ms": duration.Milliseconds(),
			"error":       err.Error(),
			"url":         url,
		})
		return nil, 0, err
	}
	defer resp.Body.Close()

	// Чтение ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("Ошибка чтения ответа", map[string]interface{}{
			"duration":     duration.String(),
			"status_code":  resp.StatusCode,
			"error":        err.Error(),
			"url":          url,
			"content_type": resp.Header.Get("Content-Type"),
		})
		return nil, resp.StatusCode, err
	}

	// Логируем получение ответа
	log.Warn("Получен HTTP ответ", map[string]interface{}{
		"duration":       duration.String(),
		"duration_ms":    duration.Milliseconds(),
		"status_code":    resp.StatusCode,
		"response_size":  len(body),
		"url":            url,
		"content_type":   resp.Header.Get("Content-Type"),
		"content_length": resp.Header.Get("Content-Length"),
		"server":         resp.Header.Get("Server"),
		"date":           resp.Header.Get("Date"),
	})

	log.Debug("Получен HTTP ответ", map[string]interface{}{
		"duration":    duration.String(),
		"status_code": resp.StatusCode,
		"size":        len(body),
		"headers":     resp.Header,
	})

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warn("Сервер вернул ошибку", map[string]interface{}{
			"status_code":  resp.StatusCode,
			"body_preview": string(body[:min(200, len(body))]),
		})
		return body, resp.StatusCode, fmt.Errorf("сервер вернул статус: %d", resp.StatusCode)
	}

	return body, resp.StatusCode, nil
}

// saveResponse сохраняет ответ директорию
func saveResponse(fileName string, response []byte, path string, log *logger.Logger) error {
	startTime := time.Now()

	log.Debug("Начало сохранения ответа", map[string]interface{}{
		"file_name":     fileName,
		"response_size": len(response),
		"target_dir":    path,
		"start_time":    startTime.Format(time.RFC3339Nano),
	})

	// Форматирование JSON для красивого вывода
	var formattedJSON bytes.Buffer
	formatStart := time.Now()
	if err := json.Indent(&formattedJSON, response, "", "  "); err != nil {
		log.Warn("Не удалось отформатировать JSON, сохраняем как есть", map[string]interface{}{
			"file_name": fileName,
			"error":     err.Error(),
			"warning":   "response might not be valid JSON",
		})
		formattedJSON.Write(response) // Если JSON невалидный, сохраняем как есть
	}
	formatDuration := time.Since(formatStart)

	// Определяем полный путь к файлу
	filePath := filepath.Join(path, fileName)

	log.Debug("Подготовка к записи файла", map[string]interface{}{
		"file_name":         fileName,
		"full_path":         filePath,
		"original_size":     len(response),
		"formatted_size":    formattedJSON.Len(),
		"format_time_ms":    formatDuration.Milliseconds(),
		"compression_ratio": fmt.Sprintf("%.2f%%", float64(formattedJSON.Len())*100/float64(len(response))),
	})

	// Записываем файл
	writeStart := time.Now()
	if err := os.WriteFile(filePath, formattedJSON.Bytes(), 0644); err != nil {
		log.Error("Ошибка записи файла", map[string]interface{}{
			"file_path":     filePath,
			"file_size":     formattedJSON.Len(),
			"error":         err.Error(),
			"write_time_ms": time.Since(writeStart).Milliseconds(),
			"total_time_ms": time.Since(startTime).Milliseconds(),
			"permissions":   "0644",
		})
		return fmt.Errorf("запись файла %s: %v", filePath, err)
	}

	return nil
}

func statistic(resultsChan <-chan Result, log *logger.Logger) {
	// Собираем результаты с расширенной статистикой
	successCount, errorCount := 0, 0
	var totalDuration time.Duration
	var totalFileSize int64
	var totalRequestSize int
	var totalResponseSize int
	statusCodeStats := make(map[int]int)
	durationStats := struct {
		min time.Duration
		max time.Duration
		sum time.Duration
	}{min: time.Hour} // Инициализируем большим значением

	for result := range resultsChan {
		if result.Err != nil {
			errorCount++
			log.Error("Ошибка обработки файла", map[string]interface{}{
				"file_name":     result.FileName,
				"error":         result.Err.Error(),
				"duration_ms":   result.Duration.Milliseconds(),
				"status_code":   result.StatusCode,
				"file_size":     result.FileSize,
				"request_size":  result.RequestSize,
				"response_size": result.ResponseSize,
			})
		} else {
			successCount++

			// Обновляем статистику
			totalDuration += result.Duration
			totalFileSize += result.FileSize
			totalRequestSize += result.RequestSize
			totalResponseSize += result.ResponseSize

			// Статистика по статус кодам
			if result.StatusCode > 0 {
				statusCodeStats[result.StatusCode]++
			}

			// Минимальное/максимальное время
			if result.Duration < durationStats.min {
				durationStats.min = result.Duration
			}
			if result.Duration > durationStats.max {
				durationStats.max = result.Duration
			}
			durationStats.sum += result.Duration

			log.Info("Файл успешно обработан", map[string]interface{}{
				"file_name":     result.FileName,
				"duration_ms":   result.Duration.Milliseconds(),
				"status_code":   result.StatusCode,
				"file_size":     result.FileSize,
				"request_size":  result.RequestSize,
				"response_size": result.ResponseSize,
				"success":       true,
			})
		}
	}

	// Выводим итоговую статистику
	if successCount > 0 {
		avgDuration := durationStats.sum / time.Duration(successCount)
		log.Info("Статистика обработки файлов", map[string]interface{}{
			"total_files":              successCount + errorCount,
			"successful":               successCount,
			"failed":                   errorCount,
			"success_rate":             fmt.Sprintf("%.2f%%", float64(successCount)*100/float64(successCount+errorCount)),
			"total_duration_sec":       totalDuration.Seconds(),
			"avg_duration_ms":          avgDuration.Milliseconds(),
			"min_duration_ms":          durationStats.min.Milliseconds(),
			"max_duration_ms":          durationStats.max.Milliseconds(),
			"total_file_size_mb":       fmt.Sprintf("%.2f MB", float64(totalFileSize)/(1024*1024)),
			"total_request_size_mb":    fmt.Sprintf("%.2f MB", float64(totalRequestSize)/(1024*1024)),
			"total_response_size_mb":   fmt.Sprintf("%.2f MB", float64(totalResponseSize)/(1024*1024)),
			"avg_file_size_kb":         fmt.Sprintf("%.2f KB", float64(totalFileSize)/float64(successCount)/1024),
			"avg_request_size_kb":      fmt.Sprintf("%.2f KB", float64(totalRequestSize)/float64(successCount)/1024),
			"avg_response_size_kb":     fmt.Sprintf("%.2f KB", float64(totalResponseSize)/float64(successCount)/1024),
			"throughput_files_per_sec": fmt.Sprintf("%.2f", float64(successCount)/totalDuration.Seconds()),
			"status_codes":             statusCodeStats,
		})
	}

	log.Info("Обработка завершена", map[string]interface{}{
		"successful": successCount,
		"failed":     errorCount,
		"total":      successCount + errorCount,
		"duration":   totalDuration.String(),
	})
}
