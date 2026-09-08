package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/realtime"
)

func (h *Handler) registerGameRoutes(public, auth *gin.RouterGroup) {
	public.POST("/games/:gameId/join", func(c *gin.Context) {
		var body struct {
			Name string `json:"name"`
		}
		if !bind(c, &body) {
			return
		}
		session, err := h.Service.Join(c.Request.Context(), c.Param("gameId"), body.Name)
		if err == nil {
			h.Realtime.Broadcast(realtime.NewEvent(
				realtime.EventPlayerJoined, session.Game.ID, session.Player.ID,
			))
		}
		respond(c, session, err, http.StatusCreated)
	})
	auth.GET("/games/:gameId", func(c *gin.Context) {
		game, err := h.Service.Game(c.Request.Context(), c.Param("gameId"), currentPlayer(c))
		respond(c, game, err, http.StatusOK)
	})
	auth.GET("/games/:gameId/leaderboard", func(c *gin.Context) {
		game, err := h.Service.Game(c.Request.Context(), c.Param("gameId"), currentPlayer(c))
		respond(c, gin.H{"players": game.Players}, err, http.StatusOK)
	})
	auth.POST("/games/:gameId/start", func(c *gin.Context) {
		gameID := c.Param("gameId")
		err := h.Service.Start(c.Request.Context(), gameID, currentPlayer(c))
		if err == nil {
			h.Realtime.Broadcast(realtime.NewEvent(realtime.EventGameStarted, gameID, ""))
		}
		respond(c, gin.H{"status": "playing"}, err, http.StatusOK)
	})
	auth.POST("/games/:gameId/end", func(c *gin.Context) {
		gameID := c.Param("gameId")
		err := h.Service.End(c.Request.Context(), gameID, currentPlayer(c))
		if err == nil {
			h.Realtime.Broadcast(realtime.NewEvent(realtime.EventGameEnded, gameID, ""))
		}
		respond(c, gin.H{"status": "ended"}, err, http.StatusOK)
	})
}
