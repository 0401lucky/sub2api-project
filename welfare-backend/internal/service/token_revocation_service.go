package service

import (
	"strings"
	"sync"
	"time"
)

type TokenRevocationService struct {
	mu      sync.Mutex
	revoked map[string]time.Time
}

func NewTokenRevocationService() *TokenRevocationService {
	return &TokenRevocationService{
		revoked: make(map[string]time.Time),
	}
}

func (s *TokenRevocationService) Revoke(tokenID string, expiresAt time.Time) {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(time.Now())
	s.revoked[tokenID] = expiresAt
}

func (s *TokenRevocationService) IsRevoked(tokenID string) bool {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return false
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	exp, ok := s.revoked[tokenID]
	if !ok {
		return false
	}
	return exp.After(now)
}

func (s *TokenRevocationService) cleanupLocked(now time.Time) {
	for tokenID, exp := range s.revoked {
		if !exp.After(now) {
			delete(s.revoked, tokenID)
		}
	}
}
