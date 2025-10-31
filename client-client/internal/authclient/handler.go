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

func (h *Handler) FinishAuth(w http.ResponseWriter, r *http.Request) {
	var req AuthFinishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("bad request: ", err)
		utils.WriteJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	_, err := h.service.FinishAuth(req.Login, req.MultiHash, req.MultiHash, req.MultiHash)
	if err != nil {
		log.Println("finish auth err: ", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "finish auth err"+err.Error())
		return
	}
}
