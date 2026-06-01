package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type RegistrationData struct {
	Name         string
	Username     string
	Email        string
	BusinessType string
	Country      string
	Currency     string
	GoogleID     string
	FacebookID   string
	AvatarURL    string
	CreatedAt    time.Time
}

type RegistrationStore struct {
	mu    sync.RWMutex
	store map[string]*RegistrationData
}

var RegStore = &RegistrationStore{
	store: make(map[string]*RegistrationData),
}

func init() {
	go RegStore.cleanup()
}

func (rs *RegistrationStore) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		rs.mu.Lock()
		for token, data := range rs.store {
			if time.Since(data.CreatedAt) > 30*time.Minute {
				delete(rs.store, token)
			}
		}
		rs.mu.Unlock()
	}
}

func (rs *RegistrationStore) Save(data *RegistrationData) string {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)

	data.CreatedAt = time.Now()
	rs.store[token] = data
	return token
}

func (rs *RegistrationStore) Get(token string) (*RegistrationData, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	data, ok := rs.store[token]
	return data, ok
}

func (rs *RegistrationStore) Delete(token string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	delete(rs.store, token)
}
