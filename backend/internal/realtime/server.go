package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type AuthorizeFunc func(context.Context, string, string) error

type Server struct {
	hub       *Hub
	tickets   *TicketStore
	authorize AuthorizeFunc
	upgrader  websocket.Upgrader
}

func NewServer(frontendURL string, authorize AuthorizeFunc) *Server {
	return &Server{
		hub:       NewHub(),
		tickets:   NewTicketStore(30 * time.Second),
		authorize: authorize,
		upgrader: websocket.Upgrader{
			HandshakeTimeout: 5 * time.Second,
			CheckOrigin:      allowedOrigin(frontendURL),
		},
	}
}

func allowedOrigin(frontendURL string) func(*http.Request) bool {
	configured, _ := url.Parse(strings.TrimSuffix(frontendURL, "/"))
	return func(request *http.Request) bool {
		origin := request.Header.Get("Origin")
		if origin == "" {
			return true
		}
		parsed, err := url.Parse(origin)
		if err != nil {
			return false
		}
		return parsed.Host == request.Host ||
			(configured != nil && parsed.Scheme == configured.Scheme && parsed.Host == configured.Host)
	}
}

func (s *Server) IssueTicket(playerID, gameID string) (string, time.Time, error) {
	return s.tickets.Issue(playerID, gameID)
}

func (s *Server) Broadcast(event Event) {
	s.hub.Broadcast(event)
}

func (s *Server) Close() {
	s.hub.Close()
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request, gameID string) {
	if request.Method != http.MethodGet {
		writeJSONError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "Méthode non autorisée.")
		return
	}
	claims, err := s.tickets.Consume(request.URL.Query().Get("ticket"), gameID)
	if err != nil {
		writeJSONError(writer, http.StatusUnauthorized, "invalid_ticket", ErrInvalidTicket.Error())
		return
	}
	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()
	if err = s.authorize(ctx, gameID, claims.PlayerID); err != nil {
		writeJSONError(writer, http.StatusForbidden, "forbidden", "Ce joueur n’appartient pas à cette partie.")
		return
	}
	conn, err := s.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	newClient(gameID, claims.PlayerID, conn).run(s.hub)
}

func writeJSONError(writer http.ResponseWriter, status int, code, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}
