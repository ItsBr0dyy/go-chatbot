package loop

import "time"

func Every(interval time.Duration, fn func()) {
	go func() {
		fn()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			fn()
		}
	}()
}
