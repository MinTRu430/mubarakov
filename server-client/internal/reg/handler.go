package reg

import (
	"encoding/json"
	"log"
	"net/http"
	"server-client/internal/utils"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	TgLogin  string `json:"tg_login"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Println("Method not allowed:", r.Method)
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "Only POST allowed")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("Invalid JSON in request body:", err)
		utils.WriteJSONError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	id, err := h.service.RegisterUser(r.Context(), req.Login, req.Password, req.TgLogin)
	if err != nil {
		log.Println("Failed to register user:", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "DB error: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"userId": id,
	})
}
