package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"poster/internal/config"
	"poster/internal/logger"
)

// noopLogger возвращает логгер, который ничего не пишет (или пишет в io.Discard)
func noopLogger(t *testing.T) *logger.Logger {
	t.Helper()
	// Создаём логгер с уровнем "error" и файлом в TempDir, чтобы не засорять вывод
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")
	l, err := logger.New("error", logPath)
	if err != nil {
		t.Fatalf("не удалось создать логгер: %v", err)
	}
	// Можно также переопределить вывод в io.Discard, если логгер поддерживает,
	// но в текущей реализации он пишет в файл, что нормально.
	return l
}

// TestReadFile проверяет чтение файлов.
func TestReadFile(t *testing.T) {
	t.Run("успешное чтение", func(t *testing.T) {
		content := []byte(`{"test": true}`)
		tmpFile := filepath.Join(t.TempDir(), "test.json")
		if err := os.WriteFile(tmpFile, content, 0644); err != nil {
			t.Fatal(err)
		}

		buf := make([]byte, 0, 1024)
		data, size, err := readFile(tmpFile, &buf)
		if err != nil {
			t.Fatalf("не ожидалась ошибка: %v", err)
		}
		if size != int64(len(content)) {
			t.Errorf("размер = %d, ожидалось %d", size, len(content))
		}
		if !bytes.Equal(data, content) {
			t.Errorf("данные = %q, ожидалось %q", data, content)
		}
		// Проверяем, что буфер переиспользуется и имеет достаточную ёмкость
		if cap(buf) < len(content) {
			t.Error("ёмкость буфера не увеличилась")
		}
	})

	t.Run("файл не существует", func(t *testing.T) {
		buf := make([]byte, 0, 10)
		_, _, err := readFile("/nonexistent", &buf)
		if err == nil {
			t.Error("ожидалась ошибка, но не получена")
		}
	})

	t.Run("пустой файл", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "empty.json")
		if err := os.WriteFile(tmpFile, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 0, 10)
		data, size, err := readFile(tmpFile, &buf)
		if err != nil {
			t.Fatalf("не ожидалась ошибка: %v", err)
		}
		if size != 0 {
			t.Errorf("размер = %d, ожидалось 0", size)
		}
		if len(data) != 0 {
			t.Errorf("длина данных = %d, ожидалось 0", len(data))
		}
	})
}

// TestSaveResponse проверяет сохранение ответов.
func TestSaveResponse(t *testing.T) {
	tmpDir := t.TempDir()
	responsesDir := filepath.Join(tmpDir, "responses")
	if err := os.MkdirAll(responsesDir, 0755); err != nil {
		t.Fatal(err)
	}

	validJSON := []byte(`{"key":"value"}`)
	invalidJSON := []byte(`not json`)

	t.Run("без форматирования", func(t *testing.T) {
		filename := "resp1.json"
		err := saveResponse(filename, validJSON, responsesDir, false)
		if err != nil {
			t.Fatalf("не ожидалась ошибка: %v", err)
		}
		path := filepath.Join(responsesDir, filename)
		read, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(read, validJSON) {
			t.Errorf("сохранённые данные = %q, ожидалось %q", read, validJSON)
		}
	})

	t.Run("с форматированием (валидный JSON)", func(t *testing.T) {
		filename := "resp2.json"
		err := saveResponse(filename, validJSON, responsesDir, true)
		if err != nil {
			t.Fatalf("не ожидалась ошибка: %v", err)
		}
		path := filepath.Join(responsesDir, filename)
		read, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		// Проверяем, что это отформатированный JSON
		var dst bytes.Buffer
		if err := json.Indent(&dst, validJSON, "", "  "); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(read, dst.Bytes()) {
			t.Errorf("сохранённые данные = %q, ожидалось форматированное %q", read, dst.Bytes())
		}
	})

	t.Run("с форматированием (невалидный JSON) – сохраняется как есть", func(t *testing.T) {
		filename := "resp3.json"
		err := saveResponse(filename, invalidJSON, responsesDir, true)
		if err != nil {
			t.Fatalf("не ожидалась ошибка: %v", err)
		}
		path := filepath.Join(responsesDir, filename)
		read, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(read, invalidJSON) {
			t.Errorf("сохранённые данные = %q, ожидалось исходное %q", read, invalidJSON)
		}
	})
}

