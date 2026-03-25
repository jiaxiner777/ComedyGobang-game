package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"codebreaker/frontend"
	"codebreaker/internal/game"
)

type Server struct {
	service *game.GameService
}

func NewServer() *Server {
	return &Server{service: game.NewGameService()}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", frontend.StaticHandler()))
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/session", s.handleSessionCreate)
	mux.HandleFunc("/api/session/", s.handleSessionRoutes)
	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := frontend.ReadIndex()
	if err != nil {
		http.Error(w, "index not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) handleSessionCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		Protocol string `json:"protocol"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	session := s.service.Create(req.Protocol)
	writeJSON(w, http.StatusOK, session.Snapshot())
}

func (s *Server) handleSessionRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/session/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	session, ok := s.service.Get(parts[0])
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "session not found"})
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, session.Snapshot())
		return
	}
	if len(parts) < 2 || r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "unsupported route"})
		return
	}

	var err error
	switch parts[1] {
	case "move":
		var req struct {
			Direction string `json:"direction"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		err = session.Move(req.Direction)
	case "choice":
		var req struct {
			Index int `json:"index"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		err = session.Choose(req.Index)
	case "purge":
		var req struct {
			Index int `json:"index"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		err = session.Purge(req.Index)
	case "combat":
		if len(parts) < 3 {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "missing combat action"})
			return
		}
		switch parts[2] {
		case "play":
			var req struct {
				Index int `json:"index"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			err = session.CombatPlay(req.Index)
		case "end":
			err = session.CombatEndTurn()
		case "hijack":
			err = session.CombatHijack()
		default:
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "unknown combat action"})
			return
		}
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "unknown route"})
		return
	}

	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error(), "state": session.Snapshot()})
		return
	}
	writeJSON(w, http.StatusOK, session.Snapshot())
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
