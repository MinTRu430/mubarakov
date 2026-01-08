package vote

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"server-client/internal/utils"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type CreateElectionRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type CreateElectionResponse struct {
	ID int `json:"id"`
}

type ChangeStatusRequest struct {
	ElectionID int `json:"election_id"`
}

type CountElectionRequest struct {
	ElectionID int `json:"election_id"`
}

type SubmitVoteRequest struct {
	ElectionID int    `json:"election_id"`
	Login      string `json:"login"`
	Ciphertext string `json:"ciphertext"`
}

func (h *Handler) CreateElection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "only POST allowed")
		return
	}

	var req CreateElectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Title == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, "title required")
		return
	}

	id, err := h.service.CreateElection(r.Context(), req.Title, req.Description)
	if err != nil {
		log.Println("CreateElection error:", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to create election")
		return
	}

	json.NewEncoder(w).Encode(CreateElectionResponse{ID: id})
}

func (h *Handler) OpenElection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "only POST allowed")
		return
	}

	var req ChangeStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ElectionID <= 0 {
		utils.WriteJSONError(w, http.StatusBadRequest, "election_id required")
		return
	}

	if err := h.service.OpenElection(r.Context(), req.ElectionID); err != nil {
		log.Println("OpenElection error:", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to open election")
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) CloseElection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "only POST allowed")
		return
	}

	var req ChangeStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ElectionID <= 0 {
		utils.WriteJSONError(w, http.StatusBadRequest, "election_id required")
		return
	}

	if err := h.service.CloseElection(r.Context(), req.ElectionID); err != nil {
		log.Println("CloseElection error:", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to close election")
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) CountElection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "only POST allowed")
		return
	}

	var req CountElectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ElectionID <= 0 {
		utils.WriteJSONError(w, http.StatusBadRequest, "election_id required")
		return
	}

	res, err := h.service.CountVotes(r.Context(), req.ElectionID)
	if err != nil {
		log.Println("CountElection error:", err)
		if err == ErrElectionNotClosed {
			utils.WriteJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to count")
		return
	}

	json.NewEncoder(w).Encode(res)
}

func (h *Handler) ListElections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "only GET allowed")
		return
	}

	es, err := h.service.ListElections(r.Context())
	if err != nil {
		log.Println("ListElections error:", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to list")
		return
	}

	json.NewEncoder(w).Encode(es)
}

func (h *Handler) GetResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "only GET allowed")
		return
	}

	idStr := r.URL.Query().Get("election_id")
	if idStr == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, "election_id required")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid election_id")
		return
	}

	res, err := h.service.GetPublicResult(r.Context(), id)
	if err != nil {
		log.Println("GetResult error:", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to get result")
		return
	}

	json.NewEncoder(w).Encode(res)
}

func (h *Handler) Info(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "only GET allowed")
		return
	}

	idStr := r.URL.Query().Get("election_id")
	if idStr == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, "election_id required")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid election_id")
		return
	}

	info, err := h.service.GetElectionPublicInfo(r.Context(), id)
	if err != nil {
		log.Println("Info error:", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to get info")
		return
	}

	json.NewEncoder(w).Encode(info)
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "only POST allowed")
		return
	}

	var req SubmitVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.ElectionID <= 0 || req.Login == "" || req.Ciphertext == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, "election_id, login, ciphertext required")
		return
	}

	err := h.service.SubmitVote(r.Context(), req.ElectionID, req.Login, req.Ciphertext)
	if err != nil {
		log.Println("SubmitVote error:", err)
		switch err {
		case ErrElectionNotOpen:
			utils.WriteJSONError(w, http.StatusBadRequest, "election is not open")
			return
		case ErrUserNotFound:
			utils.WriteJSONError(w, http.StatusBadRequest, "user not found")
			return
		case ErrAlreadyVoted:
			utils.WriteJSONError(w, http.StatusBadRequest, "user has already voted")
			return
		default:
			utils.WriteJSONError(w, http.StatusInternalServerError, "failed to submit vote")
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
