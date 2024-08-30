package util

import "time"

func RunEvery(t time.Duration, f func()) {
	ticker := time.NewTicker(t)
	for range ticker.C {
		f()
	}
}
