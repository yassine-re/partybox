package realtime

import (
	"errors"
	"testing"
	"time"
)

func TestTicketIsBoundShortLivedAndSingleUse(t *testing.T) {
	now := time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC)
	store := NewTicketStore(30 * time.Second)
	store.now = func() time.Time { return now }

	ticket, expires, err := store.Issue("alice", "game-a")
	if err != nil {
		t.Fatal(err)
	}
	if expires != now.Add(30*time.Second) {
		t.Fatalf("expires=%v, want %v", expires, now.Add(30*time.Second))
	}
	claims, err := store.Consume(ticket, "game-a")
	if err != nil || claims.PlayerID != "alice" {
		t.Fatalf("first use: claims=%+v err=%v", claims, err)
	}
	if _, err = store.Consume(ticket, "game-a"); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("second use err=%v, want ErrInvalidTicket", err)
	}

	wrongGameTicket, _, err := store.Issue("alice", "game-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Consume(wrongGameTicket, "game-b"); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("cross-game use err=%v, want ErrInvalidTicket", err)
	}

	expiredTicket, _, err := store.Issue("alice", "game-a")
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(30 * time.Second)
	if _, err = store.Consume(expiredTicket, "game-a"); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("expired use err=%v, want ErrInvalidTicket", err)
	}
}
