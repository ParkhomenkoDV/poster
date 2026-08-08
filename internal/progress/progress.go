package progress

import (
	"bufio"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

type Bar struct {
	Total      uint64        // общее количество единиц работы (0 – неизвестно)
	Interval   time.Duration // частота обновления
	ShowSpeed  bool          // показывать скорость обработки (шт/сек)
	ShowETA    bool          // показывать оценочное время до завершения
	ShowErrors bool          // показывать счётчик ошибок (если передан)
}

func New(total uint64, interval time.Duration, showSpeed, showETA, showErrors bool) *Bar {
	return &Bar{
		Total:      total,
		Interval:   interval.Abs(),
		ShowSpeed:  showSpeed,
		ShowETA:    showETA,
		ShowErrors: showErrors,
	}
}

// Show отображает прогресс в реальном времени.
// Параметры:
//   - items   – указатель на атомарный счётчик обработанных элементов
//   - success – указатель на атомарный счётчик успешных операций (может быть nil)
//   - errors  – указатель на атомарный счётчик ошибок (может быть nil)
//   - done    – канал, закрытие которого останавливает вывод
func (b *Bar) Show(items, success, errors *uint64, done <-chan struct{}) {
	ticker := time.NewTicker(b.Interval)
	defer ticker.Stop()

	// Буферизованный вывод снижает число системных вызовов.
	bw := bufio.NewWriter(os.Stdout)
	defer bw.Flush()

	var (
		prevItems uint64
		prevTime  = time.Now()
	)

	// Выводим прогресс; если канал done закрыт – выводим финальную строку и выходим.
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			b.printProgress(bw, items, success, errors, prevItems, prevTime)
			// обновляем предыдущие значения после вывода
			prevItems = atomic.LoadUint64(items)
			prevTime = time.Now()
		}
	}
}

// printProgress формирует и выводит строку прогресса.
func (b *Bar) printProgress(bw *bufio.Writer, items, success, errors *uint64, prevItems uint64, prevTime time.Time) {
	now := time.Now()
	req := atomic.LoadUint64(items)
	succ, errs := uint64(0), uint64(0)

	if success != nil {
		succ = atomic.LoadUint64(success)
	}
	if errors != nil {
		errs = atomic.LoadUint64(errors)
	}

	// Строим строку прогресса с помощью strings.Builder (эффективно).
	var line string
	if b.Total > 0 {
		percent := float64(req) / float64(b.Total) * 100
		line = fmt.Sprintf("\r⏳ %d / %d (%.1f%%)", req, b.Total, percent)
	} else {
		line = fmt.Sprintf("\r⏳ Обработано: %d", req)
	}

	// Скорость (items/sec)
	if b.ShowSpeed {
		elapsed := now.Sub(prevTime).Seconds()
		if elapsed > 0 && req > prevItems {
			speed := float64(req-prevItems) / elapsed
			line += fmt.Sprintf(" | %.1f шт/с", speed)
		}
	}

	// ETA (оценочное время до завершения)
	if b.ShowETA && b.Total > 0 && req > 0 && req < b.Total {
		elapsed := now.Sub(prevTime).Seconds()
		if elapsed > 0 && req > prevItems {
			rate := float64(req-prevItems) / elapsed
			if rate > 0 {
				remaining := float64(b.Total-req) / rate
				line += fmt.Sprintf(" | ETA: %s", formatDuration(time.Duration(remaining*float64(time.Second))))
			}
		}
	}

	// Счётчики успехов и ошибок
	if b.ShowErrors && errors != nil {
		line += fmt.Sprintf(" | ✅ %d ❌ %d", succ, errs)
	} else if success != nil {
		line += fmt.Sprintf(" | ✅ %d", succ)
	}

	// Очищаем текущую строку перед выводом, чтобы избежать артефактов.
	line = "\033[2K" + line

	// Запись в буферизованный writer.
	fmt.Fprint(bw, line)
	bw.Flush() // немедленный вывод, чтобы пользователь видел прогресс
}

// formatDuration форматирует длительность в удобочитаемый вид (например, "2m15s").
func formatDuration(dur time.Duration) string {
	dur = dur.Round(time.Second)
	h := dur / time.Hour
	dur -= h * time.Hour
	m := dur / time.Minute
	dur -= m * time.Minute
	s := dur / time.Second

	return fmt.Sprintf("%dh%dm%ds", h, m, s)
}
