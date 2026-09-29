package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/realtime"
)

func (h *Handler) registerGameRoutes(public, auth *gin.RouterGroup) {
	public.POST("/games/:gameId/join", func(c *gin.Context) {
		if ended, err := h.Service.FinishDueGame(c.Request.Context(), c.Param("gameId")); err != nil {
			respond(c, nil, err, 0)
			return
		} else if ended {
			h.Realtime.Broadcast(realtime.NewEvent(realtime.EventGameEnded, c.Param("gameId"), ""))
		}
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
	auth.GET("/games/:gameId/validations", func(c *gin.Context) {
		requests, err := h.Service.PendingValidations(c.Request.Context(), c.Param("gameId"), currentPlayer(c))
		respond(c, gin.H{"requests": requests}, err, http.StatusOK)
	})
	auth.POST("/games/:gameId/validations/:requestId/resolve", func(c *gin.Context) {
		var body struct {
			Approved bool `json:"approved"`
		}
		if !bind(c, &body) {
			return
		}
		gameID := c.Param("gameId")
		result, err := h.Service.ResolveValidation(c.Request.Context(), gameID, c.Param("requestId"), currentPlayer(c), body.Approved)
		if err == nil {
			h.Realtime.Broadcast(realtime.NewEvent(realtime.EventValidationResolved, gameID, result.Player.ID))
			if body.Approved {
				h.Realtime.Broadcast(realtime.NewEvent(realtime.EventMissionCompleted, gameID, result.Player.ID))
				h.Realtime.Broadcast(realtime.NewEvent(realtime.EventLeaderboardChanged, gameID, ""))
				h.Realtime.Broadcast(realtime.NewEvent(realtime.EventChaosChanged, gameID, ""))
			}
		}
		respond(c, result, err, http.StatusOK)
	})
	auth.GET("/games/:gameId/recap", func(c *gin.Context) {
		recap, err := h.Service.FinalRecap(c.Request.Context(), c.Param("gameId"), currentPlayer(c))
		respond(c, recap, err, http.StatusOK)
	})
}
