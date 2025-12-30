package storage

import (
    "database/sql"
    "github.com/google/uuid"
    _ "modernc.org/sqlite"
)

type Store struct {
    db *sql.DB
}

type Secret struct {
    ID          int
    Project     string
    Environment string
    Key         string
    Value       string
    CreatedAt   string
    UpdatedAt   string
}

type Token struct {
    ID        int
    Token     string
    Name      string
    CreatedAt string
}

func New(dbPath string) (*Store, error) {
    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        return nil, err
    }

    store := &Store{db: db}
    if err := store.init(); err != nil {
        return nil, err
    }

    return store, nil
}

func (s *Store) init() error {
    schema := `
    CREATE TABLE IF NOT EXISTS secrets (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        project TEXT NOT NULL,
        environment TEXT NOT NULL,
        key TEXT NOT NULL,
        value TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(project, environment, key)
    );

    CREATE INDEX IF NOT EXISTS idx_project_env ON secrets(project, environment);

    CREATE TABLE IF NOT EXISTS tokens (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        token TEXT UNIQUE NOT NULL,
        name TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    `

    _, err := s.db.Exec(schema)
    return err
}

func (s *Store) CreateAdminToken() (string, error) {
    token := "hush_" + uuid.New().String()
    _, err := s.db.Exec("INSERT INTO tokens (token, name) VALUES (?, ?)", token, "admin")
    return token, err
}

func (s *Store) UpsertSecret(secret *Secret) error {
    query := `
    INSERT INTO secrets (project, environment, key, value, updated_at)
    VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
    ON CONFLICT(project, environment, key) 
    DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
    `
    _, err := s.db.Exec(query, secret.Project, secret.Environment, secret.Key, secret.Value)
    return err
}

func (s *Store) GetSecrets(project, environment string) ([]Secret, error) {
    query := `SELECT id, project, environment, key, value, created_at, updated_at 
              FROM secrets WHERE project = ? AND environment = ?`
    
    rows, err := s.db.Query(query, project, environment)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var secrets []Secret
    for rows.Next() {
        var s Secret
        err := rows.Scan(&s.ID, &s.Project, &s.Environment, &s.Key, &s.Value, &s.CreatedAt, &s.UpdatedAt)
        if err != nil {
            return nil, err
        }
        secrets = append(secrets, s)
    }

    return secrets, nil
}

func (s *Store) ValidateToken(token string) bool {
    var count int
    err := s.db.QueryRow("SELECT COUNT(*) FROM tokens WHERE token = ?", token).Scan(&count)
    return err == nil && count > 0
}

func (s *Store) Close() error {
    return s.db.Close()
}