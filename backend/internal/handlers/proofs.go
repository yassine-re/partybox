package handlers

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/models"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/vision"
)

const proofPath = "/api/players/me/mission/proof"

func (h *Handler) registerProofRoutes(auth *gin.RouterGroup) {
	auth.GET("/players/me/mission/proof", func(c *gin.Context) {
		status, err := h.Service.MissionProofStatus(c.Request.Context(), currentPlayer(c))
		respond(c, status, err, http.StatusOK)
	})
	auth.POST("/players/me/mission/proof", func(c *gin.Context) {
		// Stream multipart parts into bounded memory; ParseMultipartForm may spill
		// photos to disk, so it must not be used here.
		reader, err := c.Request.MultipartReader()
		if err != nil {
			respond(c, nil, models.ErrInvalid, 0)
			return
		}
		var id string
		var data []byte
		seen := map[string]bool{}
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				uploadError(c, err)
				return
			}
			name := part.FormName()
			if seen[name] || (name != "assignment_id" && name != "image") {
				respond(c, nil, models.ErrInvalid, 0)
				return
			}
			seen[name] = true
			limit := int64(36)
			if name == "image" {
				limit = vision.MaxImageBytes
			}
			value, err := io.ReadAll(io.LimitReader(part, limit+1))
			if err != nil {
				uploadError(c, err)
				return
			}
			if int64(len(value)) > limit {
				uploadError(c, &http.MaxBytesError{Limit: limit})
				return
			}
			part.Close()
			if name == "image" {
				data = value
			} else {
				id = string(value)
			}
		}
		if !seen["assignment_id"] || !seen["image"] || len(data) == 0 {
			respond(c, nil, models.ErrInvalid, 0)
			return
		}
		player := currentPlayer(c)
		result, err := h.Service.SubmitMissionProof(c.Request.Context(), player, id, data)
		if err == nil && result.Completion != nil && !result.Completion.AlreadyCompleted {
			h.Realtime.Broadcast(realtime.NewEvent(realtime.EventMissionCompleted, player.GameID, player.ID))
		}
		respond(c, result, err, http.StatusOK)
	})
}

func uploadError(c *gin.Context, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": gin.H{"code": "image_too_large", "message": "Photo trop volumineuse (5 Mo maximum)."}})
		return
	}
	respond(c, nil, models.ErrInvalid, 0)
}
