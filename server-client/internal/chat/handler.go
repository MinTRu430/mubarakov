package chat

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"server-client/internal/utils"
	"time"

	"github.com/gorilla/websocket"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type ChatStartRequest struct {
	Login     string `json:"login"`
	B         string `json:"B"`
	Signature string `json:"signature"`
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

	if req.Login == "" || req.B == "" || req.Signature == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, "login, B and signature are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.service.VerifyBSignature(ctx, req.Login, req.B, req.Signature); err != nil {
		log.Println("invalid RSA signature for B:", err)
		utils.WriteJSONError(w, http.StatusUnauthorized, "invalid RSA signature")
		return
	}

	K, err := h.service.ComputeSharedKey(ctx, req.Login, req.B)
	if err != nil {
		log.Println("failed to compute shared key:", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to compute shared key")
		return
	}

	h.service.SetSessionKey(req.Login, K)

	log.Println("Login:", req.Login)
	log.Println("BBB:", req.B)
	log.Println("KKK:", K)

	resp := ChatStartResponse{Message: "DH OK, подпись проверена, теперь подключайся по gRPC и начинаем чат"}
	json.NewEncoder(w).Encode(resp)
}

type ServerSendRequest struct {
	Login   string `json:"login"`
	Message string `json:"message"`
}

type ServerSendResponse struct {
	Status string `json:"status"`
}

func (h *Handler) ServerSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "only POST allowed")
		return
	}

	var req ServerSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Login == "" || req.Message == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, "login and message required")
		return
	}

	if err := h.service.SendToClient(req.Login, req.Message); err != nil {
		log.Println("ServerSend error:", err)
		utils.WriteJSONError(w, http.StatusBadRequest, "cannot send to client: "+err.Error())
		return
	}

	json.NewEncoder(w).Encode(ServerSendResponse{Status: "ok"})
}

type ActiveChatsResponse struct {
	Logins []string `json:"logins"`
}

func (h *Handler) ActiveChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "only GET allowed")
		return
	}

	logins := h.service.ListActiveLogins()
	json.NewEncoder(w).Encode(ActiveChatsResponse{Logins: logins})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Handler) AdminWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws upgrade error:", err)
		return
	}

	wsConn := &WSConn{
		Send: func(b []byte) error {
			return conn.WriteMessage(websocket.TextMessage, b)
		},
	}
	h.service.RegisterWS(wsConn)
	defer func() {
		h.service.UnregisterWS(wsConn)
		conn.Close()
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			log.Println("ws read error:", err)
			return
		}
	}
}

type HistoryResponse struct {
	Messages []ServerMessage `json:"messages"`
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "only GET allowed")
		return
	}

	login := r.URL.Query().Get("login")
	if login == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, "login required")
		return
	}

	msgs, err := h.service.GetHistory(r.Context(), login)
	if err != nil {
		log.Println("GetHistory error:", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to get history")
		return
	}

	json.NewEncoder(w).Encode(HistoryResponse{Messages: msgs})
}
