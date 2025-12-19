package session

import "time"

func StartSessionCleanup(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			cleanupExpired()
		}
	}()
}

func cleanupExpired() {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now()
	for id, session := range sessions {
		if now.After(session.ExpiresAt) {
			delete(sessions, id)
		}
	}
}
