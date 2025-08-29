package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// DatabaseInterface defines the database operations
type DatabaseInterface interface {
	CreateBuild(build *BuildRequest) (int, error)
	GetBuild(id int) (*BuildRequest, error)
	ListBuilds() ([]*BuildRequest, error)
	UpdateBuildStatus(id int, status string) error
	Ping() error
	Close() error
	InitTables() error
}

// PostgreSQLDatabase implements DatabaseInterface
type PostgreSQLDatabase struct {
	db *sql.DB
}

// NewPostgreSQLDatabase creates a new PostgreSQL database connection
func NewPostgreSQLDatabase() (*PostgreSQLDatabase, error) {
	// Get database connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Default connection for local development
		dbURL = "postgres://postgres:password@localhost:5432/buildservice?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgreSQLDatabase{db: db}, nil
}

// InitTables creates the necessary database tables
func (pg *PostgreSQLDatabase) InitTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS builds (
		id SERIAL PRIMARY KEY,
		project_name VARCHAR(255) NOT NULL,
		git_url VARCHAR(500) NOT NULL,
		branch VARCHAR(100) NOT NULL DEFAULT 'main',
		status VARCHAR(50) NOT NULL DEFAULT 'queued',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_builds_status ON builds(status);
	CREATE INDEX IF NOT EXISTS idx_builds_project ON builds(project_name);
	CREATE INDEX IF NOT EXISTS idx_builds_created_at ON builds(created_at);
	`

	_, err := pg.db.Exec(query)
	return err
}

// CreateBuild creates a new build record
func (pg *PostgreSQLDatabase) CreateBuild(build *BuildRequest) (int, error) {
	query := `
	INSERT INTO builds (project_name, git_url, branch, status, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id
	`

	var id int
	err := pg.db.QueryRow(
		query,
		build.ProjectName,
		build.GitURL,
		build.Branch,
		build.Status,
		build.CreatedAt,
		build.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetBuild retrieves a build by ID
func (pg *PostgreSQLDatabase) GetBuild(id int) (*BuildRequest, error) {
	query := `
	SELECT id, project_name, git_url, branch, status, created_at, updated_at
	FROM builds
	WHERE id = $1
	`

	build := &BuildRequest{}
	err := pg.db.QueryRow(query, id).Scan(
		&build.ID,
		&build.ProjectName,
		&build.GitURL,
		&build.Branch,
		&build.Status,
		&build.CreatedAt,
		&build.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("build not found")
	}

	return build, err
}

// ListBuilds retrieves all builds
func (pg *PostgreSQLDatabase) ListBuilds() ([]*BuildRequest, error) {
	query := `
	SELECT id, project_name, git_url, branch, status, created_at, updated_at
	FROM builds
	ORDER BY created_at DESC
	LIMIT 100
	`

	rows, err := pg.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var builds []*BuildRequest
	for rows.Next() {
		build := &BuildRequest{}
		err := rows.Scan(
			&build.ID,
			&build.ProjectName,
			&build.GitURL,
			&build.Branch,
			&build.Status,
			&build.CreatedAt,
			&build.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		builds = append(builds, build)
	}

	return builds, rows.Err()
}

// UpdateBuildStatus updates the status of a build
func (pg *PostgreSQLDatabase) UpdateBuildStatus(id int, status string) error {
	query := `
	UPDATE builds
	SET status = $1, updated_at = NOW()
	WHERE id = $2
	`

	_, err := pg.db.Exec(query, status, id)
	return err
}

// Ping checks if the database connection is alive
func (pg *PostgreSQLDatabase) Ping() error {
	return pg.db.Ping()
}

// Close closes the database connection
func (pg *PostgreSQLDatabase) Close() error {
	return pg.db.Close()
}
