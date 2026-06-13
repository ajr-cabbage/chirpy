package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/ajr-cabbage/chirpy/internal/auth"
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
		Email    string `json:"email"`
		Password string `json:"password"`
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
		respondWithError(w, 500, err.Error())
		return
	}

	hashedPW, err := auth.HashPassword(newUserInfo.Password)
	if err != nil {
		respondWithError(w, 500, err.Error())
	}

	userParams := database.CreateUserParams{
		Email:          newUserInfo.Email,
		HashedPassword: hashedPW,
	}

	newUser, err := cfg.db.CreateUser(r.Context(), userParams)
	if err != nil {
		respondWithError(w, 400, err.Error())
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

	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	userID, err := auth.ValidateJWT(tokenString, cfg.secret)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}

	decoder := json.NewDecoder(r.Body)
	newChirpInfo := chirpInfo{}
	err = decoder.Decode(&newChirpInfo)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}

	if len(newChirpInfo.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	newChirpParams := database.CreateChirpParams{
		Body:   cleanChirpBody(newChirpInfo.Body),
		UserID: userID,
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
		return
	}

	chirp, err := cfg.db.GetChirpByID(r.Context(), reqUUID)
	if err != nil {
		respondWithError(w, 404, err.Error())
		return
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

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	type loginInfo struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	type validUserResponse struct {
		ID           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		Token        string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}

	decoder := json.NewDecoder(r.Body)

	newLoginInfo := loginInfo{}
	err := decoder.Decode(&newLoginInfo)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}

	validUser, err := cfg.db.GetUserByEmail(r.Context(), newLoginInfo.Email)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}

	authenticated, err := auth.CheckPasswordHash(newLoginInfo.Password, validUser.HashedPassword)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}

	if !authenticated {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	token, err := auth.MakeJWT(validUser.ID, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}

	refreshTokParams := database.CreateRefreshTokenParams{
		Token:     auth.MakeRefreshToken(),
		UserID:    validUser.ID,
		ExpiresAt: time.Now().Add(1440 * time.Hour),
	}

	newRefreshTok, err := cfg.db.CreateRefreshToken(r.Context(), refreshTokParams)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}

	validResp := validUserResponse{
		ID:           validUser.ID,
		CreatedAt:    validUser.CreatedAt.Time,
		UpdatedAt:    validUser.UpdatedAt.Time,
		Email:        validUser.Email,
		Token:        token,
		RefreshToken: newRefreshTok.Token,
	}

	respondWithJSON(w, 200, validResp)
}

func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, r *http.Request) {
	type validResponse struct {
		Token string `json:"token"`
	}

	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}

	refreshToken, err := cfg.db.GetUserFromRefreshToken(r.Context(), tokenString)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}

	if refreshToken.ExpiresAt.Before(time.Now()) || refreshToken.RevokedAt.Valid == true {
		w.WriteHeader(401)
		return
	}

	respAccessTok, err := auth.MakeJWT(refreshToken.UserID, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}

	validResp := validResponse{
		Token: respAccessTok,
	}

	respondWithJSON(w, 200, validResp)
}

func (cfg *apiConfig) revokeHandler(w http.ResponseWriter, r *http.Request) {
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}

	err = cfg.db.RevokeRefreshToken(r.Context(), tokenString)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}

	w.WriteHeader(204)
}

func (cfg *apiConfig) udpateUserHandler(w http.ResponseWriter, r *http.Request) {
	type validResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}

	type updateRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}

	userID, err := auth.ValidateJWT(tokenString, cfg.secret)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}

	updateReq := updateRequest{}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&updateReq)
	if err != nil {
		respondWithError(w, 501, err.Error())
		return
	}

	hashPW, err := auth.HashPassword(updateReq.Password)

	usrInfoParams := database.UpdateUserInfoParams{
		ID:             userID,
		Email:          updateReq.Email,
		HashedPassword: hashPW,
	}

	usr, err := cfg.db.UpdateUserInfo(r.Context(), usrInfoParams)
	if err != nil {
		respondWithError(w, 501, err.Error())
	}

	usrResponse := validResponse{
		ID:        usr.ID,
		CreatedAt: usr.CreatedAt.Time,
		UpdatedAt: usr.UpdatedAt.Time,
		Email:     usr.Email,
	}

	respondWithJSON(w, 200, usrResponse)
}

func (cfg *apiConfig) deleteChirpHandler(w http.ResponseWriter, r *http.Request) {
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}

	userID, err := auth.ValidateJWT(tokenString, cfg.secret)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}

	chirpUUID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}

	chirpData, err := cfg.db.GetChirpByID(r.Context(), chirpUUID)
	if err != nil {
		respondWithError(w, 404, err.Error())
		return
	}

	if userID != chirpData.UserID {
		w.WriteHeader(403)
		return
	}

	err = cfg.db.DeleteChirp(r.Context(), chirpUUID)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}

	w.WriteHeader(204)
}
