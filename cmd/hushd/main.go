package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "strings"
    
    "github.com/spf13/cobra"
    "github.com/adith2005-20/hush/pkg/storage"
)

type Server struct {
    store *storage.Store
}

var rootCmd = &cobra.Command{
    Use:   "hushd",
    Short: "Hush secrets server",
}

var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize Hush server (first-time setup)",
    Run: func(cmd *cobra.Command, args []string) {
        dbPath := getDBPath()
        
        // Check if already initialized
        if _, err := os.Stat(dbPath); err == nil {
            fmt.Println("❌ Server already initialized!")
            fmt.Printf("Database exists at: %s\n", dbPath)
            fmt.Println("\nTo reinitialize, delete the database file first:")
            fmt.Printf("  rm %s\n", dbPath)
            os.Exit(1)
        }
        
        fmt.Println("Initializing Hush server...")
        fmt.Println()
        
        store, err := storage.New(dbPath)
        if err != nil {
            log.Fatal("Failed to create database:", err)
        }
        defer store.Close()
        
        token, err := store.CreateAdminToken()
        if err != nil {
            log.Fatal("Failed to create admin token:", err)
        }
        
        fmt.Println("✓ Database created")
        fmt.Println("✓ Admin token generated")
        fmt.Println()
        fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        fmt.Println("🔑 Your admin token (save this securely!):")
        fmt.Println()
        fmt.Printf("   %s\n", token)
        fmt.Println()
        fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        fmt.Println()
        fmt.Println("Next steps:")
        fmt.Println("  1. Start the server:")
        fmt.Println("       hushd start")
        fmt.Println()
        fmt.Println("  2. On your dev machine, login:")
        fmt.Printf("       hush login http://your-server:55555 %s\n", token)
    },
}

var startCmd = &cobra.Command{
    Use:   "start",
    Short: "Start the Hush server",
    Run: func(cmd *cobra.Command, args []string) {
        dbPath := getDBPath()
        
        // Check if initialized
        if _, err := os.Stat(dbPath); os.IsNotExist(err) {
            fmt.Println("❌ Server not initialized!")
            fmt.Println("\nRun this first:")
            fmt.Println("  hushd init")
            os.Exit(1)
        }
        
        store, err := storage.New(dbPath)
        if err != nil {
            log.Fatal(err)
        }
        defer store.Close()

        server := &Server{store: store}
        
        http.HandleFunc("/health", server.handleHealth)
        http.HandleFunc("/api/secrets", server.authMiddleware(server.handleSecrets))

        port := getPort()
        fmt.Printf("🤫 Hush server listening on :%s\n", port)
        fmt.Printf("   Database: %s\n", dbPath)
        log.Fatal(http.ListenAndServe(":"+port, nil))
    },
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
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

func (s *Server) handleSecrets(w http.ResponseWriter, r *http.Request) {
    if r.Method == "POST" {
        s.handleSetSecret(w, r)
    } else {
        s.handleGetSecrets(w, r)
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

func getDBPath() string {
    if path := os.Getenv("HUSH_DB_PATH"); path != "" {
        return path
    }
    return "./hush.db"
}

func getPort() string {
    if port := os.Getenv("PORT"); port != "" {
        return port
    }
    return "55555"
}

func main() {
    rootCmd.AddCommand(initCmd)
    rootCmd.AddCommand(startCmd)
    
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}