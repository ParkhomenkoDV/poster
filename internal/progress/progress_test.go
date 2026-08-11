package progress

import (
	"bufio"
	"bytes"
	"sync/atomic"
	"testing"
	"time"
)

// contains проверяет, содержит ли строка s подстроку substr (без учёта escape-последовательностей).
// Для простоты используем strings.Contains; escape-последовательность \033[2K не мешает.
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

// TestShow проверяет, что Show обновляет вывод и завершается по контексту.

// TestPrintProgress проверяет вывод printProgress для разных конфигураций.
func TestPrintProgress(t *testing.T) {
	tests := []struct {
		name       string
		bar        *Bar
		itemsVal   uint64
		successVal uint64
		errorsVal  uint64
		prevItems  uint64
		prevTime   time.Time
		wantSubstr []string // ожидаемые подстроки в выводе
	}{
		{
			name: "без Total, без доп. опций",
			bar: &Bar{
				Total:      0,
				ShowSpeed:  false,
				ShowETA:    false,
				ShowErrors: false,
			},
			itemsVal:   42,
			successVal: 0,
			errorsVal:  0,
			prevItems:  0,
			prevTime:   time.Now().Add(-time.Second),
			wantSubstr: []string{"⏳ Обработано: 42"},
		},
		{
			name: "с Total и процентами",
			bar: &Bar{
				Total:      100,
				ShowSpeed:  false,
				ShowETA:    false,
				ShowErrors: false,
			},
			itemsVal:   30,
			successVal: 0,
			errorsVal:  0,
			prevItems:  20,
			prevTime:   time.Now().Add(-time.Second),
			wantSubstr: []string{"30 / 100", "30.0%"},
		},
		{
			name: "со скоростью",
			bar: &Bar{
				Total:      100,
				ShowSpeed:  true,
				ShowETA:    false,
				ShowErrors: false,
			},
			itemsVal:   50,
			successVal: 0,
			errorsVal:  0,
			prevItems:  40,
			prevTime:   time.Now().Add(-500 * time.Millisecond), // разница 0.5 сек -> 20 шт/с
			wantSubstr: []string{"50 / 100", "50.0%", "шт/с"},
		},
		{
			name: "с ETA",
			bar: &Bar{
				Total:      100,
				ShowSpeed:  false,
				ShowETA:    true,
				ShowErrors: false,
			},
			itemsVal:   60,
			successVal: 0,
			errorsVal:  0,
			prevItems:  50,
			prevTime:   time.Now().Add(-2 * time.Second), // 5 шт/с, осталось 40 -> 8 сек
			wantSubstr: []string{"60 / 100", "60.0%", "ETA:"},
		},
		{
			name: "с ошибками и успехами",
			bar: &Bar{
				Total:      100,
				ShowSpeed:  false,
				ShowETA:    false,
				ShowErrors: true,
			},
			itemsVal:   70,
			successVal: 65,
			errorsVal:  5,
			prevItems:  60,
			prevTime:   time.Now().Add(-time.Second),
			wantSubstr: []string{"70 / 100", "70.0%", "✅ 65", "❌ 5"},
		},
		{
			name: "все опции включены",
			bar: &Bar{
				Total:      200,
				ShowSpeed:  true,
				ShowETA:    true,
				ShowErrors: true,
			},
			itemsVal:   120,
			successVal: 110,
			errorsVal:  10,
			prevItems:  100,
			prevTime:   time.Now().Add(-time.Second),
			wantSubstr: []string{"120 / 200", "60.0%", "шт/с", "ETA:", "✅ 110", "❌ 10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var items, success, errors uint64
			atomic.StoreUint64(&items, tt.itemsVal)
			atomic.StoreUint64(&success, tt.successVal)
			atomic.StoreUint64(&errors, tt.errorsVal)

			var buf bytes.Buffer
			bw := bufio.NewWriter(&buf)

			// Вызываем printProgress с подготовленными параметрами
			tt.bar.printProgress(bw, &items, &success, &errors, tt.prevItems, tt.prevTime)

			// Принудительный flush (printProgress уже вызывает Flush, но на всякий случай)
			bw.Flush()

			output := buf.String()

			// Проверяем наличие всех ожидаемых подстрок
			for _, sub := range tt.wantSubstr {
				if !contains(output, sub) {
					t.Errorf("вывод не содержит %q: %q", sub, output)
				}
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		dur  time.Duration
		want string
	}{
		{"нулевая", 0, "0h0m0s"},
		{"только секунды", 5 * time.Second, "0h0m5s"},
		{"только минуты", 2 * time.Minute, "0h2m0s"},
		{"только часы", 3 * time.Hour, "3h0m0s"},
		{"часы+минуты", 2*time.Hour + 30*time.Minute, "2h30m0s"},
		{"часы+минуты+секунды", 1*time.Hour + 20*time.Minute + 15*time.Second, "1h20m15s"},
		{"округление вверх", 1*time.Hour + 20*time.Minute + 15*time.Second + 500*time.Millisecond, "1h20m16s"},
		{"округление вниз", 1*time.Hour + 20*time.Minute + 15*time.Second + 200*time.Millisecond, "1h20m15s"},
		{"большая длительность", 100 * time.Hour, "100h0m0s"},
		{"отрицательная (только часы)", -1 * time.Hour, "-1h0m0s"},
		{"отрицательная (часы+минуты)", -2*time.Hour - 30*time.Minute, "-2h30m0s"},
		{"отрицательная с секундами", -1*time.Hour - 2*time.Minute - 3*time.Second, "-1h2m3s"},
		{"отрицательная с округлением", -1*time.Hour - 2*time.Minute - 3*time.Second - 600*time.Millisecond, "-1h2m4s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.dur)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.dur, got, tt.want)
			}
		})
	}
}
