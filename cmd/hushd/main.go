package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/adith2005-20/hush/pkg/storage"
)

type Server struct {
	store *storage.Store
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if !s.store.ValidateToken(token) {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (s *Server) handleSetSecret(w http.ResponseWriter, r *http.Request) {
	var secret storage.Secret
	if err := json.NewDecoder(r.Body).Decode(&secret); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.store.UpsertSecret(&secret); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleGetSecrets(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	env := r.URL.Query().Get("environment")

	if project == "" || env == "" {
		http.Error(w, "project and environment required", http.StatusBadRequest)
		return
	}

	secrets, err := s.store.GetSecrets(project, env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(secrets)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(projects)
}

func main() {
	dbPath := os.Getenv("HUSH_DB_PATH")
	if dbPath == "" {
		dbPath = "./hush.db"
	}

	store, err := storage.New(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	server := &Server{store: store}

	http.HandleFunc("/api/secrets", func(w http.ResponseWriter, r *http.Request) {
		handler := server.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				server.handleSetSecret(w, r)
			} else {
				server.handleGetSecrets(w, r)
			}
		})
		handler(w, r)
	})

	http.HandleFunc("/api/projects", server.authMiddleware(server.handleListProjects))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Hush daemon listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
