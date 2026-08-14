package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	json "github.com/goccy/go-json"
)

// Средняя емкость логгера
const capacity = 3

// Level = уровень логирования
type Level int

const (
	STDOUT Level = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
	NOLOG
)

var levelNames = map[Level]string{
	STDOUT: "STDOUT",
	DEBUG:  "DEBUG",
	INFO:   "INFO",
	WARN:   "WARN",
	ERROR:  "ERROR",
	FATAL:  "FATAL",
	NOLOG:  "NOLOG",
}

// Log = запись лога
type Log struct {
	Timestamp time.Time      `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
	File      string         `json:"file,omitempty"`
	Line      int            `json:"line,omitempty"`
	Function  string         `json:"function,omitempty"`
}

// Logger основной логгер
type Logger struct {
	level  Level
	output io.Writer
	mu     sync.Mutex
	fields map[string]any
}

// New создает новый логгер
func New(levelName string, outputFile string) (*Logger, error) {
	// Определяем уровень логирования
	var level Level
	switch strings.ToLower(levelName) {
	case "fatal":
		level = FATAL
	case "error":
		level = ERROR
	case "warn", "warning":
		level = WARN
	case "info":
		level = INFO
	case "debug":
		level = DEBUG
	case "stdout":
		return &Logger{level: STDOUT, output: os.Stdout, fields: make(map[string]any, capacity)}, nil
	default: // nolog
		return &Logger{level: NOLOG, output: io.Discard, fields: nil}, nil
	}

	// Настраиваем вывод
	output, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return &Logger{
			level:  level,
			output: os.Stderr,
			fields: make(map[string]any, capacity),
		}, fmt.Errorf("открытие файла логов: %v", err)
	}

	return &Logger{
		level:  level,
		output: output,
		fields: make(map[string]any, capacity),
	}, nil
}

// SetLevel устанавливает уровень логирования
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput устанавливает вывод
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// WithFields добавляет постоянные поля к логгеру
func (l *Logger) WithFields(fields map[string]any) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(map[string]any, len(l.fields)+len(fields)+capacity)
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		level:  l.level,
		output: l.output,
		fields: newFields,
	}
}

// log записывает сообщение
func (l *Logger) log(level Level, msg string, fields map[string]any) {
	if level < l.level {
		return
	}

	// Получаем информацию о caller
	pc, file, line, ok := runtime.Caller(2)
	var funcName string
	if ok {
		file = filepath.Base(file)
		if fn := runtime.FuncForPC(pc); fn != nil {
			funcName = fn.Name()
			// Оставляем только имя функции
			if idx := strings.LastIndex(funcName, "."); idx != -1 {
				funcName = funcName[idx+1:]
			}
		}
	}

	// Создаем запись
	entry := Log{
		Timestamp: time.Now().UTC(),
		Level:     levelNames[level],
		Message:   msg,
		Fields:    make(map[string]any, len(l.fields)+len(fields)),
		File:      file,
		Line:      line,
		Function:  funcName,
	}

	// Объединяем поля
	for k, v := range l.fields {
		entry.Fields[k] = v
	}
	for k, v := range fields {
		entry.Fields[k] = v
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Проверяем, есть ли output
	if l.output == nil {
		return
	}

	data, err := json.Marshal(entry)
	if err != nil {
		// Если не можем замаршалить в JSON, пишем просто текст
		fmt.Fprintf(l.output, "[%s] %s: %s\n",
			entry.Timestamp.Format(time.DateTime),
			entry.Level,
			msg)
	} else {
		fmt.Fprintln(l.output, string(data))
	}
}

// Debug логирует отладочное сообщение
func (l *Logger) Debug(msg string, fields ...map[string]any) {
	l.log(DEBUG, msg, mergeFields(fields))
}

// Info логирует информационное сообщение
func (l *Logger) Info(msg string, fields ...map[string]any) {
	l.log(INFO, msg, mergeFields(fields))
}

// Warn логирует предупреждение
func (l *Logger) Warn(msg string, fields ...map[string]any) {
	l.log(WARN, msg, mergeFields(fields))
}

// Error логирует ошибку
func (l *Logger) Error(msg string, fields ...map[string]any) {
	l.log(ERROR, msg, mergeFields(fields))
}

// Fatal логирует фатальную ошибку и завершает программу
func (l *Logger) Fatal(msg string, fields ...map[string]any) {
	l.log(FATAL, msg, mergeFields(fields))
	os.Exit(1)
}

// mergeFields объединяет несколько мап полей
func mergeFields(fields []map[string]any) map[string]any {
	switch len(fields) {
	case 0:
		return nil
	case 1:
		return fields[0] // без копирования
	default:
		totalCapacity := 0
		for _, f := range fields {
			totalCapacity += len(f)
		}

		result := make(map[string]any, totalCapacity)
		for _, f := range fields {
			for k, v := range f {
				result[k] = v
			}
		}
		return result
	}
}
