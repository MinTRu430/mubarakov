package authclient

import (
	"client-client/internal/utils"
	"encoding/json"
	"log"
	"net/http"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) StartAuth(w http.ResponseWriter, r *http.Request) {
	var req AuthStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("bad request: ", err)
		utils.WriteJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	codeWord, err := h.service.StartAuth(req.Login)
	if err != nil {
		log.Println("start auth err: ", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "start auth err"+err.Error())
		return
	}

	json.NewEncoder(w).Encode(AuthStartResponse{CodeWord: codeWord})
}

type FinishAuthWebRequest struct {
	Login      string `json:"login"`
	Password   string `json:"password"`
	TgCode     string `json:"tg_code"`
	CodeWord   string `json:"code_word"`
	PrivateKey string `json:"private_key"`
}

type FinishAuthWebResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func (h *Handler) FinishAuthWeb(w http.ResponseWriter, r *http.Request) {
	var req FinishAuthWebRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("bad request: ", err)
		utils.WriteJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	if req.Login == "" || req.Password == "" || req.TgCode == "" || req.CodeWord == "" || req.PrivateKey == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, "login, password, tg_code, code_word, private_key required")
		return
	}

	message, _, err := h.service.FinishAuthWithRSA(req.Login, req.Password, req.CodeWord, req.TgCode, req.PrivateKey)
	if err != nil {
		log.Println("finish auth web err: ", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "finish auth err: "+err.Error())
		return
	}

	json.NewEncoder(w).Encode(FinishAuthWebResponse{
		OK:      true,
		Message: message,
	})
}
