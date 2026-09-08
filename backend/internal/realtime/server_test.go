package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func testRealtimeServer(t *testing.T, memberships map[string]string) (*Server, *httptest.Server) {
	t.Helper()
	rt := NewServer("", func(_ context.Context, gameID, playerID string) error {
		if memberships[playerID] != gameID {
			return errors.New("wrong game")
		}
		return nil
	})
	httpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		rt.ServeHTTP(writer, request, strings.TrimPrefix(request.URL.Path, "/ws/"))
	}))
	t.Cleanup(func() {
		rt.Close()
		httpServer.Close()
	})
	return rt, httpServer
}

func dialClient(t *testing.T, rt *Server, httpServer *httptest.Server, playerID, ticketGameID, routeGameID string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	ticket, _, err := rt.IssueTicket(playerID, ticketGameID)
	if err != nil {
		t.Fatal(err)
	}
	endpoint := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws/" + routeGameID + "?ticket=" + url.QueryEscape(ticket)
	conn, response, err := websocket.DefaultDialer.Dial(endpoint, nil)
	return conn, response, err
}

func waitForConnections(t *testing.T, hub *Hub, gameID string, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hub.connectionCount(gameID) == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("game %s: got %d connections, want %d", gameID, hub.connectionCount(gameID), want)
}

func TestWebSocketAuthorizationAndGameMembership(t *testing.T) {
	rt, httpServer := testRealtimeServer(t, map[string]string{
		"alice": "game-a",
		"bob":   "game-b",
	})

	alice, _, err := dialClient(t, rt, httpServer, "alice", "game-a", "game-a")
	if err != nil {
		t.Fatalf("authorized connection failed: %v", err)
	}
	defer alice.Close()
	waitForConnections(t, rt.hub, "game-a", 1)

	conn, response, err := dialClient(t, rt, httpServer, "bob", "game-b", "game-a")
	if conn != nil {
		conn.Close()
	}
	if response != nil {
		defer response.Body.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("cross-game ticket: err=%v status=%v, want HTTP 401", err, responseStatus(response))
	}

	conn, response, err = dialClient(t, rt, httpServer, "bob", "game-a", "game-a")
	if conn != nil {
		conn.Close()
	}
	if response != nil {
		defer response.Body.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("stale membership: err=%v status=%v, want HTTP 403", err, responseStatus(response))
	}
}

func TestBroadcastIsScopedAndReachesEveryGameClient(t *testing.T) {
	rt, httpServer := testRealtimeServer(t, map[string]string{
		"alice": "game-a",
		"bob":   "game-a",
		"chris": "game-b",
	})

	alice, _, err := dialClient(t, rt, httpServer, "alice", "game-a", "game-a")
	if err != nil {
		t.Fatal(err)
	}
	defer alice.Close()
	bob, _, err := dialClient(t, rt, httpServer, "bob", "game-a", "game-a")
	if err != nil {
		t.Fatal(err)
	}
	defer bob.Close()
	chris, _, err := dialClient(t, rt, httpServer, "chris", "game-b", "game-b")
	if err != nil {
		t.Fatal(err)
	}
	defer chris.Close()
	waitForConnections(t, rt.hub, "game-a", 2)
	waitForConnections(t, rt.hub, "game-b", 1)

	want := NewEvent(EventMissionCompleted, "game-a", "alice")
	rt.Broadcast(want)
	for name, conn := range map[string]*websocket.Conn{"alice": alice, "bob": bob} {
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		_, payload, readErr := conn.ReadMessage()
		if readErr != nil {
			t.Fatalf("%s did not receive broadcast: %v", name, readErr)
		}
		var got Event
		if err = json.Unmarshal(payload, &got); err != nil {
			t.Fatal(err)
		}
		if got.Type != want.Type || got.GameID != want.GameID || got.PlayerID != want.PlayerID {
			t.Fatalf("%s received %+v, want %+v", name, got, want)
		}
	}

	_ = chris.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	if _, _, err = chris.ReadMessage(); err == nil {
		t.Fatal("client in game-b unexpectedly received game-a event")
	}
}

func TestClientDisconnectIsUnregistered(t *testing.T) {
	rt, httpServer := testRealtimeServer(t, map[string]string{"alice": "game-a"})
	conn, _, err := dialClient(t, rt, httpServer, "alice", "game-a", "game-a")
	if err != nil {
		t.Fatal(err)
	}
	waitForConnections(t, rt.hub, "game-a", 1)
	if err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye")); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	waitForConnections(t, rt.hub, "game-a", 0)
}

func TestHubCloseClosesActiveConnections(t *testing.T) {
	rt, httpServer := testRealtimeServer(t, map[string]string{"alice": "game-a"})
	conn, _, err := dialClient(t, rt, httpServer, "alice", "game-a", "game-a")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	waitForConnections(t, rt.hub, "game-a", 1)

	rt.Close()
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err = conn.ReadMessage(); err == nil {
		t.Fatal("connection remained open after hub shutdown")
	}
	waitForConnections(t, rt.hub, "game-a", 0)
}

func responseStatus(response *http.Response) any {
	if response == nil {
		return nil
	}
	return response.StatusCode
}
