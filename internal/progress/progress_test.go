package progress

import (
	"bytes"
	"sync/atomic"
	"testing"
	"time"
)

func TestShow_UpdatesCorrectly(t *testing.T) {
	var items, success, errors uint64
	done := make(chan struct{})
	buf := &bytes.Buffer{}

	bar := Bar{
		Total:      100,
		Interval:   50 * time.Millisecond,
		ShowSpeed:  true,
		ShowETA:    true,
		ShowErrors: true,
	}

	go bar.Show(&items, &success, &errors, done)

	// Имитируем обработку
	for i := 0; i < 50; i++ {
		atomic.AddUint64(&items, 1)
		if i%2 == 0 {
			atomic.AddUint64(&success, 1)
		} else {
			atomic.AddUint64(&errors, 1)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Даём время на отображение
	time.Sleep(100 * time.Millisecond)
	close(done)
	time.Sleep(50 * time.Millisecond) // даём завершиться

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("50 / 100")) {
		t.Errorf("ожидалось 50 / 100, получено: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("✅")) || !bytes.Contains([]byte(output), []byte("❌")) {
		t.Errorf("ожидались счётчики успехов и ошибок, получено: %s", output)
	}
}
