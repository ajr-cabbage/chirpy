package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/ajr-cabbage/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
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
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	cfg.fileserverHits.Store(0)
	w.Write([]byte("Metrics reset."))
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

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}

	type validResponse struct {
		CleanedBody string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	newChirp := chirp{}
	err := decoder.Decode(&newChirp)
	if err != nil {
		respondWithError(w, 500, "Error decoding chirp JSON")
		return
	}

	if len(newChirp.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	validResp := validResponse{
		CleanedBody: cleanChirpBody(newChirp.Body),
	}

	respondWithJSON(w, 200, validResp)
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)
	cfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             dbQueries,
	}
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", cfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", readinessHandler)
	mux.HandleFunc("GET /admin/metrics", cfg.hitsHandler)
	mux.HandleFunc("POST /admin/reset", cfg.resetHandler)
	mux.HandleFunc("POST /api/validate_chirp", validateChirpHandler)
	srv := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
