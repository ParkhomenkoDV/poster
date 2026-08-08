package progress

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Show отображает прогресс в реальном времени
func Show(
	totalItems, totalSuccess, totalErrors *int64,
	total int64,
	done chan struct{},
) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			req := atomic.LoadInt64(totalItems)
			succ := atomic.LoadInt64(totalSuccess)
			errs := atomic.LoadInt64(totalErrors)

			if req > 0 {
				percent := float64(req/total) * 100
				fmt.Printf("\r⏳ Progress: %d/%d (%.1f%%) | ✅ %d | ❌ %d",
					req, total, percent, succ, errs)
			}
		}
	}
}
