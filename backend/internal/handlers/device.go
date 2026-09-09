package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/services"
)

func (h *Handler) registerDeviceRoutes(device *gin.RouterGroup) {
	device.POST("/heartbeat", func(c *gin.Context) {
		var body services.HeartbeatInput
		if !bind(c, &body) {
			return
		}
		serverTime, err := h.Service.DeviceHeartbeat(c.Request.Context(), c.Param("boxId"), body)
		respond(c, gin.H{"status": "online", "server_time": serverTime}, err, http.StatusOK)
	})
	device.GET("/commands", func(c *gin.Context) {
		commands, err := h.Service.DeviceCommands(c.Request.Context(), c.Param("boxId"))
		respond(c, gin.H{"commands": commands}, err, http.StatusOK)
	})
	device.POST("/commands/:commandId/ack", func(c *gin.Context) {
		gameID, changed, err := h.Service.AcknowledgeDeviceCommand(
			c.Request.Context(), c.Param("boxId"), c.Param("commandId"),
		)
		if err == nil && changed {
			h.Realtime.Broadcast(realtime.NewEvent(realtime.EventReactionChallengeChanged, gameID, ""))
		}
		respond(c, gin.H{"acknowledged": true}, err, http.StatusOK)
	})
	device.POST("/reaction-results", func(c *gin.Context) {
		var body services.DeviceReactionResult
		if !bind(c, &body) {
			return
		}
		result, err := h.Service.SubmitReactionResult(c.Request.Context(), c.Param("boxId"), body)
		if err == nil && !result.AlreadyProcessed {
			h.Realtime.Broadcast(realtime.NewEvent(
				realtime.EventReactionChallengeResolved, result.Challenge.GameID, "",
			))
		}
		respond(c, result, err, http.StatusOK)
	})
}
