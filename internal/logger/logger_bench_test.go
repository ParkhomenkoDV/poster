package logger

import (
	"io"
	"testing"
)

// BenchmarkLogInfo измеряет производительность логирования уровня Info без полей.
func BenchmarkLogInfo(b *testing.B) {
	logger := &Logger{
		level:  DEBUG,
		output: io.Discard,
		fields: make(map[string]interface{}),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("тестовое сообщение")
	}
}

// BenchmarkLogDebug измеряет производительность логирования уровня Debug без полей.
func BenchmarkLogDebug(b *testing.B) {
	logger := &Logger{
		level:  DEBUG,
		output: io.Discard,
		fields: make(map[string]interface{}),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("отладочное сообщение")
	}
}

// BenchmarkLogError измеряет производительность логирования уровня Error без полей.
func BenchmarkLogError(b *testing.B) {
	logger := &Logger{
		level:  DEBUG,
		output: io.Discard,
		fields: make(map[string]interface{}),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Error("сообщение об ошибке")
	}
}

// BenchmarkLogWithOneField измеряет производительность логирования с одним полем.
func BenchmarkLogWithOneField(b *testing.B) {
	logger := &Logger{
		level:  DEBUG,
		output: io.Discard,
		fields: make(map[string]interface{}),
	}
	fields := map[string]interface{}{"user_id": 12345}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("сообщение с полем", fields)
	}
}

// BenchmarkLogWithTenFields измеряет производительность логирования с десятью полями.
func BenchmarkLogWithTenFields(b *testing.B) {
	logger := &Logger{
		level:  DEBUG,
		output: io.Discard,
		fields: make(map[string]interface{}),
	}
	fields := map[string]interface{}{
		"user_id":    12345,
		"method":     "GET",
		"path":       "/api/v1/test",
		"status":     200,
		"latency":    1.5,
		"source":     "web",
		"region":     "eu-west",
		"version":    "v1.2.3",
		"trace_id":   "abc123",
		"session_id": "xyz789",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("сообщение с десятью полями", fields)
	}
}

// BenchmarkLogWithFieldsMerge измеряет производительность логирования с объединением полей (несколько map).
// Имитирует ситуацию, когда поля передаются в виде нескольких аргументов.
func BenchmarkLogWithFieldsMerge(b *testing.B) {
	logger := &Logger{
		level:  DEBUG,
		output: io.Discard,
		fields: make(map[string]interface{}),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("сообщение с несколькими источниками полей",
			map[string]interface{}{"a": 1, "b": 2},
			map[string]interface{}{"c": 3, "d": 4},
			map[string]interface{}{"e": 5, "f": 6},
		)
	}
}

// BenchmarkLogParallel измеряет производительность при параллельном логировании из нескольких горутин.
// Это показывает влияние мьютекса и конкуренции.
func BenchmarkLogParallel(b *testing.B) {
	logger := &Logger{
		level:  DEBUG,
		output: io.Discard,
		fields: make(map[string]interface{}),
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("параллельное сообщение", map[string]interface{}{"goroutine": 0})
		}
	})
}

// BenchmarkLogWithCallerInfo показывает накладные расходы на получение информации о caller.
// В текущей реализации это происходит всегда, поэтому этот бенчмарк просто измеряет это.
func BenchmarkLogWithCallerInfo(b *testing.B) {
	logger := &Logger{
		level:  DEBUG,
		output: io.Discard,
		fields: make(map[string]interface{}),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("сообщение с caller")
	}
}

// BenchmarkLogNoOutput проверяет, как быстро работает логгер, если вывод отсутствует (output == nil).
// Это может произойти при уровне NOLOG, но там сразу выход.
// Мы специально создаём логгер с output = nil, чтобы измерить только накладные расходы на формирование.
func BenchmarkLogNoOutput(b *testing.B) {
	logger := &Logger{
		level:  DEBUG,
		output: nil,
		fields: make(map[string]interface{}),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("сообщение без вывода")
	}
}
