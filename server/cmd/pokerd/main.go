package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"texaspoker/server/internal/network"
	"texaspoker/server/internal/room"
)

func main() {
	addr := os.Getenv("POKER_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	manager := room.NewManager()
	mux := http.NewServeMux()
	mux.HandleFunc("/events", network.NewEventsHandler(manager))
	mux.HandleFunc("/action", network.NewActionHandler(manager))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "time": room.ServerNow()})
	})

	handler := withCORS(mux)
	log.Printf("poker server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
