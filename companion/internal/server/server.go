package server

import (
	"encoding/json"
	"net/http"

	"xeneoncc/internal/protocol"
	"xeneoncc/internal/store"
)

type Server struct {
	store *store.Store
	token string
}

func New(st *store.Store, token string) http.Handler {
	s := &Server{store: st, token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/usage", s.auth(s.postUsage))
	mux.HandleFunc("/v1/notify", s.auth(s.postNotify))
	mux.HandleFunc("/v1/state", s.auth(s.getState))
	return mux
}

func (s *Server) auth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "X-Bridge-Token, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Header.Get("X-Bridge-Token") != s.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		h(w, r)
	}
}

func (s *Server) postUsage(w http.ResponseWriter, r *http.Request) {
	var u protocol.Usage
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	s.store.SetUsage(u)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) postNotify(w http.ResponseWriter, r *http.Request) {
	var n protocol.Notification
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	s.store.AddNotification(n)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(s.store.Snapshot())
}
