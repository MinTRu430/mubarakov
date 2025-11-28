package voteclient

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ====== /vote/info (HTTP для браузера) ======
// Просто проксирует GetElectionInfo к серверу и отдаёт JSON.

func (h *Handler) Info(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET allowed", http.StatusMethodNotAllowed)
		return
	}

	eidStr := r.URL.Query().Get("election_id")
	if eidStr == "" {
		http.Error(w, "election_id required", http.StatusBadRequest)
		return
	}
	eid, err := strconv.Atoi(eidStr)
	if err != nil || eid <= 0 {
		http.Error(w, "invalid election_id", http.StatusBadRequest)
		return
	}

	info, err := h.service.GetElectionInfo(eid)
	if err != nil {
		log.Println("GetElectionInfo error:", err)
		http.Error(w, "failed to get election info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(info)
}

// ====== /vote/submit (HTTP для браузера) ======

type submitVoteClientRequest struct {
	ElectionID int    `json:"election_id"`
	Choice     string `json:"choice"`
}

type submitVoteClientResponse struct {
	Status string `json:"status"`
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req submitVoteClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.ElectionID <= 0 || req.Choice == "" {
		http.Error(w, "election_id and choice required", http.StatusBadRequest)
		return
	}

	if err := h.service.SubmitVote(req.ElectionID, req.Choice); err != nil {
		log.Println("SubmitVote error:", err)
		http.Error(w, "failed to submit vote: "+err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(submitVoteClientResponse{Status: "ok"})
}
