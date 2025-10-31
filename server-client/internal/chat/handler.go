package chat

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"server-client/internal/utils"
	"time"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type ChatStartRequest struct {
	Login string `json:"login"`
	B     string `json:"B"`
}

type ChatStartResponse struct {
	Message string `json:"message"`
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "only POST allowed")
		return
	}

	var req ChatStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	K, err := h.service.ComputeSharedKey(ctx, req.Login, req.B)
	if err != nil {
		log.Println("failed to compute shared key:", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to compute shared key")
		return
	}
	log.Println("Login:", req.Login)
	log.Println("BBB:", req.B)
	log.Println("KKK:", K)
	resp := ChatStartResponse{Message: "по сути остался чат и интерфейс его сделаю через две недели😎)"}
	json.NewEncoder(w).Encode(resp)
}
