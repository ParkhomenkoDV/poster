package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"poster/internal/config"
	"poster/internal/logger"
	"poster/internal/progress"
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
		os.Exit(1)
	}

	// Создание логгера
	lgr, err := logger.New(cfg.Log, "log.json")
	if err != nil {
		fmt.Printf("Ошибка инициализации логгера: %v\n", err)
		os.Exit(1)
	}
	defer lgr.Info("Приложение завершено")

	// Добавляем поля по умолчанию
	lgr = lgr.WithFields(map[string]interface{}{
		"app":       "poster",
		"pid":       os.Getpid(),
		"timestamp": time.Now().Format(time.DateTime),
	})

	lgr.Info("Логгер инициализирован", map[string]interface{}{
		"level": cfg.Log,
		"file":  "log.json",
	})

	lgr.Info("Запуск приложения", map[string]interface{}{
		"config": map[string]interface{}{
			"url":          cfg.URL,
			"requests_dir": cfg.RequestsDir,
			"timeout":      cfg.Timeout,
			"workers":      cfg.Workers,
			"indent":       cfg.Indent,
			"level":        cfg.Log,
			"file":         "log.json",
		},
	})

	// Проверка наличия директории с запросами
	if _, err := os.Stat(cfg.RequestsDir); os.IsNotExist(err) {
		lgr.Fatal("Директория с запросами не существует", map[string]interface{}{
			"directory": cfg.RequestsDir,
		})
	}

	// Получаем родительскую директорию
	parentDir := filepath.Dir(cfg.RequestsDir)

	// Создание директории для ответов, если её нет
	responsesDir := filepath.Join(parentDir, "responses")
	if err := os.MkdirAll(responsesDir, 0755); err != nil {
		lgr.Fatal("Ошибка создания директории для ответов", map[string]interface{}{
			"directory": responsesDir,
			"error":     err.Error(),
		})
	}

	// Получение списка файлов
	fileDirs, err := filepath.Glob(cfg.RequestsDir + "/*.json")
	if err != nil {
		lgr.Fatal("Ошибка чтения директории с запросами", map[string]interface{}{
			"directory": cfg.RequestsDir,
			"error":     err.Error(),
		})
	}

	if len(fileDirs) == 0 {
		lgr.Info("В папке requests не найдено JSON файлов")
		return
	}
	lgr.Info("Найдены файлы для отправки", map[string]interface{}{
		"count":   len(fileDirs),
		"workers": cfg.Workers,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigChan:
			lgr.Warn("Signal received, shutting down gracefully", map[string]interface{}{
				"signal": sig.String(),
			})
			cancel()
		case <-ctx.Done():
		}
	}()

	// Создание HTTP клиента с пулом соединений
	client := &http.Client{
		Timeout: time.Duration(cfg.Timeout) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100, // Максимальное общее количество "бездействующих" (idle) соединений в пуле ко всем хостам.
			MaxIdleConnsPerHost: 100, // Максимальное количество idle-соединений к одному конкретному хосту.
			MaxConnsPerHost:     100, // Максимальное общее количество соединений к одному хосту (idle + active).

			IdleConnTimeout:       90 * time.Second, // Таймаут на неактивные соединения
			ResponseHeaderTimeout: 30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			// DisableCompression: true, // Отключение сжатия
		},
	}

	// Запускаем обработку
	startTime := time.Now()
	results := post(cfg, ctx, fileDirs, responsesDir, client, lgr)
	totalDuration := time.Since(startTime)

	// Считаем статистику
	successCount, errorCount := 0, 0
	for _, result := range results {
		if result.Err != nil {
			errorCount++
			lgr.Error("❌", map[string]interface{}{
				"filename": result.FileName,
				"error":    result.Err,
			})
		} else {
			successCount++
			lgr.Info("✅", map[string]interface{}{
				"filename":   result.FileName,
				"statusCode": result.StatusCode,
				"duration":   result.Duration.Milliseconds(),
			})
		}
	}

	fmt.Println()
	fmt.Printf("📊 ИТОГО: Успешно %d | Ошибок %d | Всего %d\n", successCount, errorCount, len(fileDirs))
	fmt.Printf("⏱️ Общее время: %v\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("⚡ Скорость: %.2f файлов/сек\n", float64(len(fileDirs))/totalDuration.Seconds())
}

