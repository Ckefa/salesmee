package business

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type PendingProfileChange struct {
	BusinessID  uint
	Token       string
	Name        string
	Username    string
	Email       string
	Password    string
	Logo        string
	Country     string
	Currency    string
	OTPCode     string
	OTPExpiresAt time.Time
	ExpiresAt   time.Time
}

type ProfileChangeStore struct {
	mu    sync.RWMutex
	store map[string]*PendingProfileChange
}

var profileChangeStore = &ProfileChangeStore{
	store: make(map[string]*PendingProfileChange),
}

func init() {
	go profileChangeStore.cleanup()
}

func (s *ProfileChangeStore) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		s.mu.Lock()
		for token, data := range s.store {
			if time.Since(data.ExpiresAt) > 0 {
				delete(s.store, token)
			}
		}
		s.mu.Unlock()
	}
}

func (s *ProfileChangeStore) Save(data *PendingProfileChange) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := data.Token
	if token == "" {
		b := make([]byte, 16)
		rand.Read(b)
		token = hex.EncodeToString(b)
		data.Token = token
	}

	data.ExpiresAt = time.Now().Add(15 * time.Minute)
	s.store[token] = data
	return token
}

func (s *ProfileChangeStore) Get(token string) (*PendingProfileChange, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.store[token]
	if !ok {
		return nil, false
	}
	if time.Since(data.ExpiresAt) > 0 {
		return nil, false
	}
	return data, true
}

func (s *ProfileChangeStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.store, token)
}
