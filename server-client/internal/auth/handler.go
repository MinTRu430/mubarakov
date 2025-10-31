package auth

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"math/big"
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

// Models
type AuthStartRequest struct {
	Login string `json:"login"`
}
type AuthStartResponse struct {
	CodeWord string `json:"code_word"`
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Println("Method not allowed:", r.Method)
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "Only POST allowed")
		return
	}

	var req AuthStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("Invalid JSON in request body:", err)
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	codeWord, err := utils.RandomString(16)
	if err != nil {
		log.Println("codeword randon string generate err:", err)
		utils.WriteJSONError(w, http.StatusBadRequest, "Internal Server Error")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.service.SaveCodeWord(ctx, req.Login, codeWord); err != nil {
		log.Println("failed save codeword err: ", err)
		utils.WriteJSONError(w, http.StatusBadRequest, "Db error"+err.Error())
		return
	}
	hashCodeWord := utils.HashMD5(codeWord)
	json.NewEncoder(w).Encode(AuthStartResponse{CodeWord: hashCodeWord})
}

type AuthFinishRequest struct {
	Login     string `json:"login"`
	MultiHash string `json:"multihash"`
}

type AuthFinishResponse struct {
	A          string `json:"A"`
	G          string `json:"g"`
	P          string `json:"p"`
	H          string `json:"h"`
	W          string `json:"W"`
	RsaPublicE int    `json:"rsa_public_e"`
	RsaPublicN string `json:"rsa_public_n"`
}

func (h *Handler) Finish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Println("Method not allowed:", r.Method)
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "Only POST allowed")
		return
	}

	var req AuthFinishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("Invalid JSON in request body:", err)
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	ok, err := h.service.CheckAuth(ctx, req.Login, req.MultiHash)
	if err != nil {
		log.Println("failed check auth err: ", err)
		utils.WriteJSONError(w, http.StatusBadRequest, "Db error"+err.Error())
		return
	}
	if !ok {
		log.Println("user is not autorized", req.Login)
		utils.WriteJSONError(w, http.StatusBadRequest, "unauthorized")
		return
	}

	p, _ := rand.Prime(rand.Reader, 128)
	g := big.NewInt(5)
	a, _ := rand.Int(rand.Reader, p)
	A := new(big.Int).Exp(g, a, p) // A = g^a mod p

	rsaPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to generate RSA: "+err.Error())
		return
	}
	rsaPub := rsaPriv.PublicKey

	data := []byte(A.String() + g.String() + p.String())
	hash := sha256.Sum256(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaPriv, 0, hash[:])
	if err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to sign data: "+err.Error())
		return
	}

	md5hash := md5.Sum(data)
	wHash := hex.EncodeToString(md5hash[:])

	//добавление диффи в базу

	err = h.service.repo.SaveDiffe(ctx, req.Login, A.String(), a.String(), p.String())
	if err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to added diffe to base: "+err.Error())
		return
	}
	// добавление подписи в базу
	err = h.service.repo.UpdateSign(ctx, req.Login, signature)
	if err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, "failed to updated signature to base: "+err.Error())
		return
	}

	resp := AuthFinishResponse{
		A:          A.String(),
		G:          g.String(),
		P:          p.String(),
		H:          hex.EncodeToString(signature),
		W:          wHash,
		RsaPublicE: rsaPub.E,
		RsaPublicN: rsaPub.N.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
