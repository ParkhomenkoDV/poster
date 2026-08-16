package progress

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Bar struct {
	Interval    time.Duration // Частота обновления
	Description string        // Описание
	Length      uint8         // Длина окна (0 - не отображать)
	Total       uint64        // Общее количество единиц работы (0 – неизвестно)
	ShowETA     bool          // Показывать оценочное время до завершения
	ShowSpeed   bool          // Показывать скорость обработки (шт/сек)
	Leave       bool          // Оставить прогресс после завершения
}

func New(
	interval time.Duration,
	description string,
	length uint8,
	total uint64,
	showETA, showSpeed, leave bool,
) *Bar {
	return &Bar{
		Interval:    interval.Abs(),
		Description: description,
		Length:      length,
		Total:       total,
		ShowETA:     showETA,
		ShowSpeed:   showSpeed,
		Leave:       leave,
	}
}

// Start запускает прогресс-бар в фоне.
// Возвращает функцию stop, которую нужно вызвать по окончании работы.
func (b *Bar) Start(ctx context.Context, done, errors *uint64) (stop func()) {
	ctx, cancel := context.WithCancel(ctx) // контекст отмены

	var wg sync.WaitGroup // ждун
	wg.Add(1)

	go func() {
		defer wg.Done()
		b.Show(ctx, done, errors)
	}()

	return func() {
		cancel()
		wg.Wait()
	}
}

// Show запускает периодический вывод прогресса выполнения.
// Параметры:
//   - ctx    – контекст для управления завершением.
//   - done   – указатель на атомарный счётчик обработанных элементов (не должен быть nil).
//   - errors – указатель на атомарный счётчик ошибок (может быть nil, тогда ошибки не выводятся).
func (b *Bar) Show(ctx context.Context, done, errors *uint64) {
	ticker := time.NewTicker(b.Interval)
	defer ticker.Stop()

	prevDone := atomic.LoadUint64(done)
	prevTime := time.Now()

	defer func() {
		if !b.Leave {
			fmt.Fprint(os.Stdout, "\033[2K\r") // стираем строку
		} else {
			b.print(done, errors, prevDone, prevTime) // выводим заполненный прогресс
			fmt.Fprint(os.Stdout, "\n")               // переходим на новую строку
		}
	}()

	// Выводим прогресс
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.print(done, errors, prevDone, prevTime)
			// обновляем предыдущие значения после вывода
			prevDone = atomic.LoadUint64(done)
			prevTime = time.Now()
		}
	}
}

// print формирует и выводит строку прогресса.
func (b *Bar) print(done, errors *uint64, prevDone uint64, prevTime time.Time) {
	now := time.Now()
	dn := atomic.LoadUint64(done)

	// Очищаем текущую строку перед выводом, чтобы избежать артефактов.
	var line string = fmt.Sprintf("\033[2K\r%s ", b.Description)

	if b.Total > 0 {
		percent := float64(dn) / float64(b.Total)
		line += fmt.Sprintf("%3.0f%% %s %d/%d ", percent*100, b.getLoad(percent), dn, b.Total)
	} else {
		line += fmt.Sprintf("%d ", dn)
	}

	// Счётчик ошибок
	if errors != nil {
		line += fmt.Sprintf("❌ %d ", atomic.LoadUint64(errors))
	}

	// ETA (оценочное время до завершения)
	if b.ShowETA && b.Total > 0 && dn > 0 && dn < b.Total {
		elapsed := now.Sub(prevTime).Seconds()
		if elapsed > 0 && dn > prevDone {
			rate := float64(dn-prevDone) / elapsed
			if rate > 0 {
				remaining := float64(b.Total-dn) / rate
				line += fmt.Sprintf("⏰ %s ", formatDuration(time.Duration(remaining*float64(time.Second))))
			}
		}
	}

	// Скорость (it/sec)
	if b.ShowSpeed {
		elapsed := now.Sub(prevTime).Seconds()
		if elapsed > 0 && dn > prevDone {
			speed := float64(dn-prevDone) / elapsed
			line += fmt.Sprintf("⚡️ %.1f it/s ", speed)
		}
	}

	fmt.Fprint(os.Stdout, line)
}

// getLoad - получение линии загрузки.
func (b *Bar) getLoad(percent float64) string {
	if b.Length == 0 {
		return ""
	}
	done := int(percent * float64(b.Length))
	extra := int(b.Length) - done
	return "|" + strings.Repeat("-", done) + strings.Repeat(" ", extra) + "|"
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
