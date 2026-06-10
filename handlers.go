package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/ajr-cabbage/chirpy/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errMsg struct {
		Error string `json:"error"`
	}

	respBody := errMsg{
		Error: msg,
	}

	errDat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error Marshaling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(errDat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error Marshaling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) hitsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(200)
	body := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())
	w.Write([]byte(body))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, 403, "403 FORBIDDEN")
		return
	}

	cfg.fileserverHits.Store(0)
	err := cfg.db.ResetUsers(r.Context())
	if err != nil {
		respondWithError(w, 400, "Error resetting users")
		return
	}

	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("DB reset."))
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {
	type userInfo struct {
		Email string `json:"email"`
	}

	type newUserResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	newUserInfo := userInfo{}
	err := decoder.Decode(&newUserInfo)
	if err != nil {
		respondWithError(w, 500, "Error decoding new user JSON")
		return
	}

	newUser, err := cfg.db.CreateUser(r.Context(), newUserInfo.Email)
	if err != nil {
		respondWithError(w, 400, "Error adding user to db")
		return
	}

	newUserResp := newUserResponse{
		ID:        newUser.ID,
		CreatedAt: newUser.CreatedAt.Time,
		UpdatedAt: newUser.UpdatedAt.Time,
		Email:     newUser.Email,
	}

	respondWithJSON(w, 201, newUserResp)
}

func cleanChirpBody(chirp string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(chirp, " ")

	for i, word := range words {
		if slices.Contains(badWords, strings.ToLower(word)) {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}

func (cfg *apiConfig) createChirpHandler(w http.ResponseWriter, r *http.Request) {
	type chirpInfo struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	type validChirpResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	newChirpInfo := chirpInfo{}
	err := decoder.Decode(&newChirpInfo)
	if err != nil {
		respondWithError(w, 500, "Error decoding new chirp JSON")
		return
	}

	if len(newChirpInfo.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	newChirpParams := database.CreateChirpParams{
		Body:   cleanChirpBody(newChirpInfo.Body),
		UserID: newChirpInfo.UserID,
	}

	newChirp, err := cfg.db.CreateChirp(r.Context(), newChirpParams)
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}

	chirpResp := validChirpResponse{
		ID:        newChirp.ID,
		CreatedAt: newChirp.CreatedAt.Time,
		UpdatedAt: newChirp.UpdatedAt.Time,
		Body:      newChirp.Body,
		UserID:    newChirp.UserID,
	}

	respondWithJSON(w, 201, chirpResp)
}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	type chirpResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	allChirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}

	allChirpsResponse := []chirpResponse{}
	for _, chirp := range allChirps {
		allChirpsResponse = append(allChirpsResponse, chirpResponse{ID: chirp.ID, CreatedAt: chirp.CreatedAt.Time, UpdatedAt: chirp.UpdatedAt.Time, Body: chirp.Body, UserID: chirp.UserID})
	}

	respondWithJSON(w, 200, allChirpsResponse)

}

func (cfg *apiConfig) getChirpByID(w http.ResponseWriter, r *http.Request) {
	type chirpResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	reqUUID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 500, err.Error())
	}

	chirp, err := cfg.db.GetChirpByID(r.Context(), reqUUID)
	if err != nil {
		respondWithError(w, 404, err.Error())
	}

	chirpResp := chirpResponse{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt.Time,
		UpdatedAt: chirp.UpdatedAt.Time,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}

	respondWithJSON(w, 200, chirpResp)

}
