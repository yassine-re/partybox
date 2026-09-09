package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/models"
)

func (h *Handler) registerFeedbackRoutes(auth *gin.RouterGroup) {
	auth.PUT("/players/me/missions/:assignmentId/feedback", func(c *gin.Context) {
		var body struct {
			Rating *int `json:"rating"`
		}
		if !bind(c, &body) {
			return
		}
		if body.Rating == nil {
			respond(c, nil, models.ErrInvalid, 0)
			return
		}
		assignmentID := c.Param("assignmentId")
		err := h.Service.SetMissionFeedback(c.Request.Context(), currentPlayer(c), assignmentID, *body.Rating)
		respond(c, gin.H{"assignment_id": assignmentID, "rating": *body.Rating}, err, http.StatusOK)
	})
}
