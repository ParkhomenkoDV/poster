package progress

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
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
func (b *Bar) Show(ctx context.Context, items, success, errors *uint64) {
	ticker := time.NewTicker(b.Interval)
	defer ticker.Stop()

	// Буферизованный вывод снижает число системных вызовов.
	bw := bufio.NewWriter(os.Stdout)
	defer bw.Flush()

	var (
		prevItems uint64
		prevTime  = time.Now()
	)

	// Выводим прогресс
	for {
		select {
		case <-ctx.Done():
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

	fmt.Fprint(bw, line) // Запись в буферизованный writer.
	bw.Flush()           // Немедленный вывод
}

// formatDuration форматирует длительность в удобочитаемый вид.
func formatDuration(dur time.Duration) string {
	totalSec := int64(dur.Round(time.Second) / time.Second)
	neg := totalSec < 0
	if neg {
		totalSec = -totalSec
	}

	var (
		h = totalSec / 3600        // часы
		m = (totalSec % 3600) / 60 // минуты
		s = totalSec % 60          // секунды
	)

	// Буфер достаточного размера: знак + до 10 цифр часов + 'h' + 2 цифры минут + 'm' + 2 цифры секунд + 's' = 18
	buf := make([]byte, 0, 20)
	if neg {
		buf = append(buf, '-')
	}
	buf = strconv.AppendInt(buf, h, 10)
	buf = append(buf, 'h')
	buf = strconv.AppendInt(buf, m, 10)
	buf = append(buf, 'm')
	buf = strconv.AppendInt(buf, s, 10)
	buf = append(buf, 's')
	return string(buf)
}
