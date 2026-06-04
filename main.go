package main

import (
	"log"
	"net/http"
)

func readinessHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
	mux.HandleFunc("/healthz", readinessHandler)
	srv := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
