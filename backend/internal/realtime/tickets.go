package realtime

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

var ErrInvalidTicket = errors.New("ticket WebSocket invalide ou expiré")

type ticketClaims struct {
	PlayerID string
	GameID   string
	Expires  time.Time
}

// TicketStore keeps short-lived, one-time credentials in memory. Only a hash
// of each credential is retained, just like the longer-lived player tokens.
type TicketStore struct {
	mu      sync.Mutex
	tickets map[[32]byte]ticketClaims
	ttl     time.Duration
	now     func() time.Time
}

func NewTicketStore(ttl time.Duration) *TicketStore {
	return &TicketStore{
		tickets: make(map[[32]byte]ticketClaims),
		ttl:     ttl,
		now:     time.Now,
	}
}

func (s *TicketStore) Issue(playerID, gameID string) (string, time.Time, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", time.Time{}, err
	}
	ticket := base64.RawURLEncoding.EncodeToString(raw[:])
	hash := sha256.Sum256([]byte(ticket))
	expires := s.now().Add(s.ttl)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeExpiredLocked(s.now())
	s.tickets[hash] = ticketClaims{PlayerID: playerID, GameID: gameID, Expires: expires}
	return ticket, expires, nil
}

func (s *TicketStore) Consume(ticket, gameID string) (ticketClaims, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(ticket)
	if err != nil || len(decoded) != 32 {
		return ticketClaims{}, ErrInvalidTicket
	}
	hash := sha256.Sum256([]byte(ticket))
	now := s.now()

	s.mu.Lock()
	defer s.mu.Unlock()
	claims, found := s.tickets[hash]
	delete(s.tickets, hash) // Every presented ticket is single-use.
	if !found || !now.Before(claims.Expires) || claims.GameID != gameID {
		return ticketClaims{}, ErrInvalidTicket
	}
	return claims, nil
}

func (s *TicketStore) removeExpiredLocked(now time.Time) {
	for hash, claims := range s.tickets {
		if !now.Before(claims.Expires) {
			delete(s.tickets, hash)
		}
	}
}