// TestSendRequest проверяет отправку HTTP запросов.
func TestSendRequest(t *testing.T) {
	// Создаём тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем метод и заголовки
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		// Читаем тело
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Отвечаем успехом
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		// Эхо тела
		_, _ = w.Write(body)
	}))
	defer server.Close()

	ctx := context.Background()
	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("успешный запрос", func(t *testing.T) {
		jsonData := []byte(`{"msg":"hello"}`)
		respBody, status, err := sendRequest(ctx, client, server.URL, jsonData)
		if err != nil {
			t.Fatalf("не ожидалась ошибка: %v", err)
		}
		if status != http.StatusOK {
			t.Errorf("статус = %d, ожидалось %d", status, http.StatusOK)
		}
		if !bytes.Equal(respBody, jsonData) {
			t.Errorf("тело ответа = %q, ожидалось %q", respBody, jsonData)
		}
	})

	t.Run("отмена контекста", func(t *testing.T) {
		ctxCancel, cancel := context.WithCancel(ctx)
		cancel() // сразу отменяем
		_, _, err := sendRequest(ctxCancel, client, server.URL, []byte(`{}`))
		if err == nil {
			t.Error("ожидалась ошибка отмены контекста")
		}
	})

	t.Run("сервер возвращает ошибку 404", func(t *testing.T) {
		// Создаём другой сервер, который всегда возвращает 404
		server404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server404.Close()

		_, status, err := sendRequest(ctx, client, server404.URL, []byte(`{}`))
		if err == nil {
			t.Error("ожидалась ошибка из-за статуса 404")
		}
		if status != http.StatusNotFound {
			t.Errorf("статус = %d, ожидалось %d", status, http.StatusNotFound)
		}
	})
}

// TestPostGracefulShutdown проверяет, что post корректно завершается при отмене контекста.
func TestPostGracefulShutdown(t *testing.T) {
	// Создаём много файлов, чтобы обработка не завершилась мгновенно
	tmpDir := t.TempDir()
	reqDir := filepath.Join(tmpDir, "requests")
	if err := os.MkdirAll(reqDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Создаём 20 файлов
	fileNames := make([]string, 20)
	for i := 0; i < 20; i++ {
		fname := fmt.Sprintf("%d.json", i+1)
		path := filepath.Join(reqDir, fname)
		content := fmt.Sprintf(`{"id":%d}`, i+1)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		fileNames[i] = path
	}

	// Сервер с задержкой, чтобы замедлить обработку
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // небольшая задержка
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer server.Close()

	cfg := &config.Config{
		URL:     server.URL,
		Req:     reqDir,
		Indent:  false,
		Timeout: 10,
		Workers: 4,
		Log:     "error",
	}
	log := noopLogger(t)
	client := &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Second}

	ctx, cancel := context.WithCancel(context.Background())

	// Запускаем post в горутине
	resultsChan := make(chan []Result, 1)
	go func() {
		results := post(cfg, ctx, fileNames, reqDir, client, log) // ответы сохраняются в reqDir, но это не важно
		resultsChan <- results
	}()

	// Ждём немного, чтобы началась обработка
	time.Sleep(200 * time.Millisecond)
	// Отменяем контекст
	cancel()

	// Ждём завершения post
	select {
	case results := <-resultsChan:
		// Проверяем, что все результаты имеют ошибку контекста или успешны (если успели завершиться)
		// Но важно, что часть запросов была прервана.
		// Мы не можем точно предсказать количество, но хотя бы одна ошибка должна быть из-за ctx.Done()
		hasContextError := false
		for _, r := range results {
			if r.Err == context.Canceled || r.Err == context.DeadlineExceeded {
				hasContextError = true
				break
			}
		}
		if !hasContextError {
			t.Error("не найдено ни одного результата с ошибкой отмены контекста")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("тест завис, post не завершился")
	}
}
