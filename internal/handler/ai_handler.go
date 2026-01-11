package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/wealthpath/backend/internal/service"
)

type AIHandler struct {
	aiService *service.AIService
}

func NewAIHandler(aiService *service.AIService) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// Chat godoc
// @Summary AI financial assistant chat
// @Description Send a message to the AI financial assistant and receive personalized advice
// @Tags ai
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body service.ChatRequest true "Chat message"
// @Success 200 {object} service.ChatResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /chat [post]
func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Message == "" {
		respondError(w, http.StatusBadRequest, "message is required")
		return
	}

	resp, err := h.aiService.Chat(r.Context(), userID, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to process message: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}
