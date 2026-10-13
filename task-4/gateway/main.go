package main

import (
	"fmt"
	"log"
	"net/http"
)

const serverAddress = ":8080"

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, err := fmt.Fprint(w, "pong"); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", pingHandler)

	log.Printf("Gateway service started on http://localhost%s", serverAddress)

	if err := http.ListenAndServe(serverAddress, mux); err != nil {
		log.Fatalf("gateway server failed: %v", err)
	}
}