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

	id, privateKey, err := h.service.RegisterUser(r.Context(), req.Login, req.Password, req.TgLogin)
	if err != nil {
		log.Println("Failed to register user:", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "DB error: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "ok",
		"userId":      id,
		"private_key": privateKey,
	})
}

type RegenerateRSARequest struct {
	Login string `json:"login"`
}

func (h *Handler) RegenerateRSA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "Only POST allowed")
		return
	}

	var req RegenerateRSARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("Invalid JSON in RSA regenerate body:", err)
		utils.WriteJSONError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Login == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, "login is required")
		return
	}

	privateKey, err := h.service.RegenerateRSAForUser(r.Context(), req.Login)
	if err != nil {
		log.Println("Failed to regenerate RSA:", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "regenerate rsa error: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "ok",
		"login":       req.Login,
		"private_key": privateKey,
	})
}
