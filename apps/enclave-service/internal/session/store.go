package session

import (
	"sync"
	"time"
)

type SessionEntry struct {
	Key       []byte // session AEAD key, 32 bytes
	ExpiresAt time.Time
}

var (
	sessions   = make(map[string]*SessionEntry)
	mu sync.RWMutex
)

func StoreSession(id string, key []byte, expiresAt time.Time) {
	mu.Lock()
	defer mu.Unlock()
	sessions[id] = &SessionEntry{
		Key:       key,
		ExpiresAt: expiresAt,
	}
}

func GetSession(id string) (*SessionEntry, bool) {
	mu.RLock()
	defer mu.RUnlock()
	session, exists := sessions[id]
	if !exists || time.Now().After(session.ExpiresAt) {
		return nil, false
	}
	return session, true
}

func DeleteSession(id string) {
	mu.Lock()
	defer mu.Unlock()
	delete(sessions, id)
}
