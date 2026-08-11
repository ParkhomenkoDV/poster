package progress

import (
	"bufio"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkPrintProgress замеряет производительность printProgress.
// Использует io.Discard для исключения накладных расходов на реальный вывод.
func BenchmarkPrintProgress(b *testing.B) {
	bar := &Bar{
		Total:      1000,
		ShowSpeed:  true,
		ShowETA:    true,
		ShowErrors: true,
	}

	var items, success, errors uint64
	atomic.StoreUint64(&items, 500)
	atomic.StoreUint64(&success, 450)
	atomic.StoreUint64(&errors, 50)

	prevItems := uint64(400)
	prevTime := time.Now().Add(-time.Second) // симулируем разницу в 1 секунду

	// Используем буферизованный writer, пишущий в /dev/null (io.Discard)
	bw := bufio.NewWriter(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Можно слегка изменять значения для большей реалистичности,
		// но это не критично для бенчмарка.
		bar.printProgress(bw, &items, &success, &errors, prevItems, prevTime)
	}
}

// BenchmarkFormatDuration замеряет производительность форматирования.
func BenchmarkFormatDuration(b *testing.B) {
	durations := []time.Duration{
		0,
		5 * time.Second,
		2*time.Minute + 30*time.Second,
		1*time.Hour + 20*time.Minute + 15*time.Second,
		100 * time.Hour,
		-2*time.Hour - 30*time.Minute,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, d := range durations {
			formatDuration(d)
		}
	}
}
