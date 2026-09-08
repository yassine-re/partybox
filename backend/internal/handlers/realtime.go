package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/models"
)

func (h *Handler) registerRealtimeRoutes(public, auth *gin.RouterGroup) {
	auth.POST("/games/:gameId/ws-ticket", func(c *gin.Context) {
		player := currentPlayer(c)
		gameID := c.Param("gameId")
		if player.GameID != gameID {
			respond(c, nil, models.ErrForbidden, 0)
			return
		}
		ticket, expiresAt, err := h.Realtime.IssueTicket(player.ID, gameID)
		respond(c, gin.H{"ticket": ticket, "expires_at": expiresAt}, err, http.StatusCreated)
	})
	public.GET("/games/:gameId/ws", func(c *gin.Context) {
		h.Realtime.ServeHTTP(c.Writer, c.Request, c.Param("gameId"))
	})
}

func isWebSocketRequest(request *http.Request) bool {
	return request.Method == http.MethodGet &&
		strings.HasPrefix(request.URL.Path, "/api/games/") &&
		strings.HasSuffix(request.URL.Path, "/ws")
}