// post запускает конвейерную обработку
func post(
	cfg *config.Config,
	ctx context.Context,
	fileDirs []string,
	responsesDir string,
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
		totalRequests uint64
		totalSuccess  uint64
		totalErrors   uint64
	)

	// Запускаем рабочих
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go work(i, ctx, client, cfg.URL, responsesDir, cfg.Indent,
			taskChan, resultChan, &wg, log,
			&totalRequests, &totalSuccess, &totalErrors) // счетчики
	}

	// Горутина прогресса
	progressDone := make(chan struct{})
	bar := progress.New(uint64(len(fileDirs)), time.Second, true, false, true)
	go bar.Show(&totalRequests, &totalSuccess, &totalErrors, progressDone)

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
	id int,
	ctx context.Context,
	client *http.Client,
	url string,
	responsesDir string,
	format bool,
	taskChan <-chan string,
	resultChan chan<- Result,
	wg *sync.WaitGroup,
	log *logger.Logger,
	totalRequests, totalSuccess, totalErrors *uint64,
) {
	defer wg.Done()

	// Локальный буфер для чтения файлов
	buf := make([]byte, 0, 1024*1024) // 1MB начальный размер

	for fileDir := range taskChan {
		select {
		case <-ctx.Done():
			resultChan <- Result{
				FileName: filepath.Base(fileDir),
				Err:      ctx.Err(),
			}
			atomic.AddUint64(totalErrors, 1)
			atomic.AddUint64(totalRequests, 1)
			continue
		default:
		}

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
			atomic.AddUint64(totalErrors, 1)
			atomic.AddUint64(totalRequests, 1)
			continue
		}

		// Валидируем JSON
		if !json.Valid(jsonData) {
			resultChan <- Result{
				FileName:    fileName,
				FileSize:    fileSize,
				RequestSize: len(jsonData),
				Duration:    time.Since(startTime),
				Err:         fmt.Errorf("invalid JSON"),
				StatusCode:  0,
			}
			atomic.AddUint64(totalErrors, 1)
			atomic.AddUint64(totalRequests, 1)
			continue
		}

		// Отправляем HTTP запрос
		response, statusCode, err := sendRequest(ctx, client, url, jsonData)
		if err != nil {
			resultChan <- Result{
				FileName:     fileName,
				FileSize:     fileSize,
				RequestSize:  len(jsonData),
				ResponseSize: 0,
				Duration:     time.Since(startTime),
				StatusCode:   statusCode,
				Err:          fmt.Errorf("отправка: %v", err),
			}
			atomic.AddUint64(totalErrors, 1)
			atomic.AddUint64(totalRequests, 1)
			continue
		}

		// Сохраняем ответ
		saveErr := saveResponse(fileName, response, responsesDir, format)
		totalDuration := time.Since(startTime)
		if saveErr != nil {
			resultChan <- Result{
				FileName:     fileName,
				FileSize:     fileSize,
				RequestSize:  len(jsonData),
				ResponseSize: len(response),
				Duration:     totalDuration,
				StatusCode:   statusCode,
				Err:          fmt.Errorf("saving response: %v", err),
			}
			atomic.AddUint64(totalErrors, 1)
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
			atomic.AddUint64(totalSuccess, 1)
		}

		// Обновляем статистику
		atomic.AddUint64(totalRequests, 1)
	}
}

// readFile читает файл, переиспользуя буфер
func readFile(dir string, buf *[]byte) ([]byte, int64, error) {
	file, err := os.Open(dir)
	if err != nil {
		return nil, 0, fmt.Errorf("oppening file: %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("getting file info: %v", err)
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
		return nil, 0, fmt.Errorf("reading file: %v", err)
	}

	return data[:n], fileSize, nil
}

// sendRequest отправляет HTTP запрос и возвращает ответ
func sendRequest(ctx context.Context, client *http.Client, url string, jsonData []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
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
		return body, resp.StatusCode, fmt.Errorf("HTTP status: %d", resp.StatusCode)
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
