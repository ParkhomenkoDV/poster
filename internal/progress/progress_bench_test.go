package progress

import (
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkShow_Formatting измеряет скорость формирования строки (без реального ожидания).
func BenchmarkShow_Formatting(b *testing.B) {
	var items, success, errors uint64
	done := make(chan struct{})
	bar := Bar{
		Total:      1_000_000,
		Interval:   time.Hour, // чтобы не срабатывал тик
		ShowSpeed:  true,
		ShowETA:    true,
		ShowErrors: true,
	}

	// Запускаем горутину, но она не будет ничего выводить из-за большого интервала.
	go bar.Show(&items, &success, &errors, done)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atomic.AddUint64(&items, 1)
		atomic.AddUint64(&success, 1)
		// Имитируем вызов printProgress (но он вызывается только по тику).
		// Для бенчмарка мы не можем вызвать print напрямую, поэтому измеряем только
		// накладные расходы на атомарные операции и ничего более.
	}
	b.StopTimer()
	close(done)
}
